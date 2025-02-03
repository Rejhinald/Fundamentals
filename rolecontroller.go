package controllers

import (
	"errors"
	"grooper/app"
	"grooper/app/constants"
	"grooper/app/mail"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	// uuid "github.com/satori/go.uuid"
)

type RoleController struct {
	*revel.Controller
}

/*****************
CreateRole()
Creates a role with selected permissions
Body:
company_id - required
role_permission[] - required
role_name - required
*****************/

func (c RoleController) CreateRole() revel.Result {
	var rolePermission []string
	var userIDs []string

	c.Params.Bind(&rolePermission, "role_permission")
	c.Params.Bind(&userIDs, "user_id")

	roleId := utils.GenerateTimestampWithUID()

	companyId := c.Params.Form.Get("company_id")

	createdBy := utils.TrimSpaces(c.Params.Form.Get("created_by"))

	roleName := utils.TrimSpaces(c.Params.Form.Get("role_name"))

	var inputRequest []*dynamodb.WriteRequest
	var input *dynamodb.BatchWriteItemInput
	tableName := app.TABLE_NAME
	regExp := regexp.MustCompile("^[a-zA-Z0-9 ]*$")

	//Get current timestamp
	currentTime := utils.GetCurrentTimestamp()

	var recipients []mail.Recipient

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	role := models.Role{
		PK:              utils.AppendPrefix(constants.PREFIX_ROLE, roleId),
		SK:              utils.AppendPrefix(constants.PREFIX_COMPANY, companyId),
		RoleID:          roleId,
		CompanyID:       companyId,
		RolePermissions: rolePermission,
		RoleName:        roleName,
		CreatedBy:       createdBy,
		CreatedAt:       currentTime,
		Type:            constants.ENTITY_TYPE_ROLE,
	}

	role.Validate(c.Validation, constants.SERVICE_TYPE_CREATE_ROLE)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	//Checking if RoleName premade in system
	newRoleName := strings.ToLower(roleName)
	isRoleNamePreMade := false
	if newRoleName == "company admin" || newRoleName == "department admin" || newRoleName == "group admin" || newRoleName == "group member" {
		dataRole := true
		isRoleNamePreMade = dataRole
	}

	result := IsRoleNameUnique(strings.ToLower(role.RoleName), role.CompanyID)
	if (result) || (isRoleNamePreMade) {
		data["errors"] = "The role name already exists."
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	if !regExp.MatchString(role.RoleName) {
		data["errors"] = "Special characters are not allowed"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	rolePermissions, err := dynamodbattribute.MarshalList(role.RolePermissions)
	if err != nil {
		data["errors"] = "Unable to marshal list"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}
	inputRequest = append(inputRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
		Item: map[string]*dynamodb.AttributeValue{
			"PK": &dynamodb.AttributeValue{
				S: aws.String(role.PK),
			},
			"SK": &dynamodb.AttributeValue{
				S: aws.String(role.SK),
			},
			"RoleID": &dynamodb.AttributeValue{
				S: aws.String(role.RoleID),
			},
			"CompanyID": &dynamodb.AttributeValue{
				S: aws.String(role.CompanyID),
			},
			"RolePermissions": &dynamodb.AttributeValue{
				L: rolePermissions,
			},
			"RoleName": &dynamodb.AttributeValue{
				S: aws.String(role.RoleName),
			},
			"SearchKey": &dynamodb.AttributeValue{
				S: aws.String(strings.ToLower(role.RoleName)),
			},
			"Type": &dynamodb.AttributeValue{
				S: aws.String(role.Type),
			},
			"CreatedBy": &dynamodb.AttributeValue{
				S: aws.String(role.CreatedBy),
			},
			"CreatedAt": &dynamodb.AttributeValue{
				S: aws.String(role.CreatedAt),
			},
		},
	}})

	input = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			tableName: inputRequest,
		},
	}

	_, roleErr := app.SVC.BatchWriteItem(input)
	if roleErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		data["error"] = roleErr.Error()
		return c.RenderJSON(data)
	}

	if len(userIDs) != 0 {
		for _, userID := range userIDs {
			result, err := ops.CheckUserRole(userID, role.RoleID)
			if err != nil {
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(data)
			}

			if !result {
				item := models.UserRole{
					PK: utils.AppendPrefix(constants.PREFIX_USER, userID),
					// SK:        utils.AppendPrefix(constants.PREFIX_ROLE, role.RoleID),
					SK:        utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, role.RoleID), utils.AppendPrefix(constants.PREFIX_COMPANY, role.CompanyID)),
					UserID:    userID,
					RoleID:    role.RoleID,
					CompanyID: role.CompanyID,
					Type:      constants.ENTITY_TYPE_USER_ROLE,
				}

				av, err := dynamodbattribute.MarshalMap(item)
				if err != nil {
					data["error"] = "Error at marshalmap"
					data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
					return c.RenderJSON(data)
				}

				input := &dynamodb.PutItemInput{
					Item:      av,
					TableName: aws.String(app.TABLE_NAME),
				}

				_, err = app.SVC.PutItem(input)
				if err != nil {
					data["error"] = "Cannot assign role due to server error"
					data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
					return c.RenderJSON(data)
				}
				user, opsErr := ops.GetUserByIDNew(userID)
				if opsErr != nil {
					return c.RenderJSON(opsErr)
				}
				recipients = append(recipients, mail.Recipient{
					Name:           user.FirstName + " " + user.LastName,
					Email:          user.Email,
					ActionType:     "assigned",
					RoleName:       role.RoleName,
					RolePermission: role.RolePermissions,
				})

			}
		}
	}
	jobs.Now(mail.SendEmail{
		Subject:    "You have been assigned to a role",
		Recipients: recipients,
		Template:   "change_permissions.html",
	})
	// generate log
	var logs = []*models.Logs{}
	// message: UserX has created RoleNameX
	logs = append(logs, &models.Logs{
		CompanyID: companyId,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_ROLE,
		LogType:   constants.ENTITY_TYPE_ROLE,
		LogInfo: &models.LogInformation{
			Role: &models.LogModuleParams{
				ID:   roleId,
				Name: roleName,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		data["logs"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	data["role"] = role
	return c.RenderJSON(data)

}

/*
****************
AssignRole()
Assign multiple roles to multple users
****************
*/
func (c RoleController) AssignRole() revel.Result {
	var roleIDs []string
	var userIDs []string
	c.Params.Bind(&roleIDs, "role_id")
	c.Params.Bind(&userIDs, "user_id")
	companyID := c.ViewArgs["companyID"].(string)

	data := make(map[string]interface{})

	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		return c.RenderJSON(opsError)
	}
	// var usersToInvite []models.User
	var recipients []mail.Recipient

	for _, userID := range userIDs {
		user, opsErr := ops.GetUserByIDNew(userID)
		if opsErr != nil {
			return c.RenderJSON(opsErr)
		}
		for _, roleID := range roleIDs {
			role, opsErr := ops.GetRoleByID(roleID, companyID)
			if opsErr != nil {
				return c.RenderJSON(opsErr)
			}
			// result, err := ops.CheckUserRole(userID, roleID)
			// if err != nil {
			// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			// 	return c.RenderJSON(data)
			// }

			//
			//
			//
			//
			//

			// if !result {
			item := models.UserRole{
				PK:        utils.AppendPrefix(constants.PREFIX_USER, userID),
				SK:        utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, roleID), utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
				UserID:    userID,
				RoleID:    roleID,
				CompanyID: companyID,
				Type:      constants.ENTITY_TYPE_USER_ROLE,
			}

			av, err := dynamodbattribute.MarshalMap(item)
			if err != nil {
				data["error"] = "Error at marshalmap"
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(data)
			}

			input := &dynamodb.PutItemInput{
				Item:      av,
				TableName: aws.String(app.TABLE_NAME),
			}

			//
			//
			//
			//
			//

			_, err = app.SVC.PutItem(input)
			if err != nil {
				data["error"] = "Cannot assign role due to server error"
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(data)
			}

			// // create account if the user doesn't have
			// user, err := ops.GetUserByID(userID)
			// if err == nil {
			// 	// return c.RenderJSON(user)
			// 	if user.UserToken == "DEFAULT_USER" || user.Password == "" {

			// 		userToken := utils.GenerateRandomString(8)
			// 		user.UserToken = userToken

			// 		updateInput := &dynamodb.UpdateItemInput{
			// 			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			// 				":us": {
			// 					S: aws.String(userToken),
			// 				},
			// 				":s": {
			// 					S: aws.String(constants.ITEM_STATUS_PENDING),
			// 				},
			// 				":ac": {
			// 					S: aws.String(companyID),
			// 				},
			// 			},
			// 			ExpressionAttributeNames: map[string]*string{
			// 				"#s": aws.String("Status"),
			// 			},
			// 			TableName: aws.String(app.TABLE_NAME),
			// 			Key: map[string]*dynamodb.AttributeValue{
			// 				"PK": {
			// 					S: aws.String(user.PK),
			// 				},
			// 				"SK": {
			// 					S: aws.String(user.SK),
			// 				},
			// 			},
			// 			UpdateExpression: aws.String("SET UserToken = :us, #s = :s, ActiveCompany = :ac"),
			// 		}
			// 		// update
			// 		updateItemResult, err := app.SVC.UpdateItem(updateInput)
			// 		if err == nil {
			// 			// data["message"] = err.Error()
			// 			// data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			// 			// return c.RenderJSON(data)
			// 			if !utils.StringInSlice(userID, usersToInvite) {
			// 				usersToInvite = append(usersToInvite, user)
			// 			}
			// 		}
			// 		_ = updateItemResult
			// 	}
			// }

			recipients = append(recipients, mail.Recipient{
				Name:           user.FirstName + " " + user.LastName,
				Email:          user.Email,
				ActionType:     "assigned",
				RoleName:       role.RoleName,
				RolePermission: role.RolePermissions,
				CompanyName:    company.CompanyName,
			})
		}
	}
	jobs.Now(mail.SendEmail{
		Subject:    "[SaaSConsole] Your access to " + company.CompanyName + " has changed",
		Recipients: recipients,
		Template:   "change_permissions.html",
	})
	// if len(usersToInvite) != 0 {
	// 	err := SendCompanyInvitation(companyID, usersToInvite)
	// 	if err != nil { }
	// }

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
UnassignRole()
Unassign multiple roles to multple users
****************
*/
func (c RoleController) UnassignRole() revel.Result {
	var roleIDs []string
	var userIDs []string
	c.Params.Bind(&roleIDs, "role_id")
	c.Params.Bind(&userIDs, "user_id")
	companyID := c.ViewArgs["companyID"].(string)

	data := make(map[string]interface{})
	var recipients []mail.Recipient

	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		return c.RenderJSON(opsError)
	}

	for _, userID := range userIDs {
		user, opsErr := ops.GetUserByIDNew(userID)
		if opsErr != nil {
			return c.RenderJSON(opsErr)
		}
		for _, roleID := range roleIDs {
			role, opsErr := ops.GetRoleByID(roleID, companyID)
			if opsErr != nil {
				return c.RenderJSON(opsErr)
			}

			// result, err := ops.CheckUserRole(userID, roleID)
			// if err != nil {
			// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			// 	return c.RenderJSON(data)
			// }

			// if result {

			userrole := &dynamodb.DeleteItemInput{
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, roleID), utils.AppendPrefix(constants.PREFIX_COMPANY, companyID))),
					},
				},
				TableName: aws.String(app.TABLE_NAME),
			}

			_, err := app.SVC.DeleteItem(userrole)
			if err != nil {
				data["message"] = "Got error calling DeleteItem at userrole"
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(data)
			}
			recipients = append(recipients, mail.Recipient{
				Name:           user.FirstName + " " + user.LastName,
				Email:          user.Email,
				ActionType:     "unassigned",
				RoleName:       role.RoleName,
				RolePermission: role.RolePermissions,
				CompanyName:    company.CompanyName,
			})
			// }
		}
	}

	jobs.Now(mail.SendEmail{
		Subject:    "[SaaSConsole] Your access to " + company.CompanyName + " has changed",
		Recipients: recipients,
		Template:   "change_permissions.html",
	})
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
UpdateRole()
Update a roles permission
Body:
role_name - required
role_id - required
role_permission - required
company_id - required
****************
*/
func (c RoleController) UpdateRole() revel.Result {
	var rolePermission []string
	c.Params.Bind(&rolePermission, "role_permission")
	roleId := c.Params.Form.Get("role_id")
	roleName := utils.TrimSpaces(c.Params.Form.Get("role_name"))
	companyId := c.Params.Form.Get("company_id")

	//Get current timestamp
	currentTime := utils.GetCurrentTimestamp()

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	if len(roleName) == 0 {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	rolePermissions, err := dynamodbattribute.MarshalList(rolePermission)
	if err != nil {
		data["errors"] = "Unable to marshal list"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	//Comparing roleNamefromId and roleName from input
	role, opsError := ops.GetRoleByID(roleId, companyId)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(opsError.Status.Code)
		return c.RenderJSON(data)
	}

	//roleNameFromId := role.RoleName
	//compareRolenames := strings.EqualFold(roleName, roleNameFromId)

	//Checking if RoleName premade in system
	isRoleNamePreMade := false
	preMadeRoles := constants.PRE_MADE_ROLES
	for i := 0; i < len(preMadeRoles)-1; i++ {
		if preMadeRoles[i] == strings.ToLower(roleName) {
			isRoleNamePreMade = true
			break
		}
	}

	//Checking if roleName is unique
	isUnique := IsRoleNameUnique(strings.ToLower(roleName), companyId)
	if !isUnique && isRoleNamePreMade {
		data["errors"] = "The role name already exists."
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pc": {
				L: rolePermissions,
			},
			":ua": {
				S: aws.String(currentTime),
			},
			":rn": {
				S: aws.String(roleName),
			},
			":key": {
				S: aws.String(strings.ToLower(roleName)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#r":   aws.String("RoleName"),
			"#rp":  aws.String("RolePermissions"),
			"#key": aws.String("SearchKey"),
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_ROLE, roleId)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyId)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET #r = :rn, #rp = :pc, #key = :key, UpdatedAt = :ua"),
	}
	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		data["error"] = err.Error()
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs = []*models.Logs{}
	// message: PermissionsX has been added to RoleNameX
	var logInfoPermissions []models.LogModuleParams
	for _, permission := range rolePermission {
		logInfoPermissions = append(logInfoPermissions, models.LogModuleParams{
			ID: permission,
		})
	}

	logs = append(logs, &models.Logs{
		CompanyID: companyId,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_ROLE,
		LogType:   constants.ENTITY_TYPE_ROLE,
		LogInfo: &models.LogInformation{
			Role: &models.LogModuleParams{
				ID:   roleId,
				Name: role.RoleName,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
			Permissions: logInfoPermissions,
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		data["logs"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
GetAllRoles()
Get all roles
****************
*/
func (c RoleController) GetAllRoles() revel.Result {
	companyId := c.Params.Query.Get("company_id")
	userCompanyID := c.ViewArgs["companyID"].(string)
	key := c.Params.Query.Get("key")
	data := make(map[string]interface{})
	roles := []models.Role{}

	result, err := ops.GetAllByGSI(
		constants.INDEX_NAME_GET_ROLES,
		constants.ENTITY_TYPE_ROLE,
		constants.PARAMS_SK,
		constants.PREFIX_COMPANY,
		constants.PARAMS_COMPANY_ID,
		companyId,
		constants.PARAMS_COMPANY_ID,
		constants.PARAMS_SYSTEM,
		strings.ToLower(key),
	)

	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result, &roles)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	for i, v := range roles {

		// num := TotalUsersToRole(v.RoleID, userCompanyID)
		// roles[i].TotalInUse = num

		num := 0

		userrole := GetAllUserID(v.RoleID, userCompanyID)

		for _, v := range userrole {
			userCompany, err := GetCompanyUser(userCompanyID, v.UserID)
			if err == nil && (userCompany.Status != constants.ITEM_STATUS_DELETED && userCompany.Status != constants.ITEM_STATUS_INACTIVE) {
				user, err := ops.GetUserByID(v.UserID)
				if err == nil {
					user.Status = userCompany.Status
					roles[i].Users = append(roles[i].Users, user)
					num = num + 1
				}
			}
		}

		// num := TotalUsersToRole(v.RoleID, userCompanyID)
		roles[i].TotalInUse = num
	}

	data["roles"] = roles
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
Get Role By ID
Params:
roleId - required
****************
*/
func (c RoleController) GetRole() revel.Result {
	roleId := c.Params.Query.Get("roleId")
	data := make(map[string]interface{})

	role, opsError := ops.GetRoleByID(roleId, c.ViewArgs["companyID"].(string))
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(opsError.Status.Code)
		return c.RenderJSON(data)
	}

	data["role"] = role

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)

}

/*****************
Delete Role By Role ID and Company ID
Params:
role_id - required
company_id - required
*****************/

func (c RoleController) DeleteRole() revel.Result {
	var roleIDs []string
	c.Params.Bind(&roleIDs, "role_id")

	companyID := c.Params.Form.Get("company_id")
	data := make(map[string]interface{})

	// check if role id exists
	for _, roleID := range roleIDs {
		roleDetails, opsError := ops.GetRoleByID(roleID, c.ViewArgs["companyID"].(string))
		if opsError != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(data)
		}

		result := GetAllUserID(roleID, companyID)
		if result == nil {
			data["message"] = "error in helpers"
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}

		if len(result) != 0 {
			for _, items := range result {
				userrole := &dynamodb.DeleteItemInput{
					Key: map[string]*dynamodb.AttributeValue{
						"PK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, items.UserID)),
						},
						"SK": {
							S: aws.String(utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, roleID), utils.AppendPrefix(constants.PREFIX_COMPANY, companyID))),
						},
					},
					TableName: aws.String(app.TABLE_NAME),
				}

				_, err := app.SVC.DeleteItem(userrole)
				if err != nil {
					data["message"] = "Got error calling DeleteItem at userrole"
					data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
					return c.RenderJSON(data)
				}
			}
		}

		role := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_ROLE, roleID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err := app.SVC.DeleteItem(role)
		if err != nil {
			data["message"] = "Got error calling DeleteItem at role"
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}

		// generate log
		var logs = []*models.Logs{}
		// message: UserX has deleted RoleNameX
		logs = append(logs, &models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_DELETE_ROLE,
			LogType:   constants.ENTITY_TYPE_ROLE,
			LogInfo: &models.LogInformation{
				Role: &models.LogModuleParams{
					ID:   roleID,
					Name: roleDetails.RoleName,
				},
				User: &models.LogModuleParams{
					ID: c.ViewArgs["userID"].(string),
				},
			},
		})
		_, err = CreateBatchLog(logs)
		if err != nil {
			data["log"] = "error while creating logs"
		}
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(data)

}

/**
*
*NUMBER OF USERS FOR THE ROLE
*
 */
func (c RoleController) RoleInUsed() revel.Result {
	roleID := c.Params.Query.Get("role_id")
	companyID := c.Params.Query.Get("company_id")
	data := make(map[string]interface{})

	numberOfRoles := len(GetAllUserID(roleID, companyID))

	data["data"] = numberOfRoles
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

// get total users in role
func TotalUsersToRole(roleID, companyID string) int {
	numberOfRoles := 0

	numberOfRoles = len(GetAllUserID(roleID, companyID))

	return numberOfRoles
}

func IsRoleNameUnique(roleName, companyId string) bool {

	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String("ROLE"),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyId),
					},
				},
			},
			"SearchKey": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(roleName),
					},
				},
			},
		},

		IndexName: aws.String(constants.INDEX_NAME_GET_ROLES),
		TableName: aws.String(app.TABLE_NAME),
	}

	res, err := app.SVC.Query(params)
	if err != nil {
		return false
	}

	if len(res.Items) > 0 {
		return true
	} else {
		return false
	}
}

func GetAllUserID(roleID, companyID string) []models.UserRole {
	userrole := []models.UserRole{}

	result, err := ops.GetIdsFromRoleID(roleID, companyID)
	if err != nil {
		//
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result, &userrole)
	if err != nil {
		//

	}

	return userrole
}

/*
****************
CreateGroupLog
- Helper function for creating logs under group
****************
*/
func (c RoleController) CreateGroupLog(companyID string, details *models.LogModuleParams, logAction string) (string, error) {
	authorID := c.ViewArgs["userID"].(string)
	logCtrl := LogController{}
	logInfo := &models.LogInformation{
		// Action:      logAction,
		Role:        details,
		PerformedBy: authorID,
	}
	logID, err := logCtrl.CreateLog(authorID, companyID, logAction, constants.ENTITY_TYPE_GROUP, logInfo)
	if err != nil {
		e := errors.New(err.Error())
		return "", e
	}
	return logID, nil
}

type RequestRolesParams struct {
	Roles  []string `json:"roles,omitempty"`
	UserID string   `json:"user_id,omitempty"`
}

func sendNotificationToCompanyAdmin(c RoleController, companyAdminUserID string, requesterUserInfo models.CompanyUser, requestedRoles []models.Role, input RequestRolesParams) revel.Result {

	companyID := c.ViewArgs["companyID"].(string)

	var roleNames []string
	for _, role := range requestedRoles {
		roleNames = append(roleNames, role.RoleName)
	}

	notificationContent := models.NotificationContentType{
		RequesterUserID: requesterUserInfo.UserID,
		ActiveCompany:   companyID,
		RolesRequested:  input.Roles,
		IsAccepted:      "UNDER_REVIEW",
	}

	if len(roleNames) == 1 {
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to take on the role of " + roleNames[0] + "."
	} else {
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to take on the following roles: " + strings.Join(roleNames, ", ") + "."
	}

	createdNotification, err := ops.CreateNotification(ops.CreateNotificationInput{
		UserID:              companyAdminUserID,
		NotificationType:    constants.REQUEST_COMPANY_ROLE_UPDATE,
		NotificationContent: notificationContent,
		Global:              false,
	}, c.Controller)

	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to create notification for " + requesterUserInfo.UserID,
		})
	}

	utils.PrintJSON(createdNotification)

	return c.RenderJSON(createdNotification)
}

func (c RoleController) RequestRoles() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)

	var input RequestRolesParams
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.UserID}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "RequestRole Error: Missing required parameter - userId",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	//get company admins
	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company Admins",
		})
	}
	//get requester info
	requesterInfo, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    input.UserID,
		CompanyID: companyID,
	},
		c.Controller,
	)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company user",
		})
	}

	var rolesRequested []models.Role
	for _, role := range input.Roles {
		roleInfo, err := ops.GetRoleByID(role, companyID)
		if err != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "Unable to retrieve Roles",
			})
		}
		rolesRequested = append(rolesRequested, roleInfo)
	}

	// Check for existing pending requests
	if len(companyAdmins) != 0 {
		for _, compAdmin := range companyAdmins {
			userNotifications, _ := ops.GetUserNotifications(compAdmin.UserID, companyID)

			//handle duplicate notifs
			duplicateFound := false
			for _, notif := range userNotifications {
				if notif.NotificationType == constants.REQUEST_COMPANY_ROLE_UPDATE &&
					notif.UserID == compAdmin.UserID &&
					notif.NotificationContent.RequesterUserID == input.UserID &&
					notif.NotificationContent.ActiveCompany == companyID {

					if len(input.Roles) > 0 &&
						len(notif.NotificationContent.RolesRequested) > 0 &&
						utils.ComparingSlices(notif.NotificationContent.RolesRequested, input.Roles) &&
						notif.NotificationContent.IsAccepted == "UNDER_REVIEW" {
						duplicateFound = true
						break
					}
				}
			}

			if duplicateFound {
				c.Response.Status = 497
				return c.RenderJSON(models.ErrorResponse{
					Code:    "497",
					Message: "Request already submitted.",
					Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_497),
				})
			}
		}
	}

	currentTime := utils.GetCurrentTimestamp()
	pendingRequest := models.PendingRoleRequest{
		PK:             utils.AppendPrefix(constants.PREFIX_USER, input.UserID),
		SK:             utils.AppendPrefix(constants.PREFIX_ROLE_REQUEST, utils.GenerateTimestampWithUID()),
		UserID:         input.UserID,
		CompanyID:      companyID,
		RequestedRoles: input.Roles,
		Status:         "PENDING",
		CreatedAt:      currentTime,
		UpdatedAt:      currentTime,
		Type:           constants.ENTITY_TYPE_ROLE_REQUEST,
		RequestedBy:    c.ViewArgs["userID"].(string),
	}

	av, marshalErr := dynamodbattribute.MarshalMap(pendingRequest)
	if marshalErr != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			Code:    "500",
			Message: "Error marshalling pending request",
			Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	putInput := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, putErr := app.SVC.PutItem(putInput)
	if putErr != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			Code:    "500",
			Message: "Error storing pending request",
			Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	for _, compAdmin := range companyAdmins {
		sendNotificationToCompanyAdmin(c, compAdmin.UserID, requesterInfo, rolesRequested, input)
	}

	return c.RenderJSON(map[string]interface{}{
		"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_200),
		"message": "Role request submitted successfully",
		"request": pendingRequest,
	})
}

func (c RoleController) GetPendingRoleRequests() revel.Result {
	userID := c.Params.Query.Get("user_id")
	companyID := c.ViewArgs["companyID"].(string)

	revel.AppLog.Info("GetPendingRoleRequests called for userID:", userID, "companyID:", companyID)

	notifications, _ := ops.GetUserNotifications(userID, companyID)
	revel.AppLog.Info("Found notifications:", len(notifications))

	var pendingRequests []models.PendingRoleRequest

	for _, notif := range notifications {
		if notif.NotificationType == constants.REQUEST_COMPANY_ROLE_UPDATE &&
			notif.NotificationContent.RequesterUserID == userID &&
			notif.NotificationContent.IsAccepted == "UNDER_REVIEW" {

			pendingRequests = append(pendingRequests, models.PendingRoleRequest{
				UserID:         userID,
				RequestedRoles: notif.NotificationContent.RolesRequested,
				Status:         "PENDING",
				CreatedAt:      notif.CreatedAt,
				CompanyID:      companyID,
			})
		}
	}

	revel.AppLog.Info("Found pending requests from notifications:", len(pendingRequests))

	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String("PENDING"),
					},
				},
			},
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_ROLE_REQUEST),
					},
				},
			},
		},
		TableName: aws.String(app.TABLE_NAME),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		revel.AppLog.Error("Error querying DynamoDB:", err)
		return c.RenderJSON(models.ErrorResponse{
			Code:    "500",
			Message: "Error retrieving pending requests",
			Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	var dbRequests []models.PendingRoleRequest
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &dbRequests)
	if err != nil {
		revel.AppLog.Error("Error unmarshalling DynamoDB results:", err)
		return c.RenderJSON(models.ErrorResponse{
			Code:    "500",
			Message: "Error unmarshalling pending requests",
			Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	pendingRequests = append(pendingRequests, dbRequests...)
	revel.AppLog.Info("Total pending requests after merge:", len(pendingRequests))

	return c.RenderJSON(map[string]interface{}{
		"status":           utils.GetHTTPStatus(constants.HTTP_STATUS_200),
		"pending_requests": pendingRequests,
	})
}
