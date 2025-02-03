package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	// "time"

	"grooper/app"
	"grooper/app/cdn"
	"grooper/app/constants"
	"grooper/app/constraints"
	awsoperations "grooper/app/integrations/aws"
	docusignoperations "grooper/app/integrations/docusign"
	dropboxoperations "grooper/app/integrations/dropbox"
	githuboperations "grooper/app/integrations/github"
	googleoperations "grooper/app/integrations/google"
	jiraoperations "grooper/app/integrations/jira"
	officeOperations "grooper/app/integrations/microsoft"
	salesforceoperations "grooper/app/integrations/salesforce"
	zendeskoperations "grooper/app/integrations/zendesk"
	"grooper/app/mail"
	"grooper/app/models"
	jiramodel "grooper/app/models/jira"
	salesforceModel "grooper/app/models/salesforce"
	ops "grooper/app/operations"
	"grooper/app/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"

	// uuid "github.com/satori/go.uuid"
	wscontroller "grooper/app/controllers/web-socket"
)

type UserController struct {
	*revel.Controller
	UserOps ops.UserOperations
}

/*
****************
Get all users
Fetch all users in the database regardless of their companies
Params:
userID - used for fetching the users
****************
*/

// @Summary Get Users
// @Description This endpoint retrieves a list of all users in the system, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Success 200 {object} []models.LoginSuccessResponse
// @Router /users/all [get]
func (c UserController) GetAllUsers() revel.Result {
	data := make(map[string]interface{})
	users := []models.User{}

	result, err := ops.GetAll(constants.ENTITY_TYPE_USER, constants.INDEX_NAME_GET_USERS)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(data)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result, &users)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["users"] = users
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
Get user
Fetch a single user using its user ID
Params:
userID - used for fetching the user
****************
*/

// @Summary Get User
// @Description This endpoint retrieves information about a user by their User ID, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param include query string false "Include related data: groups or roles"
// @Param display_photo_size query int false "Display photo size"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/:userID [get]
func (c UserController) GetUser(userID string) revel.Result {

	include := c.Params.Query.Get("include")
	displayPhotoSize := c.Params.Query.Get("display_photo_size")

	companyID := c.ViewArgs["companyID"].(string)

	data := make(map[string]interface{})

	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	companyUser, err := GetCompanyUser(companyID, userID)
	if err != nil {
		data["error"] = err.Error()
		return c.RenderJSON(data)
	}
	//This is to handle empty value for old accounts
	if companyUser.FirstName == "" {
		companyUser.FirstName = user.FirstName
	}
	if companyUser.LastName == "" {
		companyUser.LastName = user.LastName
	}
	if companyUser.JobTitle == "" {
		companyUser.JobTitle = user.JobTitle
	}
	if companyUser.ContactNumber == "" {
		companyUser.ContactNumber = user.ContactNumber
	}
	if companyUser.DisplayPhoto == "" {
		companyUser.DisplayPhoto = user.DisplayPhoto
	}
	companyUser.Email = user.Email
	companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email

	companyUser.Email = user.Email
	if companyUser.Status == "DELETED" {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(data)
	}
	//
	//
	//
	//
	//

	if companyUser.UserID == "" {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(data)
	}

	//DisplayPhoto if value not nil -gcm
	if len(companyUser.DisplayPhoto) != 0 {
		resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, displayPhotoSize)
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err != nil {
			data["error"] = err.Error()
			return c.RenderJSON(data)
		}
		companyUser.DisplayPhoto = fileName
	}

	//DisplayPhoto if empty, set value to empty string -gcm
	if len(companyUser.DisplayPhoto) == 0 {
		companyUser.DisplayPhoto = ""
	}

	// ORIGINAL CODE FOR DISPLAYPHOTO
	// resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, displayPhotoSize)
	// fileName, err := cdn.GetImageFromStorage(resizedPhoto)
	// if err != nil {
	// 	data["error"] = err.Error()
	// 	return c.RenderJSON(data)
	// }
	// user.DisplayPhoto = fileName
	//

	if len(include) != 0 {
		if strings.Contains(include, "groups") {
			// result, err := GetUserCompanyGroups(companyID, user.UserID)
			// if err != nil {
			// 	data["status"] = utils.GetHTTPStatus(err.Error())
			// }
			// user.Groups = result

			result, err := ops.GetUserConnectedGroups(companyID, user.UserID)
			if err != nil {
				data["status"] = utils.GetHTTPStatus(err.Error())
			}
			for idx, g := range result {
				group, err := getGroupInTmp(companyID, g.GroupID)
				if err == nil {
					department := getDepartmentInTmp(companyID, group.DepartmentID)
					if department != nil {
						group.Department = *department
					}
					result[idx] = group
				}
			}
			companyUser.Groups = result
		}
		if strings.Contains(include, "roles") {
			result, err := GetUserRoles(user.UserID, companyID)
			if err != nil {
				data["status"] = utils.GetHTTPStatus(err.Error())
			}
			_ = result
			companyUser.Roles = result
		}
	}

	data["user"] = companyUser

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	//
	//
	//
	//
	//

	return c.RenderJSON(data)
}

// @Summary Get Current User
// @Description This endpoint retrieves information about the user currently logged in, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/me [get]
func (c UserController) GetCurrentUser() revel.Result {
	result := make(map[string]interface{})

	userID := c.ViewArgs["userID"].(string)

	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}
	companyUser, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
		CompanyID: user.ActiveCompany,
	}, c.Controller)

	//This is to handle empty value for old accounts
	if companyUser.FirstName == "" {
		companyUser.FirstName = user.FirstName
	}
	if companyUser.LastName == "" {
		companyUser.LastName = user.LastName
	}
	if companyUser.JobTitle == "" {
		companyUser.JobTitle = user.JobTitle
	}
	if companyUser.ContactNumber == "" {
		companyUser.ContactNumber = user.ContactNumber
	}
	if companyUser.DisplayPhoto == "" {
		companyUser.DisplayPhoto = user.DisplayPhoto
	}
	if companyUser.SecondaryEmail == "" {
		companyUser.SecondaryEmail = user.SecondaryEmail
	}
	companyUser.Email = user.Email
	companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email

	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	if companyUser.UserID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	//DisplayPhoto if value not nil -gcm
	if len(companyUser.DisplayPhoto) != 0 {
		resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, "_100")
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err != nil {
			result["error"] = err.Error()
			return c.RenderJSON(result)
		}
		companyUser.DisplayPhoto = fileName
	}

	//DisplayPhoto if empty, set value to empty string -gcm
	if len(companyUser.DisplayPhoto) == 0 {
		companyUser.DisplayPhoto = ""
	}

	//ORIGINAL CODE FOR DISPLAYPHOTO
	// if user.DisplayPhoto != "" {
	// 	resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")
	// 	fileName, err := cdn.GetImageFromStorage(resizedPhoto)
	// 	if err != nil {
	// 		result["error"] = err.Error()
	// 		return c.RenderJSON(result)
	// 	}
	// 	user.DisplayPhoto = fileName
	// }

	// include user companies
	// companies, _, err := GetCompaniesByUser("", "", userID, 50, "_100")
	companies, err := GetUserActiveCompanies(userID)

	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	//

	user.Companies = companies

	// userRoles, err := GetUserRoles(user.UserID, c.ViewArgs["companyID"].(string))
	// if err != nil {
	// 	result["status"] = utils.GetHTTPStatus(err.Error())
	// }

	company, opsError := ops.GetCompanyByID(user.ActiveCompany)
	_ = opsError

	// var isTourDone string
	// if user.IsTourDone == "" {
	// 	isTourDone = constants.BOOL_FALSE
	// }else{
	// 	isTourDone = user.IsTourDone
	// }

	// return c.RenderJSON(isAdmin)

	// filtered data to be returned
	activeCompany := user.ActiveCompany
	changeCompany := false
	if len(companies) != 0 {
		if user.ActiveCompany == "" {
			activeCompany = companies[0].CompanyID
			cac := ops.ChangeActiveCompanyarams{UserID: user.UserID, CompanyID: activeCompany, Email: user.Email}
			opsError := ops.ChangeActiveCompany(cac, c.Controller)
			if opsError != nil {
				c.Response.Status = opsError.HTTPStatusCode
				return c.RenderJSON(opsError)
			}
		} else {
			isPart := ops.CheckIfActiveCompanyIsPart(companies, user.ActiveCompany)
			if isPart == false {
				activeCompany = companies[0].CompanyID
				cac := ops.ChangeActiveCompanyarams{UserID: user.UserID, CompanyID: activeCompany, Email: user.Email}
				opsError := ops.ChangeActiveCompany(cac, c.Controller)
				if opsError != nil {
					c.Response.Status = opsError.HTTPStatusCode
					return c.RenderJSON(opsError)
				}
				changeCompany = true
			}
		}

	} else {
		activeCompany = ""
		changeCompany = true
	}

	roles, err := GetCompanyUserRoles(activeCompany, user.UserID)
	if err != nil {

	}

	isAdmin := false
	// if companyUser.UserType != "" {
	// 	isAdmin = companyUser.UserType == constants.USER_TYPE_COMPANY_OWNER
	// } else {
	isAdmin, err = ops.CheckCountRolePermissions(user.UserID, user.ActiveCompany)
	if err != nil {
		c.Response.Status = 400
		result["message"] = "Error getting role permissions"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	// }
	if companyUser.UserType == constants.USER_TYPE_COMPANY_OWNER {
		isAdmin = true
	}

	rolesWithData, err := GetCompanyUserRolesInformation(activeCompany, roles)
	if err != nil {
		result["err"] = err.Error()
		// return c.RenderJSON(err.Error())
	}
	// return c.RenderJSON(rolesWithData)

	// var Roles []models.Role

	var Permissions []string

	// user.Roles = userRoles
	// for _, item := range userRoles {
	// 	role, opsError := ops.GetRoleByID(item.RoleID, user.ActiveCompany)
	// 	if opsError != nil {
	// 		result["status"] = utils.GetHTTPStatus(opsError.Status.Code)
	// 		return c.RenderJSON(result)
	// 	}

	// 	Roles = append(Roles, role)
	// }

	for _, role := range rolesWithData {
		// utils.PrintJSON(role, "rolesWithData role")
		// uniquePermissions := utils.AppendIfUnique(role.RolePermissions, Permissions)
		Permissions = append(Permissions, role.RolePermissions...)
	}

	Permissions = utils.RemoveDuplicateStrings(Permissions)

	user.Permissions = Permissions

	u := models.User{
		ActiveCompany:       activeCompany,
		ContactNumber:       companyUser.ContactNumber,
		CreatedAt:           companyUser.CreatedAt,
		DisplayPhoto:        companyUser.DisplayPhoto,
		Email:               user.Email,
		FirstName:           companyUser.FirstName,
		JobTitle:            companyUser.JobTitle,
		LastName:            companyUser.LastName,
		UserID:              user.UserID,
		Companies:           user.Companies,
		Permissions:         user.Permissions,
		BookmarkGroups:      user.BookmarkGroups,
		BookmarkDepartments: user.BookmarkDepartments,
		// GoogleAccessToken:     user.GoogleAccessToken,
		// GoogleRefreshToken:    user.GoogleRefreshToken,
		// GoogleTokenExpiration: user.GoogleTokenExpiration,
		// GoogleEmailDomain:     user.GoogleEmailDomain,
		// JiraToken:         user.JiraToken,
		Roles:             user.Roles,
		SetupWizardStatus: company.SetupWizardStatus,
		IsAdmin:           isAdmin,
		IsTourDone:        user.IsTourDone,
		IsVerified:        user.IsVerified,
		SecondaryEmail:    companyUser.SecondaryEmail,
		AutoCompanyChange: changeCompany,
	}

	// TODO: refactor, remove unecessary return values
	// result["token"] = ops.EncodeToken(user)
	// result["user"] = u
	// result["rolesWithData"] = rolesWithData
	// result["roles"] = roles
	// result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(models.GetCurrentUserResponse{
		Token:         ops.EncodeToken(user),
		User:          u,
		RolesWithData: rolesWithData,
		Roles:         roles,
		Status:        utils.GetHTTPStatus(constants.HTTP_STATUS_200), // https://hoolisoftware.atlassian.net/browse/SAAS-7381
	})
}

func GetCurrentUser(companyID, userID string, c *revel.Controller) (models.User, string, error) {
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		return models.User{}, "", errors.New(opsError.Status.Code)
	}
	companyUser, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
		CompanyID: companyID,
	}, c)
	if opsError != nil {
		return models.User{}, "", errors.New(opsError.Status.Code)
	}
	if user.UserID == "" || companyUser.UserID == "" {
		return models.User{}, "", errors.New("404")
	}

	//DisplayPhoto if value not nil -gcm
	if len(companyUser.DisplayPhoto) != 0 {
		resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, "_100")
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err != nil {
			return models.User{}, "", errors.New(err.Error())
		}
		companyUser.DisplayPhoto = fileName
	}

	//DisplayPhoto if empty, set value to empty string -gcm
	if len(companyUser.DisplayPhoto) == 0 {
		companyUser.DisplayPhoto = ""
	}

	//ORIGINAL CODE FOR DISPLAYPHOTO
	// if user.DisplayPhoto != "" {
	// 	resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")
	// 	fileName, err := cdn.GetImageFromStorage(resizedPhoto)
	// 	if err != nil {
	// 		result["error"] = err.Error()
	// 		return c.RenderJSON(result)
	// 	}
	// 	user.DisplayPhoto = fileName
	// }

	// include user companies
	// companies, _, err := GetCompaniesByUser("", "", userID, 50, "_100")
	companies, err := GetUserActiveCompanies(userID)
	if err != nil {
		return models.User{}, "", errors.New("500")
	}

	//

	user.Companies = companies

	userRoles, err := GetUserRoles(user.UserID, companyID)
	if err != nil {
		return models.User{}, "", errors.New(err.Error())
	}

	var Roles []models.Role

	var Permissions []string

	// user.Roles = userRoles
	for _, item := range userRoles {
		role, opsError := ops.GetRoleByID(item.RoleID, user.ActiveCompany)
		if opsError != nil {
			return models.User{}, "", errors.New(opsError.Status.Code)
		}

		Roles = append(Roles, role)
	}

	for _, role := range Roles {
		uniquePermissions := utils.AppendIfUnique(role.RolePermissions, Permissions)
		Permissions = append(Permissions, uniquePermissions...)

	}

	user.Permissions = Permissions

	isAdmin, err := ops.CheckCountRolePermissions(user.UserID, user.ActiveCompany)
	if err != nil {
		return models.User{}, "", errors.New("400")
	}

	company, opsError := ops.GetCompanyByID(user.ActiveCompany)
	_ = opsError

	// var isTourDone string
	// if user.IsTourDone == "" {
	// 	isTourDone = constants.BOOL_FALSE
	// }else{
	// 	isTourDone = user.IsTourDone
	// }

	// return c.RenderJSON(isAdmin)

	// filtered data to be returned
	u := models.User{
		ActiveCompany:       user.ActiveCompany,
		ContactNumber:       companyUser.ContactNumber,
		CreatedAt:           companyUser.CreatedAt,
		DisplayPhoto:        companyUser.DisplayPhoto,
		Email:               user.Email,
		FirstName:           companyUser.FirstName,
		JobTitle:            companyUser.JobTitle,
		LastName:            companyUser.LastName,
		UserID:              user.UserID,
		Companies:           user.Companies,
		Permissions:         user.Permissions,
		BookmarkGroups:      user.BookmarkGroups,
		BookmarkDepartments: user.BookmarkDepartments,
		// GoogleAccessToken:     user.GoogleAccessToken,
		// GoogleRefreshToken:    user.GoogleRefreshToken,
		// GoogleTokenExpiration: user.GoogleTokenExpiration,
		// GoogleEmailDomain:     user.GoogleEmailDomain,
		// JiraToken:         user.JiraToken,
		Roles:             user.Roles,
		SetupWizardStatus: company.SetupWizardStatus,
		IsAdmin:           isAdmin,
		IsTourDone:        user.IsTourDone,
		IsVerified:        user.IsVerified,
		SSO:               user.SSO,
	}

	return u, ops.EncodeToken(user), nil
}

/*
****************
Get users
Fetch users based on their companies or groups
Params:
company_id - for fetching users inside a certain company
group_id - for fetching users inside a certain group
key - use for searching
include - (groups)
last_evaluated_key - use for pagination
limit - limit number of items per page
****************
*/

// @Summary Get Users
// @Description This endpoint retrieves a list of users with various filtering and sorting options, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param group_id query string false "Group ID"
// @Param department_id query string false "Department ID"
// @Param include query string false "Include related data: groups or roles"
// @Param key query string false "Search Key"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param limit query int false "Limit the number of results"
// @Param status query string false "Status"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies [get]
func (c UserController) GetUsers() revel.Result {

	//Parameters
	companyID := c.ViewArgs["companyID"].(string)
	groupID := c.Params.Query.Get("group_id")
	departmentID := c.Params.Query.Get("department_id")
	include := c.Params.Query.Get("include")
	searchKey := c.Params.Query.Get("key")
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	limit := c.Params.Query.Get("limit")
	status := c.Params.Query.Get("status")

	// if status == "" {
	// 	status = constants.ITEM_STATUS_ACTIVE
	// }

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	// var serviceType string
	serviceType := "by_company"

	if companyID == "" && groupID == "" && departmentID == "" {
		data["error"] = "Missing required parameters for primary keys."
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(data)
		// } else if groupID != "" {
		// 	serviceType = "by_group"
		// } else if departmentID != "" && groupID == "" {
		// 	serviceType = "by_department"
		// } else if companyID != "" && groupID == "" && departmentID == "" {
		// 	serviceType = "by_company"
	}
	// else {
	// 	data["error"] = "Too many required parameters."
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
	// }

	if searchKey != "" {
		lowerKeys := strings.ToLower(searchKey)
		searchKey = strings.Trim(lowerKeys, " ")
	}

	var pageLimit int64
	if limit != "" {
		pageLimit = utils.ToInt64(limit)
	} else {
		pageLimit = constants.DEFAULT_PAGE_LIMIT
	}

	users := []models.CompanyUser{}
	lastEvaluatedKey := models.CompanyUser{}
	var err error

	outputUsers := []models.CompanyUser{}

	//

	switch serviceType {

	// By Group
	case "by_group":
		users, lastEvaluatedKey, err = GetUsersByGroup(searchKey, status, paramLastEvaluatedKey, groupID, companyID, departmentID, pageLimit)

		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}

		if len(include) != 0 {
			for i, user := range users {
				if strings.Contains(include, "groups") {
					result, err := GetUserCompanyGroups(companyID, user.UserID)
					if err != nil {
						data["status"] = utils.GetHTTPStatus(err.Error())
					}
					users[i].Groups = result
				}
				if strings.Contains(include, "roles") {
					result, err := GetUserRoles(user.UserID, companyID)
					if err != nil {
						data["status"] = utils.GetHTTPStatus(err.Error())
					}
					users[i].Roles = result
				}
				//TODO include integrations once implemented
			}
		}

	//By Company
	case "by_company":
		s := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}
		// s := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT, constants.ITEM_STATUS_SCHEDULED}
		_ = s
		if status != "" {
			s = []string{strings.ToUpper(status)}
		}

		users, lastEvaluatedKey, err = GetCompanyUsersNewWithLimit(GetCompanyUsersInput{
			CompanyID:        companyID,
			Status:           s,
			SearchKey:        searchKey,
			LastEvaluatedKey: paramLastEvaluatedKey,
			Include:          include,
			GroupID:          groupID,
		})
		if err != nil {

		}

		// usersWithData, err := GetCompanyUsersInformation(companyID, users, c.Controller)

		// if err != nil {

		// }

		if len(include) != 0 {
			for i, user := range users {
				_ = i
				_ = user
				if strings.Contains(include, "groups") || groupID != "" || (departmentID != "" && groupID == "") || (departmentID != "" && groupID != "") {
					result, err := GetUserCompanyGroupsNew(companyID, user.UserID)
					if err != nil {

					}

					for idx, g := range result {
						// group, err := getGroupInTmp(companyID, g.GroupID)
						if err == nil {
							result[idx].DepartmentID = g.DepartmentID
							result[idx].Department.DepartmentID = g.DepartmentID
							result[idx].GroupName = g.GroupName
							result[idx].GroupColor = g.GroupColor
						}
					}
					users[i].Groups = result
				}

				if strings.Contains(include, "roles") {
					roles, err := GetCompanyUserRoles(companyID, user.UserID)
					if err != nil {

					}

					rolesWithData, err := GetCompanyUserRolesInformation(companyID, roles)
					if err != nil {

					}
					users[i].Roles = rolesWithData
				}
			}
		}

		outputUsers = users
		data["totalUsers"] = len(outputUsers)
		data["lastEvaluatedKey"] = lastEvaluatedKey

	case "by_department":
		users, lastEvaluatedKey, err = GetUsersByDepartment(searchKey, status, paramLastEvaluatedKey, companyID, departmentID, pageLimit)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}

		if len(include) != 0 {
			for i, user := range users {
				if strings.Contains(include, "groups") {
					result, err := GetUserCompanyGroups(companyID, user.UserID)
					if err != nil {
						data["status"] = utils.GetHTTPStatus(err.Error())
					}
					users[i].Groups = result
				}
				if strings.Contains(include, "roles") {
					result, err := GetUserRoles(user.UserID, companyID)
					if err != nil {
						data["status"] = utils.GetHTTPStatus(err.Error())
					}
					users[i].Roles = result
				}
				//TODO include integrations once implemented
			}
		}

	default:
		data["error"] = "Conflicting parameters. Cannot determine service type."
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(data)
	}

	if searchKey != "" && len(users) == 0 {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(data)
	}

	data["lastEvaluatedKey"] = lastEvaluatedKey
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	// data["users"] = users

	data["users"] = outputUsers
	return c.RenderJSON(data)
}

/*
****************
AddUsersToCompany()
- Add multiple users to a company
Body:
companyID <string>
users <[]User>
****************
*/

// @Summary Add Users to Company
// @Description This endpoint adds existing users to a specified company, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.AddUsersToCompanyRequest true "Add users to company body"
// @Param is_admin query bool true "Specifies if the user is a company admin"
// @Param create_account query bool true "Indicates if an account should be created for the user"
// @Param is_add_user_to_group query bool true "Determines if the user should be added to a group"
// @Param key query string false "Search Key"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param limit query int false "Limit the number of results"
// @Param status query string false "Filter by user status"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users [post]
func (c UserController) AddUsersToCompany() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	result := make(map[string]interface{})

	var isCompanyAdmin bool
	var createAccount bool
	var isAddUserToGroup bool
	c.Params.Bind(&isCompanyAdmin, "is_admin")
	c.Params.Bind(&createAccount, "create_account")
	c.Params.Bind(&isAddUserToGroup, "is_add_user_to_group")
	if isCompanyAdmin {
		createAccount = true
	}

	groupID := c.Params.Form.Get("group_id")
	departmentID := c.Params.Form.Get("department_id")

	// form body
	users := c.Params.Form.Get("users")
	searchKey := c.Params.Query.Get("key")
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	limit := c.Params.Query.Get("limit")
	status := c.Params.Query.Get("status")
	skipExistingEmails, err := strconv.ParseBool(c.Params.Form.Get("skip_existing_emails"))
	if err != nil {
		skipExistingEmails = false
	}

	if companyID == "" || users == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	// check if company id exists
	// companyExists, err := IsCompanyIdExists(companyID)
	// if !companyExists || err != nil {
	// 	result["status"] = utils.GetHTTPStatus(err.Error())
	// 	return c.RenderJSON(result)
	// }

	// check if request user id is existing on the company
	gcm := ops.GetCompanyMemberParams{UserID: userID, CompanyID: companyID}
	usr, opsError := ops.GetCompanyMember(gcm, c.Controller)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	// todo: check permisions

	var unmarshalUsers []models.User
	json.Unmarshal([]byte(users), &unmarshalUsers)

	if len(unmarshalUsers) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	for _, user := range unmarshalUsers {
		user.Validate(c.Validation, constants.SERVICE_TYPE_ADD_USER)
		if c.Validation.HasErrors() {
			result["errors"] = c.Validation.Errors
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
			return c.RenderJSON(result)
		}
	}

	addUsersResult, err := c.AddUsers(unmarshalUsers, companyID, searchKey, paramLastEvaluatedKey, limit, status, skipExistingEmails, isCompanyAdmin, createAccount)
	if err != nil {
		if addUsersResult != nil {
			r, ok := addUsersResult.(AddUserError)
			if ok {
				result["errors"] = r.Emails
				result["reason"] = r.Reason
			}
		}
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	if isCompanyAdmin {
		//
		//
		//
		// //
		//
		//
		// // )
		//
		//
		for _, userz := range addUsersResult.([]models.User) {
			//assign user role when new user signed up
			item := models.UserRole{
				PK: utils.AppendPrefix(constants.PREFIX_USER, userz.UserID),
				SK: utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"), utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
				// SK:        utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"),
				UserID:    userz.UserID,
				RoleID:    "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e",
				CompanyID: companyID,
				Type:      constants.ENTITY_TYPE_USER_ROLE,
			}

			av, err := dynamodbattribute.MarshalMap(item)
			if err != nil {
				result["error"] = "Error at marshalmap"
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}

			input := &dynamodb.PutItemInput{
				Item:      av,
				TableName: aws.String(app.TABLE_NAME),
			}

			_, err = app.SVC.PutItem(input)
			if err != nil {
				result["error"] = "Cannot assign role due to server error"
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}
		}
	}

	var handler = usr.Handler

	if usr.Handler == "" {
		handler = c.ViewArgs["userID"].(string)
	}

	_, ok := addUsersResult.([]models.User)
	if ok && len(addUsersResult.([]models.User)) != 0 {
		err = AddUsersToCompany(companyID, addUsersResult.([]models.User), c.ViewArgs["userID"].(string), handler, createAccount)
		if err != nil {
			result["error"] = "AddUsersToCompany"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
	}

	if isAddUserToGroup && len(addUsersResult.([]models.User)) > 0 {
		// Add new user to current group from group info - add member module
		var membersID []string
		for _, user := range addUsersResult.([]models.User) {
			membersID = append(membersID, user.UserID)
		}
		groupList, err1 := GetGroupList(companyID)
		if err1 != nil {
			c.Response.Status = 400
			return c.RenderJSON(err.Error())
		}
		output := len(groupList)
		memberType := constants.MEMBER_TYPE_USER
		memberRole := constants.MEMBER_TYPE_USER
		typeGroup := constants.ENTITY_TYPE_GROUP_MEMBER
		prefix := constants.PREFIX_USER

		err := ops.AddUsersToGroups(membersID, memberType, memberRole, typeGroup, prefix, companyID, groupID, departmentID, userID, output)
		if err != nil {
			return c.RenderJSON(err)
		}
	}

	var statusCode constants.Status
	if !reflect.ValueOf(addUsersResult).IsNil() {
		statusCode = utils.GetHTTPStatus(constants.HTTP_STATUS_201)
	} else {
		statusCode = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
	}

	result["addUserResult"] = addUsersResult
	result["addedUsersLength"] = len(addUsersResult.([]models.User))
	result["status"] = statusCode
	return c.RenderJSON(result)
}

type AddUserError struct {
	Emails []string
	Reason string
}

/*
****************
AddUsers()
- Add multiple users, if there are error interface{} returns duplicates and not unique email
- Return user ids of inserted users
****************
*/
func (c UserController) AddUsers(users []models.User, activeCompany, searchKey, paramLastEvaluatedKey, limit, status string, skipExistingEmails, isCompanyAdmin, createAccount bool) (interface{}, error) {
	var emailExists []string

	userID := c.ViewArgs["userID"].(string)
	companyID := c.ViewArgs["companyID"].(string)

	gass := ops.GetAvailableSubscriptionSeatsParams{UserID: userID, CompanyID: companyID}
	availableSeats, opsError := ops.GetAvailableSubscriptionSeats(gass, c.Controller)
	if opsError != nil {
		return nil, errors.New(opsError.Status.Code)
	}

	if availableSeats == 0 {
		e := errors.New(constants.HTTP_STATUS_485)
		return nil, e
	}

	var validUsers []models.User

	// check duplicate emails
	// duplicates := CheckDuplicateEmails(users)
	// if len(duplicates) != 0 {
	// 	e := errors.New(constants.HTTP_STATUS_400)
	// 	// var result = struct {
	// 	// 	Duplicates []string
	// 	// }{duplicates}
	// 	// return result, e
	// 	return AddUserError{
	// 		Emails: duplicates,
	// 		Reason: constants.ERROR_REASON_DUPLICATE_EMAIL_ADDRESS,
	// 	}, e
	// }

	uniqueUsers := GetUniqueUsers(users)

	// loop on uniqueUsers, use batch
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	// get users in the company

	for i, user := range uniqueUsers {
		// // check if email is unique <- ORIGINAL CHECKER FOR EMAIL
		// err := IsEmailUnique(user.Email)
		// if err != nil {
		// 	if err.Error() == constants.HTTP_STATUS_473 {
		// 		emailExists = append(emailExists, user.Email)
		// 	} else {
		// 		e := errors.New(err.Error())
		// 		return nil, e
		// 	}
		// }
		// check if useremail is unique incompany <-NEW CHCECKER
		error := IsEmailUniqueInCompany(user.Email, activeCompany, searchKey, paramLastEvaluatedKey, limit, isCompanyAdmin, []string{})
		if error != nil {
			if error.Error() == constants.HTTP_STATUS_473 {
				emailExists = append(emailExists, user.Email)
			}
			// else {
			// 	e := errors.New(error.Error())
			// 	return nil, e
			// }
			continue
		}
		// generate token for activation
		userToken := utils.GenerateRandomString(8)

		//? ADDED THIS to replace the ops.GetUserByEmail
		s := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}
		_ = s
		if status != "" {
			s = []string{strings.ToUpper(status)}
		}
		users, err := GetCompanyUsersNew(GetCompanyUsersInput{
			CompanyID: activeCompany,
			Status:    s,
		})
		if err != nil {

		}
		usersWithData, err := GetCompanyUsersInformation(activeCompany, users, c.Controller)

		if err != nil {

		}

		isExisting := false
		for _, cUser := range usersWithData {
			if cUser.Email == user.Email {
				isExisting = true
			}
		}

		//? Commented this since there is bug that occur
		//? For example: I created a new fresh user and then I deleted it
		//? And then I created a new user again with different info (Name, Job) but with the same Email I used on creating the previous one.
		//? Then it gets the previous info instead of nothing, since it is a new user.
		userID := ""
		u, uerr := ops.GetUserByEmail(user.Email)
		if uerr == nil {
			if u.UserID != "" && utils.StringInSlice(u.Status, []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}) {
				userID = u.UserID
			}
		}
		// check if user is existing on the db, and status = ACTIVE
		// u, uerr := ops.GetUserByEmail(user.Email)
		// if uerr != nil {

		// }
		// if err == nil {
		// 	if u.UserID != "" && u.Status == constants.ITEM_STATUS_ACTIVE {
		// 		input := &dynamodb.UpdateItemInput{
		// 			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
		// 				":s": {
		// 					S: aws.String(constants.ITEM_STATUS_ACTIVE),
		// 				},
		// 				":c": {
		// 					S: aws.String(activeCompany),
		// 				},
		// 				":ut": {
		// 					S: aws.String(userToken),
		// 				},
		// 				":ua": {
		// 					S: aws.String(utils.GetCurrentTimestamp()),
		// 				},
		// 			},
		// 			ExpressionAttributeNames: map[string]*string{
		// 				"#s": aws.String("Status"),
		// 			},
		// 			TableName: aws.String(app.TABLE_NAME),
		// 			Key: map[string]*dynamodb.AttributeValue{
		// 				"PK": {
		// 					S: aws.String(u.PK),
		// 				},
		// 				"SK": {
		// 					S: aws.String(u.SK),
		// 				},
		// 			},
		// 			UpdateExpression: aws.String("SET #s = :s, ActiveCompany = :c, UserToken = :ut, UpdatedAt = :ua"),
		// 		}
		// 		_, err = app.SVC.UpdateItem(input)
		// 		if err == nil {
		// 			users[i].UserID = u.UserID
		// 			users[i].UserToken = userToken
		// 		}
		// 		continue
		// 	}
		// }
		if !isExisting {
			// set user uuid
			// set user uuid
			if userID == "" {
				userID = utils.GenerateTimestampWithUID()
			}

			// set search key
			searchKey := user.FirstName + " " + user.LastName + " " + user.Email
			searchKey = strings.ToLower(searchKey)

			status := constants.ITEM_STATUS_PENDING

			// push to batch
			var role []string
			if !createAccount {
				role = []string{constants.USER_ROLE_USER}
				userToken = "DEFAULT_USER"
				status = constants.ITEM_STATUS_ACTIVE
			}
			roles, _ := dynamodbattribute.MarshalList(role)

			// set status
			// var status string
			// if user.Status == "" {
			// 	status = constants.ITEM_STATUS_PENDING
			// } else {
			// 	status = user.Status
			// }

			//? START COMMENTED HERE
			// push to batch if user not exists
			// if u.PK == "" {
			// 	currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			// 		Item: map[string]*dynamodb.AttributeValue{
			// 			"PK": {
			// 				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
			// 			},
			// 			"SK": {
			// 				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
			// 			},
			// 			"UserID": {
			// 				S: aws.String(userID),
			// 			},
			// 			"Email": {
			// 				S: aws.String(user.Email),
			// 			},
			// 			"FirstName": {
			// 				S: aws.String(strings.Title(user.FirstName)),
			// 			},
			// 			"LastName": {
			// 				S: aws.String(strings.Title(user.LastName)),
			// 			},
			// 			"UserRole": {
			// 				L: roles,
			// 			},
			// 			"JobTitle": {
			// 				S: aws.String(user.JobTitle),
			// 			},
			// 			"Type": {
			// 				S: aws.String(constants.ENTITY_TYPE_USER),
			// 			},
			// 			"Status": {
			// 				S: aws.String(status),
			// 			},
			// 			"SearchKey": {
			// 				S: aws.String(searchKey),
			// 			},
			// 			"UserToken": {
			// 				S: aws.String(userToken),
			// 			},
			// 			"ActiveCompany": {
			// 				S: aws.String(activeCompany),
			// 			},
			// 			"CreatedAt": {
			// 				S: aws.String(utils.GetCurrentTimestamp()),
			// 			},
			// 			"IsTourDone": {
			// 				S: aws.String(constants.BOOL_FALSE),
			// 			},
			// 		},
			// 	}})
			// 	// uniqueUsers[i].UserID = userID
			// 	// uniqueUsers[i].UserToken = userToken
			// 	tmp := user
			// 	tmp.UserID = userID
			// 	tmp.UserToken = userToken
			// 	tmp.SearchKey = searchKey
			// 	tmp.UserRole = role
			// 	validUsers = append(validUsers, tmp)
			// } else {
			// 	// uniqueUsers[i] = u
			// 	u.FirstName = user.FirstName
			// 	u.LastName = user.LastName
			// 	u.JobTitle = user.JobTitle
			// 	u.Email = user.Email
			// 	u.SearchKey = user.SearchKey
			// 	u.UserRole = user.UserRole
			// 	u.Permissions = user.Permissions
			// 	validUsers = append(validUsers, u)
			// }
			// if i%constants.BATCH_LIMIT == 0 {
			// 	if len(currentBatch) != 0 {
			// 		batches = append(batches, currentBatch)
			// 	}
			// 	currentBatch = nil
			// }
			//? END OF COMMENT HERE

			//? REPLACED WITH THIS
			if u.UserToken != "DONE" {
				currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"PK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
						},
						"SK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
						},
						"UserID": {
							S: aws.String(userID),
						},
						"Email": {
							S: aws.String(user.Email),
						},
						"FirstName": {
							S: aws.String(strings.Title(user.FirstName)),
						},
						"LastName": {
							S: aws.String(strings.Title(user.LastName)),
						},
						"UserRole": {
							L: roles,
						},
						"JobTitle": {
							S: aws.String(user.JobTitle),
						},
						"Type": {
							S: aws.String(constants.ENTITY_TYPE_USER),
						},
						"Status": {
							S: aws.String(status),
						},
						"SearchKey": {
							S: aws.String(searchKey),
						},
						"UserToken": {
							S: aws.String(userToken),
						},
						"ActiveCompany": {
							S: aws.String(activeCompany),
						},
						"CreatedAt": {
							S: aws.String(utils.GetCurrentTimestamp()),
						},
						"IsTourDone": {
							S: aws.String(constants.BOOL_FALSE),
						},
					},
				}})
				// uniqueUsers[i].UserID = userID
				// uniqueUsers[i].UserToken = userToken
				tmp := user
				tmp.UserID = userID
				tmp.UserToken = userToken
				tmp.SearchKey = searchKey
				tmp.UserRole = role
				validUsers = append(validUsers, tmp)

				if i%constants.BATCH_LIMIT == 0 {
					if len(currentBatch) != 0 {
						batches = append(batches, currentBatch)
					}
					currentBatch = nil
				}
			} else {
				tmp := u
				tmp.FirstName = user.FirstName
				tmp.LastName = user.LastName
				tmp.JobTitle = user.JobTitle
				validUsers = append(validUsers, tmp)
			}

			//? END OF REPLACEMENT
		}
	}

	if len(emailExists) != 0 {
		e := errors.New(constants.HTTP_STATUS_473)
		// return emailExists, e
		if !skipExistingEmails {
			return AddUserError{
				Emails: emailExists,
				Reason: constants.ERROR_REASON_EMAIL_ADDRESSES_EXISTS,
			}, e
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		if len(currentBatch) != 0 {
			batches = append(batches, currentBatch)
		}
	}

	// if len(emailExists) != 0 {
	// 	e := errors.New(constants.HTTP_STATUS_473)
	// 	// return emailExists, e
	// 	return AddUserError{
	// 		Emails: emailExists,
	// 		Reason: constants.ERROR_REASON_EMAIL_ADDRESSES_EXISTS,
	// 	}, e
	// }

	if len(batches) != 0 {
		_, err := ops.BatchWriteItemHandler(batches)
		if err != nil {
			e := errors.New(err.Error())
			return nil, e
		}
	}

	// send activation link
	if createAccount {
		for _, v := range validUsers {

			if v.UserToken != "DONE" {
				sci := ops.SendCompanyInvitationParams{
					CompanyID: companyID,
					UserID:    v.UserID,
				}
				err := ops.SendCompanyInvitation(sci, c.Controller)
				if err != nil {

				}
			} else {

				company, opsErr := ops.GetCompanyByID(companyID)
				if opsErr != nil {
				}

				notificationContent := models.NotificationContentType{
					RequesterUserID: userID,
					Message:         "An admin has updated your access to " + company.CompanyName,
				}

				_, opsError := ops.CreateNotification(ops.CreateNotificationInput{
					UserID:              v.UserID,
					NotificationType:    constants.ROLE_UPDATE,
					NotificationContent: notificationContent,
					Global:              true,
				}, c.Controller)
				if opsError != nil {
				}
			}
		}
	}

	// var recipients []mail.Recipient
	// frontendUrl, _ := revel.Config.String("url.frontend")
	// company, err := ops.GetCompanyByID(companyID)

	// for _, user := range users {
	// 	if user.UserToken != "" {
	// 		token, _ := utils.EncodeToJwtToken(jwt.MapClaims{
	// 			"userToken": user.UserToken,
	// 			"companyID": companyID,
	// 			"userID": user.UserID,
	// 			// "nbf":       time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	// 		})
	// 		recipients = append(recipients, mail.Recipient{
	// 			Name:           user.FirstName + " " + user.LastName,
	// 			Email:          user.Email,
	// 			InviteUserLink: frontendUrl + "/verify-email?token=" + token,
	// 			CompanyName:    company.CompanyName,
	// 		})
	// 	}
	// }

	// jobs.Now(mail.SendEmail{
	// 	Subject:    "You have been invited!",
	// 	Recipients: recipients,
	// 	Template:   "invite_company_user.html"})

	return validUsers, nil
}

// func SendCompanyInvitation(companyID string, users []models.User) error {
// 	var recipients []mail.Recipient

// 	frontendUrl, ok := revel.Config.String("url.frontend")
// 	if !ok {
// 		return errors.New("Frontend URL not found")
// 	}

// 	company, opsError := ops.GetCompanyByID(companyID)
// 	if opsError != nil {
// 		return errors.New("500")
// 	}

// 	for _, user := range users {
// 		if user.UserToken != "" {
// 			token, _ := utils.EncodeToJwtToken(jwt.MapClaims{
// 				"userToken": user.UserToken,
// 				"companyID": companyID,
// 				"userID":    user.UserID,
// 			})
// 			recipients = append(recipients, mail.Recipient{
// 				Name:           user.FirstName + " " + user.LastName,
// 				Email:          user.Email,
// 				InviteUserLink: frontendUrl + "/verify-email?token=" + token,
// 				CompanyName:    company.CompanyName,
// 			})
// 		}
// 	}

// 	jobs.Now(mail.SendEmail{
// 		Subject:    "You have been invited!",
// 		Recipients: recipients,
// 		Template:   "invite_company_user.html",
// 	})

// 	return nil
// }

/*
****************
CheckDuplicateEmails()
- Returns array of duplicate email
****************
*/
func CheckDuplicateEmails(users []models.User) []string {
	var unique []models.User
	var duplicates []string
	for _, user := range users {
		skip := false
		for _, u := range unique {
			if user.Email == u.Email {
				duplicates = append(duplicates, u.Email)
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, user)
		}
	}
	return duplicates
}

/*
****************
GetUniqueUsers()
****************
*/
func GetUniqueUsers(users []models.User) []models.User {
	var unique []models.User
	for _, user := range users {
		skip := false
		for _, u := range unique {
			if user.Email == u.Email {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, user)
		}
	}
	return unique
}

/*
*****************
IsEmailUniqueInCompany
Checks if user email is unique in company
******************
*/
func IsEmailUniqueInCompany(email, companyID, searchKey, paramLastEvaluatedKey, limit string, isCompanyAdmin bool, status []string) error {

	if len(status) == 0 {
		status = []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}
	}

	if searchKey != "" {
		searchKey = strings.ToLower(searchKey)
	}

	var pageLimit int64
	if limit != "" {
		pageLimit = utils.ToInt64(limit)
	} else {
		pageLimit = constants.DEFAULT_PAGE_LIMIT
	}

	users, _, err := GetUsersByCompany(searchKey, status, paramLastEvaluatedKey, companyID, pageLimit)

	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return e
	}

	if len(users) != 0 {
		for _, user := range users {
			if user.Email == email {
				// if isCompanyAdmin {
				// 	userRoles, err := GetUserRoles(user.UserID, companyID)
				// 	if err != nil {
				// 		e := errors.New(constants.HTTP_STATUS_500)
				// 		return e
				// 	}
				// 	for _, item := range userRoles {
				// 		if item.RoleName == "Company Admin" {
				// 			e := errors.New(constants.HTTP_STATUS_473)
				// 			return e
				// 		}
				// 	}
				// } else {
				e := errors.New(constants.HTTP_STATUS_473)
				return e
				// }
			}
		}
	}

	return nil
}

/*
****************
IsEmailUnique()
- Returns true or false if email already exists
****************
*/
func IsEmailUnique(email string) error {
	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, email)),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
		TableName: aws.String(app.TABLE_NAME),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return e
	}

	if len(result.Items) != 0 {
		e := errors.New(constants.HTTP_STATUS_473)
		return e
	}

	return nil
}

/*
****************
AddUsersToCompany()
- Insert batch of users to a company
****************
*/
func AddUsersToCompany(companyID string, users []models.User, perfomedBy, handler string, createAccount bool) error {
	// insert batch
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest
	for i, user := range users {
		status := constants.ITEM_STATUS_ACTIVE
		if !createAccount {
			status = constants.ITEM_STATUS_DEFAULT
		}
		if user.UserToken != "DONE" && createAccount {
			status = constants.ITEM_STATUS_PENDING
		}

		assocAccounts := map[string]*dynamodb.AttributeValue{}
		if user.AssociatedAccounts != nil {
			marshal, err := dynamodbattribute.MarshalMap(user.AssociatedAccounts)
			if err != nil {
			}
			assocAccounts = marshal
		}
		// push to batch
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
				"GSI_SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, strings.ToLower(user.SearchKey))),
				},
				"Email": {
					S: aws.String(user.Email),
				},
				"CompanyID": {
					S: aws.String(companyID),
				},
				"UserID": {
					S: aws.String(user.UserID),
				},
				"FirstName": {
					S: aws.String(strings.Title(user.FirstName)),
				},
				"LastName": {
					S: aws.String(strings.Title(user.LastName)),
				},
				"JobTitle": {
					S: aws.String(user.JobTitle),
				},
				"ContactNumber": {
					S: aws.String(user.ContactNumber),
				},
				"DisplayPhoto": {
					S: aws.String(user.DisplayPhoto),
				},
				"SearchKey": {
					S: aws.String(strings.ToLower(user.SearchKey)),
				},
				"UserType": {
					S: aws.String(constants.USER_TYPE_COMPANY_MEMBER),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_COMPANY_MEMBER),
				},
				"CreatedAt": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
				"Handler": {
					S: aws.String(handler),
				},
				"IsTourDone": {
					S: aws.String(constants.BOOL_FALSE),
				},
				"Status": {
					S: aws.String(status),
				},
				"AssociatedAccounts": {
					M: assocAccounts,
				},
			},
		}})
		if i%constants.BATCH_LIMIT == 0 {
			if len(currentBatch) != 0 {
				batches = append(batches, currentBatch)
			}
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		if len(currentBatch) != 0 {
			batches = append(batches, currentBatch)
		}
	}

	_, err := ops.BatchWriteItemHandler(batches)
	if err != nil {
		e := errors.New(err.Error())
		return e
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith and John Doe has been added to CompanyX
	var logInfoUsers []models.LogModuleParams
	for _, user := range users {
		logInfoUsers = append(logInfoUsers, models.LogModuleParams{
			ID: user.UserID,
		})
	}

	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    perfomedBy,
		LogAction: constants.LOG_ACTION_ADD_COMPANY_MEMBERS,
		LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		// result["message"] = "error while creating logs"
	}

	return nil
}

/*
****************
Update User
Updates a user's personal information
user_id (parameter) - used for fetching the user
Body:
first_name
last_name
user_role
****************
*/

// @Summary Update User
// @Description This endpoint updates information for a user by their user ID, returning a 200 OK upon success. If input fields returns an error upon validation, a 422 Unprocessable Content is returned.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.UpdateUserRequest true "Update user body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/:userID [put]
func (c UserController) UpdateUser(userID string) revel.Result {

	data := make(map[string]interface{})

	firstName := utils.TrimSpaces(c.Params.Form.Get("first_name"))
	lastName := utils.TrimSpaces(c.Params.Form.Get("last_name"))
	userRole := c.Params.Form.Get("user_role")
	jobTitle := c.Params.Form.Get("job_title")

	// var roles []stringf
	// json.Unmarshal([]byte(userRole), &roles)

	//Validate Form Body
	validateForm := models.User{
		UserID:    userID,
		FirstName: firstName,
		LastName:  lastName,
		// UserRole:		roles,
		JobTitle: jobTitle,
	}

	validateForm.Validate(c.Validation, constants.SERVICE_TYPE_UPDATE_USER)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	company, err1 := ops.GetActiveCompany(userID)
	if err1 != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}

	//Validate User Role
	// comment temporarily
	// if !utils.ComparingSlices(roles, constraints.GROUP_ROLES) {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_465)
	// 	return c.RenderJSON(data)
	// }

	// rolePermissions, err := dynamodbattribute.MarshalList(roles)
	// if err != nil {
	// 	data["error"] 	= err.Error()
	// 	data["status"] 	= utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(data)
	// }

	searchKey := firstName + " " + lastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":fn": {
				S: aws.String(firstName),
			},
			":ln": {
				S: aws.String(lastName),
			},
			":jt": {
				S: aws.String(jobTitle),
			},
			":sk": {
				S: aws.String(searchKey),
			},
			// ":ur": {
			// 	L: rolePermissions,
			// },
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, company.ActiveCompany)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET FirstName = :fn, LastName = :ln, JobTitle = :jt, UpdatedAt = :ua, SearchKey = :sk"),
	}

	userChanges := models.LogModuleParams{
		Old: models.User{
			UserID:    user.UserID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			JobTitle:  user.JobTitle,
			SearchKey: user.SearchKey,
		},
		New: models.User{
			UserID:    user.UserID,
			FirstName: firstName,
			LastName:  lastName,
			JobTitle:  jobTitle,
			SearchKey: searchKey,
		},
	}

	userLogInfo := &models.LogModuleParams{
		ID:   user.UserID,
		Name: user.FirstName + " " + user.LastName,
		Old:  userChanges.Old,
		New:  userChanges.New,
	}

	logID, err := c.CreateUserLog(userLogInfo, company.ActiveCompany, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	_, err3 := app.SVC.UpdateItem(input)
	if err3 != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["logID"] = logID

	// update user roles
	var roles []string
	json.Unmarshal([]byte(userRole), &roles)

	userRoles, _ := GetUserRoles(userID, company.ActiveCompany)
	var deletedRoles []models.Role

	for _, r := range userRoles {
		if utils.StringInSlice(r.RoleID, roles) {
			utils.RemoveStringInSlice(roles, r.RoleID)
		} else {
			deletedRoles = append(deletedRoles, models.Role{
				PK: utils.AppendPrefix(constants.PREFIX_USER, userID),
				SK: utils.AppendPrefix(constants.PREFIX_ROLE, r.RoleID),
			})
		}
	}

	// insert new roles
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, role := range roles {
		// push to batch
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, role), utils.AppendPrefix(constants.PREFIX_COMPANY, company.ActiveCompany))),
				},
				"UserID": {
					S: aws.String(userID),
				},
				"RoleID": {
					S: aws.String(role),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_USER_ROLE),
				},
				"CreatedAt": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
		}})
		if i%constants.BATCH_LIMIT == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}

	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		batches = append(batches, currentBatch)
	}

	_, err = ops.BatchWriteItemHandler(batches)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// remove roles
	batches = nil
	currentBatch = nil

	for i, role := range deletedRoles {
		// push to batch
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(role.PK),
				},
				"SK": {
					S: aws.String(role.SK),
				},
			},
		}})
		if i%constants.BATCH_LIMIT == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}

	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		batches = append(batches, currentBatch)
	}

	_, err = ops.BatchWriteItemHandler(batches)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)

}

/*
**********************
UpdateUserInfo
Body:
user_id
first_name
last_name
contact_number
job_title
****
*/

func updateUserAndCompanyUserEntity(input *dynamodb.UpdateItemInput, PK, SK string) (*dynamodb.UpdateItemOutput, error) {
	input.Key = map[string]*dynamodb.AttributeValue{
		"PK": {
			S: aws.String(PK),
		},
		"SK": {
			S: aws.String(SK),
		},
	}

	updateItemRes, updateItemErr := app.SVC.UpdateItem(input)
	return updateItemRes, updateItemErr
}

// @Summary Update User Info
// @Description This endpoint updates additional information for a user, returning a 200 OK upon success. If input fields returns an error upon validation, a 422 Unprocessable Content is returned.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.UpdateUserInfoRequest true "Update user info body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/profile/updateinfo/:userID [put]
func (c UserController) UpdateUserInfo(userID string) revel.Result {

	data := make(map[string]interface{})
	firstName := utils.TrimSpaces(c.Params.Form.Get("first_name"))
	lastName := utils.TrimSpaces(c.Params.Form.Get("last_name"))
	contactNumber := utils.TrimSpaces(c.Params.Form.Get("contact_number"))
	companyID := c.ViewArgs["companyID"].(string)
	jobTitle := utils.TrimSpaces(c.Params.Form.Get("job_title"))
	secondaryEmail := c.Params.Form.Get("secondary_email")
	// image := c.Params.Files["display_photo"]

	//Validate Form Body
	validateForm := models.User{
		UserID:         userID,
		FirstName:      firstName,
		LastName:       lastName,
		JobTitle:       jobTitle,
		SecondaryEmail: secondaryEmail,
	}

	validateForm.Validate(c.Validation, constants.SERVICE_TYPE_UPDATE_USER)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	user, getUserErr := ops.GetUserByID(userID)
	if getUserErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}
	companyUser, getUserErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
		CompanyID: companyID,
	}, c.Controller)
	if getUserErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}
	//This is to handle empty value for old accounts
	if companyUser.FirstName == "" {
		companyUser.FirstName = firstName
	}
	if companyUser.LastName == "" {
		companyUser.LastName = lastName
	}
	if companyUser.JobTitle == "" {
		companyUser.JobTitle = jobTitle
	}
	if companyUser.ContactNumber == "" {
		companyUser.ContactNumber = contactNumber
	}
	if companyUser.DisplayPhoto == "" {
		companyUser.DisplayPhoto = user.DisplayPhoto
	}
	if companyUser.SecondaryEmail == "" {
		companyUser.SecondaryEmail = secondaryEmail
	}
	companyUser.Email = user.Email
	companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email

	displayPhoto := companyUser.DisplayPhoto

	if len(displayPhoto) == 0 {
		displayPhoto = ""
	}

	searchKey := firstName + " " + lastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)
	caser := cases.Title(language.English)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":fn": {
				S: aws.String(caser.String(firstName)),
			},
			":ln": {
				S: aws.String(caser.String(lastName)),
			},
			":cn": {
				S: aws.String(contactNumber),
			},
			":sk": {
				S: aws.String(searchKey),
			},
			":dp": {
				S: aws.String(displayPhoto),
			},
			":jt": {
				S: aws.String(jobTitle),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
			":se": {
				S: aws.String(secondaryEmail),
			},
			":gsisk": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, searchKey)),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		// Key: map[string]*dynamodb.AttributeValue{
		// 	"PK": {
		// 		S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
		// 	},
		// 	"SK": {
		// 		S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
		// 	},
		// },
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET FirstName = :fn, LastName = :ln, ContactNumber = :cn, DisplayPhoto = :dp, JobTitle = :jt, UpdatedAt = :ua, SearchKey = :sk, SecondaryEmail = :se, GSI_SK = :gsisk"),
	}

	//* Update user info
	err := c.UserOps.UpdatedUserInfo(ops.UpdateUserInfoParams{
		FirstName:      firstName,
		LastName:       lastName,
		ContactNumber:  contactNumber,
		SearchKey:      searchKey,
		DisplayPhoto:   displayPhoto,
		JobTitle:       jobTitle,
		SecondaryEmail: secondaryEmail,
		UserID:         userID,
		Email:          user.Email,
	})

	if err != nil {
		c.Response.Status = http.StatusNotAcceptable
		return c.RenderJSON(err.Error())
	}

	//* create log for user
	userChanges := models.LogModuleParams{
		Old: models.User{
			UserID:         user.UserID,
			FirstName:      companyUser.FirstName,
			LastName:       companyUser.LastName,
			ContactNumber:  companyUser.ContactNumber,
			SearchKey:      companyUser.SearchKey,
			JobTitle:       companyUser.JobTitle,
			DisplayPhoto:   user.DisplayPhoto,
			SecondaryEmail: user.SecondaryEmail,
		},
		New: models.User{
			UserID:         user.UserID,
			FirstName:      firstName,
			LastName:       lastName,
			ContactNumber:  contactNumber,
			SearchKey:      searchKey,
			JobTitle:       jobTitle,
			DisplayPhoto:   displayPhoto,
			SecondaryEmail: secondaryEmail,
		},
	}

	userLogInfo := &models.LogModuleParams{
		ID:   companyID,
		Name: companyUser.FirstName + " " + companyUser.LastName,
		Old:  userChanges.Old,
		New:  userChanges.New,
	}

	logID, err := c.CreateUserLog(userLogInfo, user.UserID, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	//* create log of company user
	companyUserChanges := models.LogModuleParams{
		Old: models.CompanyUser{
			UserID:        user.UserID,
			FirstName:     companyUser.FirstName,
			LastName:      companyUser.LastName,
			ContactNumber: companyUser.ContactNumber,
			SearchKey:     companyUser.SearchKey,
			JobTitle:      companyUser.JobTitle,
			DisplayPhoto:  companyUser.DisplayPhoto,
		},
		New: models.CompanyUser{
			UserID:        user.UserID,
			FirstName:     firstName,
			LastName:      lastName,
			ContactNumber: contactNumber,
			SearchKey:     searchKey,
			JobTitle:      jobTitle,
			DisplayPhoto:  displayPhoto,
		},
	}

	companyUserLogInfo := &models.LogModuleParams{
		ID:   companyID,
		Name: user.FirstName + " " + user.LastName,
		Old:  companyUserChanges.Old,
		New:  companyUserChanges.New,
	}

	logCUID, err := c.CreateUserLog(companyUserLogInfo, user.UserID, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	//* update User model
	// updateUserInput := *input
	// userPK := utils.AppendPrefix(constants.PREFIX_USER, user.UserID)
	// userSK := utils.AppendPrefix(constants.PREFIX_USER, user.Email)
	// updateItemRes, updateItemErr := updateUserAndCompanyUserEntity(&updateUserInput, userPK, userSK)
	// if updateItemErr != nil {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(data)
	// }

	//* Update Active Company User model
	updateCompanyUserInput := *input
	var updateCUOutput *dynamodb.UpdateItemOutput
	companyUserPK := utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)
	companyUserSK := utils.AppendPrefix(constants.PREFIX_USER, user.UserID)
	updateCU, updateCUItemErr := updateUserAndCompanyUserEntity(&updateCompanyUserInput, companyUserPK, companyUserSK)
	if updateCUItemErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}
	updateCUOutput = updateCU

	// users, getUserErrr := ops.GetUserByID(userID)
	// if getUserErrr != nil {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
	// 	return c.RenderJSON(data)
	// }
	//* update tmpUser cache.
	// newUser := getUserInTmp(userID)
	// if newUser.UserID != "" {
	// 	newUser.FirstName = firstName
	// 	newUser.LastName = lastName
	// 	newUser.JobTitle = jobTitle
	// 	newUser.ContactNumber = contactNumber
	// 	newUser.SearchKey = searchKey
	// 	updateUserInfoInTmp(&newUser)
	// }

	// if updateCU.Attributes != nil {
	// 	attributes := models.User{}
	// 	err = dynamodbattribute.UnmarshalMap(updateCU.Attributes, &attributes)
	// 	if attributes.DisplayPhoto != "" {
	// 		photo := utils.ChangeCDNImageSize(attributes.DisplayPhoto, constants.IMAGE_SUFFIX_100)
	// 		fileName, err := cdn.GetImageFromStorage(photo)
	// 		if err == nil {
	// 			attributes.DisplayPhoto = fileName
	// 		}
	// 	}
	// 	if err == nil {
	// 		data["attributes"] = attributes
	// 	}
	// }

	//* update tmpCompanyUser cache
	newCompanyUser := getCompanyMemberInTmp(companyID, userID, c.Controller)
	if newCompanyUser.UserID != "" {
		newCompanyUser.FirstName = firstName
		newCompanyUser.LastName = lastName
		newCompanyUser.JobTitle = jobTitle
		newCompanyUser.ContactNumber = contactNumber
		newCompanyUser.SearchKey = searchKey
		newCompanyUser.SecondaryEmail = secondaryEmail
		updateCompanyUserInfoInTmp(&newCompanyUser)
	}

	if updateCUOutput.Attributes != nil {
		attributes := models.CompanyUser{}
		err = dynamodbattribute.UnmarshalMap(updateCUOutput.Attributes, &attributes)
		if attributes.DisplayPhoto != "" {
			photo := utils.ChangeCDNImageSize(attributes.DisplayPhoto, constants.IMAGE_SUFFIX_100)
			fileName, err := cdn.GetImageFromStorage(photo)
			if err == nil {
				attributes.DisplayPhoto = fileName
			}
		}
		if err == nil {
			data["attributes"] = attributes
		}
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith's information was updated.
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_USER,
		LogType:   constants.ENTITY_TYPE_USER,
		LogInfo: &models.LogInformation{
			User: &models.LogModuleParams{
				ID: user.UserID,
			},
		},
	})

	_, err = CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}
	data["logs"] = userLogInfo
	data["users"] = newCompanyUser
	data["logID"] = logID
	data["logCUID"] = logCUID
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
Update User Photo - Put
Updates a user's personal information
user_id (parameter) - used for fetching the user
Body:
display_photo
****************
*/

// @Summary Update User Photo
// @Description This endpoint updates the profile photo of a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body formData file true "Profile Photo"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/profile/photo/:userID [patch]
func (c UserController) UpdateUserPhoto(userID string) revel.Result {

	data := make(map[string]interface{})

	// userID := c.ViewArgs["userID"].(string)
	// userID := c.Params.Query.Get("user_id")
	companyID := c.ViewArgs["companyID"].(string)
	image := c.Params.Files["display_photo"]
	//

	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}

	// user, getUserErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
	// 	UserID:    userID,
	// 	CompanyID: companyID,
	// }, c.Controller)
	// if getUserErr != nil {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
	// 	return c.RenderJSON(data)
	// }

	displayPhoto := user.DisplayPhoto

	if len(image) <= 4 || image == nil {
		displayPhoto = ""
	}

	if image == nil {
		if user.DisplayPhoto != "" {
			// remove
			_, err := cdn.DeleteS3Object(user.DisplayPhoto)
			if err != nil {
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
				return c.RenderJSON(data)
			}
		}
	}
	if len(image) > 0 {
		photo, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_USER)
		displayPhoto = photo
		if err != nil {
			//
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":dp": {
				S: aws.String(displayPhoto),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
		// UpdateExpression: aws.String("SET FirstName = :fn, LastName = :ln, ContactNumber = :cn, DisplayPhoto = :dp, UpdatedAt = :ua, SearchKey = :sk"),
		// UpdateExpression: aws.String("SET DisplayPhoto = :dp, UpdatedAt = :ua, SearchKey = :sk"),
		UpdateExpression: aws.String("SET DisplayPhoto = :dp, UpdatedAt = :ua"),
	}

	userChanges := models.LogModuleParams{
		Old: models.User{
			UserID:       user.UserID,
			DisplayPhoto: user.DisplayPhoto,
		},
		New: models.User{
			UserID:       user.UserID,
			DisplayPhoto: displayPhoto,
		},
	}

	userLogInfo := &models.LogModuleParams{
		ID:   companyID,
		Name: user.FirstName + " " + user.LastName,
		Old:  userChanges.Old,
		New:  userChanges.New,
	}

	logID, err := c.CreateUserLog(userLogInfo, user.UserID, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	updateItemResult, updateItemErr := app.SVC.UpdateItem(input)
	if updateItemErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// user, getUserErrr := ops.GetUserByID(userID)
	// if getUserErrr != nil {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
	// 	return c.RenderJSON(data)
	// }

	if updateItemResult.Attributes != nil {
		updatedAttributes := models.User{}
		err = dynamodbattribute.UnmarshalMap(updateItemResult.Attributes, &updatedAttributes)
		if err == nil {
			if image == nil {
				updatedAttributes.DisplayPhoto = ""
				data["isRemovePhoto"] = true

			} else {
				resizedPhoto := utils.ChangeCDNImageSize(updatedAttributes.DisplayPhoto, constants.IMAGE_SUFFIX_100)
				fileName, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					updatedAttributes.DisplayPhoto = fileName
				}
			}
			data["updatedAttributes"] = updatedAttributes
		}
	}

	// data["user"] = user
	data["logID"] = logID
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(data)
}

/*
****************
ORIGINAL
Update User Profile - Put
Updates a user's personal information
user_id (parameter) - used for fetching the user
Body:
first_name
last_name
contact_number
display_photo
****************
*/

// @Summary Update User Profile
// @Description This endpoint updates the profile information for a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.UpdateUserProfileRequest true "Update user profile body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/profile/:userID [put]
func (c UserController) UpdateUserProfile(userID string) revel.Result {

	data := make(map[string]interface{})
	firstName := utils.TrimSpaces(c.Params.Form.Get("first_name"))
	lastName := utils.TrimSpaces(c.Params.Form.Get("last_name"))
	contactNumber := utils.TrimSpaces(c.Params.Form.Get("contact_number"))
	companyID := c.ViewArgs["companyID"].(string)
	image := c.Params.Files["display_photo"]

	// user, getUserErr := ops.GetUserByID(userID)
	user, getUserErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
		CompanyID: companyID,
	}, c.Controller)
	if getUserErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}

	displayPhoto := user.DisplayPhoto

	if len(image) <= 4 {
		displayPhoto = ""
	}

	if image != nil {
		photo, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_USER)
		displayPhoto = photo
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
	}

	searchKey := firstName + " " + lastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":fn": {
				S: aws.String(firstName),
			},
			":ln": {
				S: aws.String(lastName),
			},
			":cn": {
				S: aws.String(contactNumber),
			},
			":sk": {
				S: aws.String(searchKey),
			},
			":dp": {
				S: aws.String(displayPhoto),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET FirstName = :fn, LastName = :ln, ContactNumber = :cn, DisplayPhoto = :dp, UpdatedAt = :ua, SearchKey = :sk"),
	}

	userChanges := models.LogModuleParams{
		Old: models.User{
			UserID:        user.UserID,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			ContactNumber: user.ContactNumber,
			SearchKey:     user.SearchKey,
			DisplayPhoto:  user.DisplayPhoto,
		},
		New: models.User{
			UserID:        user.UserID,
			FirstName:     firstName,
			LastName:      lastName,
			ContactNumber: contactNumber,
			SearchKey:     searchKey,
			DisplayPhoto:  displayPhoto,
		},
	}

	userLogInfo := &models.LogModuleParams{
		ID:   companyID,
		Name: user.FirstName + " " + user.LastName,
		Old:  userChanges.Old,
		New:  userChanges.New,
	}

	logID, err := c.CreateUserLog(userLogInfo, user.UserID, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	_, updateItemErr := app.SVC.UpdateItem(input)
	if updateItemErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["logID"] = logID
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
Update User Status
Changes a user's status (ACTIVE, INACTIVE, DELETED)
Body:
user_ids
status
****************
*/

// @Summary Update User Status
// @Description This endpoint updates the status of a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.UpdateUserStatusRequest true "Update user status body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/status [patch]
func (c UserController) UpdateUserStatus() revel.Result {

	data := make(map[string]interface{})

	userIDs := c.Params.Form.Get("user_ids")
	userStatus := c.Params.Form.Get("status")
	companyID := c.ViewArgs["companyID"].(string)

	var unmarshalIDs []string
	json.Unmarshal([]byte(userIDs), &unmarshalIDs)

	if len(unmarshalIDs) == 0 {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	// validate status if valid
	if !utils.StringInSlice(strings.ToUpper(userStatus), constraints.ITEM_STATUS) {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_466)
		return c.RenderJSON(data)
	}

	var users []models.User

	// check if exists
	for _, id := range unmarshalIDs {
		user, err := ops.GetUserByID(id)
		if user.UserID == "" || err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
			return c.RenderJSON(data)
		}
		users = append(users, user)
	}

	// update
	for _, user := range users {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":cs": {
					S: aws.String(userStatus),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
				},
			},
			ExpressionAttributeNames: map[string]*string{
				"#s": aws.String("Status"),
			},
			ReturnValues:     aws.String("UPDATED_NEW"),
			UpdateExpression: aws.String("SET #s = :cs, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
	}

	//remove company member
	for _, user := range users {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":cs": {
					S: aws.String(userStatus),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, user.ActiveCompany)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
			},
			ExpressionAttributeNames: map[string]*string{
				"#s": aws.String("Status"),
			},
			ReturnValues:     aws.String("UPDATED_NEW"),
			UpdateExpression: aws.String("SET #s = :cs, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith and John Doe has been added to CompanyX
	var logInfoUsers []models.LogModuleParams
	for _, user := range users {
		logInfoUsers = append(logInfoUsers, models.LogModuleParams{
			ID: user.UserID,
		})
	}
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_COMPANY_MEMBERS,
		LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	})
	_, err := CreateBatchLog(logs)
	if err != nil {
		// result["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(data)
}

/*
****************
Update User Email
Changes a user's email
Body:
email
****************
*/

// @Summary Update User Email
// @Description This endpoint updates the email address of a user, returning a 200 OK upon success. If input fields returns an error upon validation, a 422 Unprocessable Content is returned.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.UpdateUserEmailRequest true "Update user email body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/email/:userID [patch]
func (c UserController) UpdateUserEmail(userID string) revel.Result {

	data := make(map[string]interface{})

	email := c.Params.Form.Get("email")
	companyID := c.ViewArgs["companyID"].(string)

	//Validate Form Body
	validateForm := models.User{
		Email: email,
	}

	validateForm.Validate(c.Validation, constants.SERVICE_TYPE_UPDATE_USER_EMAIL)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	user, err1 := ops.GetUserByID(userID)
	if err1 != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
		return c.RenderJSON(data)
	}

	PK := constants.PREFIX["user"] + user.UserID
	SK := constants.PREFIX["user"] + user.Email

	_, deleteErr := ops.DeleteByPartitionKey(PK, SK)
	if deleteErr != nil {
		data["error"] = "Got error calling DeleteItem"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	newSearchKey := user.FirstName + " " + user.LastName + " " + email
	newSearchKey = strings.ToLower(newSearchKey)

	// PutItem
	newUser := models.User{
		PK:        utils.AppendPrefix(constants.PREFIX_USER, user.UserID),
		SK:        utils.AppendPrefix(constants.PREFIX_USER, email),
		UserID:    user.UserID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     email,
		Password:  user.Password,
		Salt:      user.Salt,
		UserRole:  user.UserRole,
		SearchKey: strings.ToLower(newSearchKey),
		Status:    constants.ITEM_STATUS_ACTIVE,
		CreatedAt: user.CreatedAt,
		UpdatedAt: utils.GetCurrentTimestamp(),
		Type:      constants.ENTITY_TYPE_USER,
	}

	av, err := dynamodbattribute.MarshalMap(newUser)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(data)
	}

	putInput := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	userChanges := models.LogModuleParams{
		Old: models.User{
			UserID: user.UserID,
			Email:  user.Email,
		},
		New: models.User{
			UserID: user.UserID,
			Email:  email,
		},
	}

	userLogInfo := &models.LogModuleParams{
		ID:  user.UserID,
		Old: userChanges.Old,
		New: userChanges.New,
	}

	logID, err := c.CreateUserLog(userLogInfo, companyID, constants.ACTION_UPDATE)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		data["message"] = "Operation is success but error while creating logs."
		return c.RenderJSON(data)
	}

	data["logID"] = logID

	_, err = app.SVC.PutItem(putInput)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	data["data"] = user

	return c.RenderJSON(data)
}

/*
****************
UpdateUserRole()
Update a roles permission
Body:
userId - required
userEmail - required
roleId - array - required
****************
*/

// @Summary Update User Role
// @Description This endpoint updates the role of a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.UpdateUserRoleRequest true "Update user role body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/role [patch]
func (c UserController) UpdateUserRole() revel.Result {
	userId := c.Params.Form.Get("user_id")
	userEmail := c.Params.Form.Get("email")
	roleId := []string{c.Params.Form.Get("role_id")}

	//Get current timestamp
	currentTime := utils.GetCurrentTimestamp()

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	rolePermissions, err := dynamodbattribute.MarshalList(roleId)
	if err != nil {
		data["error"] = err.Error()
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
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
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userId)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userEmail)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET UserRole = :pc, UpdatedAt = :ua"),
	}

	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		data["error"] = err.Error()
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/******************************
HELPER FUNCTIONS
******************************/

/*
*****************************
Get Users By Company
Used for fetching users within a certain company
Parameters:
exclusiveStartKey
companyID
pageLimit
*****************************
*/
func GetUsersByCompany(searchKey string, status []string, exclusiveStartKey, companyID string, pageLimit int64) ([]models.CompanyUser, models.CompanyUser, error) {

	users := []models.CompanyUser{}
	users2 := []models.CompanyUser{}
	lastEvaluatedKey := models.CompanyUser{}

	queryFilter := map[string]*dynamodb.Condition{}
	if len(status) != 0 {
		s, err := dynamodbattribute.MarshalList(status)
		if err != nil {
			// )
		}
		queryFilter = map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: s,
			},
		}
	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
					},
				},
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_USER),
					},
				},
			},
		},
		QueryFilter:      queryFilter,
		ScanIndexForward: aws.Bool(false),
		// Limit:             aws.Int64(pageLimit),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
	}

	if searchKey == "" {
		params.Limit = aws.Int64(pageLimit)
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		//
		// )
		e := errors.New(constants.HTTP_STATUS_500)
		return users, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	// if len(key) != 0 {
	// for len(key) != 0 && (len(result.Items) < int(pageLimit) || searchKey != "") {
	for len(key) != 0 {
		params.ExclusiveStartKey = key
		queryResult, err := app.SVC.Query(params)
		if err != nil {
			break
		}
		result.Items = append(result.Items, queryResult.Items...)
		key = queryResult.LastEvaluatedKey
	}
	// }

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, lastEvaluatedKey, e
	}

	for _, user := range users {
		userData, err := ops.GetUserData(user.UserID, searchKey)
		if err != nil {
			e := errors.New(err.Error())
			return users, lastEvaluatedKey, e
		}

		// if userData.UserID != "" && (userData.Status == status || userData.Status == constants.ITEM_STATUS_PENDING) {
		if userData.UserID != "" {
			// include user photo
			displayPhoto := ""
			if userData.DisplayPhoto != "" {
				resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
				file, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					displayPhoto = file
				}
			}
			// append users
			users2 = append(users2, models.CompanyUser{
				PK:        userData.PK,
				SK:        userData.SK,
				UserID:    userData.UserID,
				FirstName: userData.FirstName,
				LastName:  userData.LastName,
				Email:     userData.Email,
				// Status:       userData.Status,
				Status:       user.Status,
				UserRole:     userData.UserRole,
				JobTitle:     userData.JobTitle,
				CreatedAt:    userData.CreatedAt,
				DisplayPhoto: displayPhoto,
				DeletedAt:    user.DeletedAt,
				Origin:       user.Origin,
			})
		}
	}

	return users2, lastEvaluatedKey, nil
}

// Refactored version of GetUsersByCompany()
type GetCompanyUsersInput struct {
	CompanyID        string
	SearchKey        string
	Status           []string
	LastEvaluatedKey string
	Include          string
	GroupID          string
}

func GetCompanyUsersNewWithLimit(input GetCompanyUsersInput) ([]models.CompanyUser, models.CompanyUser, error) {
	lastEvaluatedKey := models.CompanyUser{}
	users := []models.CompanyUser{}
	status, err := dynamodbattribute.MarshalList(input.Status)
	if err != nil {
		return users, lastEvaluatedKey, err
	}

	queryFilter := map[string]*dynamodb.Condition{
		"Status": {
			ComparisonOperator: aws.String(constants.CONDITION_IN),
			AttributeValueList: status,
		},
	}

	var params *dynamodb.QueryInput
	// if searchKey has value remove the limit
	// check if input.GroupID is passed, for add group members to show member without limit.
	if input.GroupID != "" {
		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
			KeyConditions: map[string]*dynamodb.Condition{
				"CompanyID": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(input.CompanyID),
						},
					},
				},
				"GSI_SK": {
					ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(constants.PREFIX_USER),
						},
					},
				},
			},
			QueryFilter: queryFilter,
		}
	} else {
		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
			KeyConditions: map[string]*dynamodb.Condition{
				"CompanyID": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(input.CompanyID),
						},
					},
				},
				"GSI_SK": {
					ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(constants.PREFIX_USER),
						},
					},
				},
			},
			QueryFilter: queryFilter,
			Limit:       aws.Int64(constants.DEFAULT_PAGE_LIMIT),
		}
	}

	if input.SearchKey != "" {
		params.QueryFilter["SearchKey"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(input.SearchKey),
				},
			},
		}
	}

	params.ExclusiveStartKey = utils.MarshalLastEvaluatedKey(models.CompanyUser{}, input.LastEvaluatedKey)

	res, err := ops.HandleQueryWithLimit(params, constants.DEFAULT_PAGE_LIMIT, false)
	if err != nil {
		return nil, lastEvaluatedKey, err
	}

	key := res.LastEvaluatedKey
	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		return nil, lastEvaluatedKey, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &users)
	if err != nil {
		return nil, lastEvaluatedKey, err
	}

	return users, lastEvaluatedKey, nil
}

func GetCompanyUsersNew(input GetCompanyUsersInput) ([]models.CompanyUser, error) {
	users := []models.CompanyUser{}

	status, err := dynamodbattribute.MarshalList(input.Status)
	if err != nil {
		return users, err
	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, input.CompanyID)),
					},
				},
			},
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(input.CompanyID),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_USER),
					},
				},
			},
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: status,
			},
		},
		ScanIndexForward: aws.Bool(false),
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return users, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		return users, err
	}

	// for _, user := range users {
	// 	userData, err := ops.GetUserData(user.UserID, searchKey)
	// 	if err != nil {
	// 		e := errors.New(err.Error())
	// 		return users, lastEvaluatedKey, e
	// 	}

	// 	// if userData.UserID != "" && (userData.Status == status || userData.Status == constants.ITEM_STATUS_PENDING) {
	// 	if userData.UserID != "" {
	// 		// include user photo
	// 		displayPhoto := ""
	// 		if userData.DisplayPhoto != "" {
	// 			resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
	// 			file, err := cdn.GetImageFromStorage(resizedPhoto)
	// 			if err == nil {
	// 				displayPhoto = file
	// 			}
	// 		}
	// 		// append users
	// 		users2 = append(users2, models.CompanyUser{
	// 			PK:        userData.PK,
	// 			SK:        userData.SK,
	// 			UserID:    userData.UserID,
	// 			FirstName: userData.FirstName,
	// 			LastName:  userData.LastName,
	// 			Email:     userData.Email,
	// 			// Status:       userData.Status,
	// 			Status:       user.Status,
	// 			UserRole:     userData.UserRole,
	// 			JobTitle:     userData.JobTitle,
	// 			CreatedAt:    userData.CreatedAt,
	// 			DisplayPhoto: displayPhoto,
	// 			DeletedAt:    user.DeletedAt,
	// 			Origin:       user.Origin,
	// 		})
	// 	}
	// }

	return users, nil
}

// Refactored version of GetUsersByCompany()
func GetCompanyUsersInformation(companyID string, companyUsers []models.CompanyUser, controller *revel.Controller) ([]models.CompanyUser, error) {
	var users []models.CompanyUser
	for _, user := range companyUsers {
		u := getCompanyMemberInTmp(companyID, user.UserID, controller)
		if u.UserID != "" {
			userData, err := ops.GetUserByIDNew(user.UserID)
			_ = err
			u.FirstName = user.FirstName
			u.LastName = user.LastName
			u.JobTitle = user.JobTitle
			u.ContactNumber = user.ContactNumber
			u.DisplayPhoto = user.DisplayPhoto
			//This is to handle empty value for old accounts
			if u.FirstName == "" {
				u.FirstName = userData.FirstName
			}
			if u.LastName == "" {
				u.LastName = userData.LastName
			}
			if u.JobTitle == "" {
				u.JobTitle = userData.JobTitle
			}
			if u.ContactNumber == "" {
				u.ContactNumber = userData.ContactNumber
			}
			if u.DisplayPhoto == "" {
				u.DisplayPhoto = userData.DisplayPhoto
			}
			u.Email = userData.Email

			u.SearchKey = u.FirstName + " " + u.LastName + " " + userData.Email

			if len(u.DisplayPhoto) != 0 {
				resizedPhoto := utils.ChangeCDNImageSize(u.DisplayPhoto, "_100")
				fileName, err := cdn.GetImageFromStorage(resizedPhoto)
				if err != nil {
					return nil, err
				}
				u.DisplayPhoto = fileName
			}

			//DisplayPhoto if empty, set value to empty string -gcm
			if len(u.DisplayPhoto) == 0 {
				u.DisplayPhoto = ""
			}
			u.Email = userData.Email
			u.ActiveCompany = companyID
			u.SearchKey = user.SearchKey
			u.Origin = user.Origin
			u.Status = user.Status
			u.DeletedAt = user.DeletedAt
			u.UserRole = userData.UserRole
			// u.AssociatedAccounts = user.Associ	atedAccounts
			users = append(users, u)
		}
	}

	return users, nil
}

/*
*****************************
Get Users By Group
Used for fetching users within a certain group
Parameters:
exclusiveStartKey
groupID
pageLimit
*****************************
*/
func GetUsersByGroup(searchKey, status, exclusiveStartKey, groupID, companyID string, departmentID string, pageLimit int64) ([]models.CompanyUser, models.CompanyUser, error) {

	users := []models.GroupMember{}
	users2 := []models.CompanyUser{}
	lastEvaluatedKey := models.CompanyUser{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
					},
				},
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_USER),
					},
				},
			},
		},
		Limit:             aws.Int64(pageLimit),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return nil, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return nil, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	// if len(key) != 0 {
	// for len(key) != 0 && (len(result.Items) < int(pageLimit) || searchKey != "") {
	for len(key) != 0 {
		params.ExclusiveStartKey = key
		queryResult, err := app.SVC.Query(params)
		if err != nil {
			break
		}
		result.Items = append(result.Items, queryResult.Items...)
		key = queryResult.LastEvaluatedKey
	}
	// }

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return nil, lastEvaluatedKey, e
	}

	for _, user := range users {
		userData, err := ops.GetUserData(user.MemberID, searchKey)
		if err != nil {
			e := errors.New(err.Error())
			return nil, lastEvaluatedKey, e
		}

		// get company user status
		var userStatus string
		var userOrigin string
		userCompany, err := GetCompanyUser(companyID, userData.UserID)
		if err == nil {
			userStatus = userCompany.Status
			userOrigin = userCompany.Origin
			//  ", userStatus)
		}

		if userData.UserID != "" && userStatus != constants.ITEM_STATUS_DELETED {
			if userData.Status == "" {
				userData.Status = constants.ITEM_STATUS_PENDING
			}

			// include user photo
			displayPhoto := ""
			if userData.DisplayPhoto != "" {
				resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
				file, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					displayPhoto = file
				}
			}

			users2 = append(users2, models.CompanyUser{
				PK:        userData.PK,
				SK:        userData.SK,
				UserID:    userData.UserID,
				FirstName: userData.FirstName,
				LastName:  userData.LastName,
				Email:     userData.Email,
				// Status:    userData.Status,
				DisplayPhoto: displayPhoto,
				Status:       userStatus,
				UserRole:     userData.UserRole,
				CreatedAt:    userData.CreatedAt,
				Origin:       userOrigin,
			})
		}
	}

	return users2, lastEvaluatedKey, nil
}

/*
*****************************
Get Users By Department
Used for fetching users within a certain department
Parameters:
exclusiveStartKey
departmentID
pageLimit
*****************************
*/
func GetUsersByDepartment(searchKey, status, exclusiveStartKey, companyID string, departmentID string, pageLimit int64) ([]models.CompanyUser, models.CompanyUser, error) {

	users := []models.CompanyUser{}
	users2 := []models.CompanyUser{}
	lastEvaluatedKey := models.CompanyUser{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID)),
					},
				},
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_USER),
					},
				},
			},
		},
		Limit:             aws.Int64(pageLimit),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return users, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	// if len(key) != 0 {
	// for len(key) != 0 && (len(result.Items) < int(pageLimit) || searchKey != "") {
	for len(key) != 0 {
		params.ExclusiveStartKey = key
		queryResult, err := app.SVC.Query(params)
		if err != nil {
			break
		}
		result.Items = append(result.Items, queryResult.Items...)
		key = queryResult.LastEvaluatedKey
	}
	// }

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, lastEvaluatedKey, e
	}

	for _, user := range users {
		userData, err := ops.GetUserData(user.UserID, searchKey)
		if err != nil {
			e := errors.New(err.Error())
			return users, lastEvaluatedKey, e
		}

		//get user company status
		var userStatus string
		var userOrigin string
		userCompany, err := GetCompanyUser(companyID, user.UserID)
		if err == nil {
			userStatus = userCompany.Status
			userOrigin = userCompany.Origin
			// : ", userStatus)
		}

		if userData.UserID != "" && userStatus != constants.ITEM_STATUS_DELETED && userStatus != "" {
			if userData.Status == "" {
				userData.Status = constants.ITEM_STATUS_PENDING
			}

			// include user photo
			displayPhoto := ""
			if userData.DisplayPhoto != "" {
				resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
				file, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					displayPhoto = file
				}
			}

			users2 = append(users2, models.CompanyUser{
				PK:        userData.PK,
				SK:        userData.SK,
				UserID:    userData.UserID,
				FirstName: userData.FirstName,
				LastName:  userData.LastName,
				Email:     userData.Email,
				// Status:    userData.Status,
				DisplayPhoto: displayPhoto,
				Status:       userStatus,
				UserRole:     userData.UserRole,
				Origin:       userOrigin,
			})
		}
	}

	return users2, lastEvaluatedKey, nil
}

func HandleQueryLimit(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	result, err := app.SVC.Query(input)
	if err != nil {
		return &dynamodb.QueryOutput{}, err
	}
	key := result.LastEvaluatedKey
	for len(key) != 0 {
		input.ExclusiveStartKey = key
		r, err := app.SVC.Query(input)
		if err != nil {
			break
		}
		result.Items = append(result.Items, r.Items...)
		key = result.LastEvaluatedKey
	}
	return result, nil
}

var tmpCompanyRoles []models.Role

func getCompanyRoleInTmp(companyID, roleID string) (*models.Role, error) {
	isExisting := false
	for _, tmpRole := range tmpCompanyRoles {
		if tmpRole.RoleID == roleID {
			isExisting = true
			return &tmpRole, nil
		}
	}

	if !isExisting {
		role, err := ops.GetRoleByID(roleID, companyID)
		// role, err := ops.GetRoleByIDNew2(companyID, roleID)
		if err != nil {
			e := errors.New("Error getting role by id:" + roleID)
			return nil, e
		}
		tmpCompanyRoles = append(tmpCompanyRoles, role)
		return &role, nil
	}

	return nil, nil
}

// GetCompanyUserRolesInformation
func GetCompanyUserRolesInformation(companyID string, argRoles []models.Role) ([]models.Role, error) {
	var roles []models.Role
	for _, role := range argRoles {
		// r, err := getCompanyRoleInTmp(role.CompanyID, role.RoleID)
		r, err := ops.GetRoleByID(role.RoleID, role.CompanyID)
		if err != nil {
			e := errors.New(err.Message)
			return roles, e
		}
		roles = append(roles, r)
	}

	return roles, nil
}

// Optimized version of GetUserRoles()
func GetCompanyUserRoles(companyID, userID string) ([]models.Role, error) {
	roles := []models.Role{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
			},
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_ROLE),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return roles, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &roles)
	if err != nil {
		return roles, err
	}

	return roles, nil
}

/*
*****************************
Get User Groups
Returns all roles of the user
Parameters: userID
*****************************
*/
func GetUserRoles(userID, companyID string) ([]models.Role, error) {
	roles := []models.Role{}
	formattedRoles := []models.Role{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
			},
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_ROLE),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		e := errors.New("500")
		return roles, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &roles)
	if err != nil {
		e := errors.New("400")
		return roles, e
	}

	for _, role := range roles {
		// if role.CompanyID == companyID {
		// r, _ := ops.GetRoleByID(role.RoleID, companyID)
		// formattedRoles = append(formattedRoles, models.Role{
		// 	RoleID:   r.RoleID,
		// 	RoleName: r.RoleName,
		// })
		// }

		// role, err := ops.GetRoleByID(companyID, role.RoleID)
		// if err != nil {
		// 	// e := errors.New("Error getting role by id: " + err.Error())
		// 	// return nil, e
		// 	continue
		// } else {
		// 	formattedRoles = append(formattedRoles, models.Role{
		// 		RoleID:          role.RoleID,
		// 		RoleName:        role.RoleName,
		// 		RolePermissions: role.RolePermissions,
		// 	})
		// }
		role, err := getCompanyRoleInTmp(companyID, role.RoleID)
		if err != nil {
			continue
		}

		formattedRoles = append(formattedRoles, models.Role{
			RoleID:          role.RoleID,
			RoleName:        role.RoleName,
			RolePermissions: role.RolePermissions,
		})
	}

	return formattedRoles, nil
}

// Optimized version of GetUserCompanyGroups()
func GetUserCompanyGroupsNew(companyID, userID string) ([]models.Group, error) {
	groups := []models.Group{}
	userGroups := []models.Group{}

	param := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_BY_SK),
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
			},
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(param)
	if err != nil {
		return userGroups, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &userGroups)
	if err != nil {
		return userGroups, err
	}

	for _, userGroup := range userGroups {

		group, err := GetGroupInformation(userGroup.GroupID)
		if err == nil {
			if group.CompanyID == companyID {
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

// Optimized version of GetGroupDepartmentByID()
// func GetCompanyGroupDepartment(companyID, groupID string) (models.Department, error) {
// 	department := models.Department{}

// 	param := &dynamodb.QueryInput{
// 		TableName: aws.String(app.TABLE_NAME),
// 		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_BY_SK),
// 		KeyConditions: map[string]*dynamodb.Condition{
// 			"PK": {
// 				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
// 				AttributeValueList: []*dynamodb.AttributeValue{
// 					{
// 						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
// 					},
// 				},
// 			},
// 			"CompanyID": {
// 				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
// 				AttributeValueList: []*dynamodb.AttributeValue{
// 					{
// 						S: aws.String(companyID),
// 					},
// 				},
// 			},
// 		},
// 		QueryFilter: map[string]*dynamodb.Condition{
// 			"SK": {
// 				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
// 				AttributeValueList: []*dynamodb.AttributeValue{
// 					{
// 						S: aws.String(constants.PREFIX_DEPARTMENT),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	result, err := ops.HandleQueryLimit(param)
// 	if err != nil {
// 		return department, err
// 	}

// 	if len(result.Items) == 0 {
// 		return department, nil
// 	}

// 	err = dynamodbattribute.UnmarshalMap(result.Items[0], &department)
// 	if err != nil {
// 		return department, err
// 	}

// 	return department, nil
// }

/*
*****************************
Get User Groups
Used for fetching the groups a certain user belongs in
Parameters:
userID
*****************************
*/
func GetUserCompanyGroups(companyID, userID string) ([]models.Group, error) {

	userGroups := []models.Group{} // user + groups entity
	groups := []models.Group{}

	//Fetch User Groups
	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String("InvertedIndex"),
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
			},
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
				},
			},
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(queryInput)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return userGroups, e
	}

	//
	//

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &userGroups)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return userGroups, e
	}

	for _, userGroup := range userGroups {
		// groupData, err := ops.GetGroupData(group.GroupID, "")
		// if err != nil {
		// 	e := errors.New(err.Error())
		// 	return groups, e
		// }
		// groups2 = append(groups2, groupData)

		group, err := GetGroupInformation(userGroup.GroupID, "department")
		if err == nil {
			if group.CompanyID == companyID {
				groups = append(groups, group)
			}
		}
		// groups2[i] = models.Group{
		// 	PK: groupData.PK,
		// 	SK: groupData.SK,
		// 	DepartmentID: groupData.DepartmentID,
		// 	GroupID: groupData.GroupID,
		// 	GroupName: groupData.GroupName,
		// 	GroupColor: groupData.GroupColor,
		// 	Status:  groupData.Status,
		// }
	}

	return groups, nil
}

// @Summary Clone User Groups
// @Description This endpoint clones the existing groups to a certain user, returning a 200 OK upon success. If user is not found, the server will respond with a 404 Not Found.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.CloneUserGroupsRequest true "Clone user groups body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/groups/clone/:userID [post]
func (c UserController) CloneUserGroups(userID string) revel.Result {
	result := make(map[string]interface{})

	groupIDs := c.Params.Form.Get("groups")
	companyID := c.ViewArgs["companyID"].(string)

	// check if user id exists
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	var unmarshalGroups []string
	json.Unmarshal([]byte(groupIDs), &unmarshalGroups)

	// return c.RenderJSON(unmarshalGroups)

	var groups []models.Group

	// check if groups are exists
	for _, id := range unmarshalGroups {
		group, err := ops.GetGroupData(id, "")
		if err != nil || group.GroupID == "" {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_474)
			return c.RenderJSON(result)
		}
		groups = append(groups, models.Group{
			GroupID:      group.GroupID,
			DepartmentID: group.DepartmentID,
		})
	}

	// insert batch
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, group := range groups {
		// push to batch
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
				"GroupID": {
					S: aws.String(group.GroupID),
				},
				"CompanyID": {
					S: aws.String(companyID),
				},
				// "GroupRole": {
				// 	S: aws.String(constants.USER_ROLE_USER), // todo change
				// },
				"MemberID": {
					S: aws.String(user.UserID),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
				},
				"MemberType": {
					S: aws.String(constants.MEMBER_TYPE_USER),
				},
				"Status": {
					S: aws.String(constants.ITEM_STATUS_ACTIVE),
				},
				"CreatedAt": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
		}})
		if i%constants.BATCH_LIMIT == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}

	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		batches = append(batches, currentBatch)
	}

	// return c.RenderJSON(batches)

	_, err := ops.BatchWriteItemHandler(batches)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// insert department members
	var dmBatches [][]*dynamodb.WriteRequest
	var dmCurrentBatch []*dynamodb.WriteRequest

	for i, group := range groups {
		// push to batch
		dmCurrentBatch = append(dmCurrentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, group.DepartmentID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
				"DepartmentID": {
					S: aws.String(group.DepartmentID),
				},
				"UserID": {
					S: aws.String(user.UserID),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_DEPARTMENT_MEMBER),
				},
			},
		}})
		if i%constants.BATCH_LIMIT == 0 {
			dmBatches = append(dmBatches, dmCurrentBatch)
			dmCurrentBatch = nil
		}

	}

	if len(dmCurrentBatch) > 0 && len(dmCurrentBatch) != constants.BATCH_LIMIT {
		dmBatches = append(dmBatches, dmCurrentBatch)
	}

	_, err = ops.BatchWriteItemHandler(dmBatches)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith and John Doe has been added to GroupX
	var groupInfoUsers []models.LogModuleParams
	for _, group := range groups {
		groupInfoUsers = append(groupInfoUsers, models.LogModuleParams{
			ID: group.GroupID,
		})
	}
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_CLONE_USER_GROUPS,
		LogType:   constants.ENTITY_TYPE_USER,
		LogInfo: &models.LogInformation{
			Groups: groupInfoUsers,
			User: &models.LogModuleParams{
				ID: userID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		// result["message"] = "error while creating logs"
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
CreateUserLog()
- Create log for a user using the CreateLog global function
Params:
details - logs details
****************
*/
func (c UserController) CreateUserLog(details *models.LogModuleParams, companyID string, logAction string) (string, error) {
	authorID := c.ViewArgs["userID"].(string)
	logCtrl := LogController{}
	logInfo := &models.LogInformation{
		User:        details,
		PerformedBy: authorID,
	}
	logID, err := logCtrl.CreateLog(authorID, companyID, logAction, constants.ENTITY_TYPE_USER, logInfo)
	if err != nil {
		e := errors.New(err.Error())
		return "", e
	}
	return logID, nil
}

/*****************
SaveUserGoogleTokens()
- save google access token that will be use for google api's
*****************/
// func (c UserController) SaveUserGoogleTokens() revel.Result {
// 	result := make(map[string]interface{})

// 	accessToken := c.Params.Form.Get("access_token")
// 	refreshToken := c.Params.Form.Get("refresh_token")
// 	tokenExpiration := c.Params.Form.Get("token_expiration")
// 	emailDomain := c.Params.Form.Get("email_domain")
// 	userID := c.ViewArgs["userID"].(string)

// 	// check if user exists
// 	user, err := ops.GetUserByID(userID)
// 	if err != nil {
// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
// 		return c.RenderJSON(result)
// 	}

// 	if refreshToken == "" {
// 		refreshToken = user.GoogleRefreshToken
// 	}

// 	if emailDomain == "" {
// 		emailDomain = user.GoogleEmailDomain
// 	}

// 	// update user details, include google access token
// 	input := &dynamodb.UpdateItemInput{
// 		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
// 			":gst": {
// 				S: aws.String(accessToken),
// 			},
// 			":grt": {
// 				S: aws.String(refreshToken),
// 			},
// 			":gte": {
// 				S: aws.String(tokenExpiration),
// 			},
// 			":ged": {
// 				S: aws.String(emailDomain),
// 			},
// 		},
// 		TableName: aws.String(app.TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			"PK": {
// 				S: aws.String(user.PK),
// 			},
// 			"SK": {
// 				S: aws.String(user.SK),
// 			},
// 		},
// 		UpdateExpression: aws.String("SET GoogleAccessToken = :gst, GoogleRefreshToken = :grt, GoogleTokenExpiration = :gte, GoogleEmailDomain = :ged"),
// 	}

// 	_, err = app.SVC.UpdateItem(input)
// 	if err != nil {
// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
// 		return c.RenderJSON(result)
// 	}

// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
// 	return c.RenderJSON(result)
// }

/*
****************
SaveUserMicrosoftTokens()
- save google access token that will be use for google api's
****************
*/
func (c UserController) SaveUserJiraTokens() revel.Result {
	appSecretKey, _ := revel.Config.String("app.encryption.key")
	result := make(map[string]interface{})
	email := c.Params.Form.Get("email")
	key := c.Params.Form.Get("key")
	domain := c.Params.Form.Get("domain")

	combinedData := email + ":" + key
	base64Data := utils.ConvertToB64(combinedData)
	encodedData := utils.EncryptString(base64Data, appSecretKey)
	modifiedDomain := "https://" + domain + ".atlassian.net"

	userID := c.ViewArgs["userID"].(string)

	// check if user exists
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	resp, err := jiraoperations.GetPermission(base64Data, modifiedDomain)
	if err != nil {
		result["message"] = "jira ops"
		result["error"] = err.Error()
		c.Response.Status = 403
		return c.RenderJSON(result)
	}

	if resp.StatusCode != 200 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		c.Response.Status = 401
		return c.RenderJSON(result)
	}

	// update user details, include jira access token
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tk": {
				S: aws.String(encodedData),
			},
			":dm": {
				S: aws.String(modifiedDomain),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(user.PK),
			},
			"SK": {
				S: aws.String(user.SK),
			},
		},
		UpdateExpression: aws.String("SET JiraToken = :tk, JiraDomain = :dm"),
	}

	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		c.Response.Status = 500
		return c.RenderJSON(result)
	}

	result["message"] = "Saved!"
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
RemoveUser - remove user by id
Params - userID
*/
func (c UserController) RemoveUser(userID string) revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)
	authorID := c.ViewArgs["userID"].(string)

	if authorID == userID {
		c.Response.Status = 422
		c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "You are not able to remove yourself.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	removeIntegrationAccountsParams := c.Params.Query.Get("remove_integration_accounts")
	removeIntegrationAccounts, err := strconv.ParseBool(removeIntegrationAccountsParams)
	if err != nil {
		removeIntegrationAccounts = false
	}

	// check user
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	// check user groups
	groups, err := GetUserCompanyGroups(companyID, user.UserID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	// get group integrations
	// include sub integrations
	for i, group := range groups {
		groupIntegrations, opsError := ops.GetGroupIntegrations(group.GroupID)
		if opsError != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
		}
		groups[i].GroupIntegrations = *groupIntegrations

		for j, groupIntegration := range groups[i].GroupIntegrations {
			subIntegrations, err := ops.GetGroupSubIntegration(group.GroupID)
			if err != nil {
				result["message"] = "Something went wrong with group sub integrations"
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}

			subInteg := []models.GroupSubIntegration{}
			for _, sub := range subIntegrations {
				if sub.ParentIntegrationID == groupIntegration.IntegrationID {
					subInteg = append(subInteg, sub)
				}
			}
			groups[i].GroupIntegrations[j].GroupSubIntegrations = subInteg
		}
	}

	// // make user inactive
	// updateInput := &dynamodb.UpdateItemInput{
	// 	ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
	// 		":s": {
	// 			S: aws.String(constants.ITEM_STATUS_INACTIVE),
	// 		},
	// 		":ua": {
	// 			S: aws.String(utils.GetCurrentTimestamp()),
	// 		},
	// 	},
	// 	ExpressionAttributeNames: map[string]*string{
	// 		"#s": aws.String("Status"),
	// 	},
	// 	TableName: aws.String(app.TABLE_NAME),
	// 	Key: map[string]*dynamodb.AttributeValue{
	// 		"PK": {
	// 			S: aws.String(user.PK),
	// 		},
	// 		"SK": {
	// 			S: aws.String(user.SK),
	// 		},
	// 	},
	// 	UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
	// }

	// _, err = app.SVC.UpdateItem(updateInput)
	// if err != nil {
	// 	result["errors"] = []string{err.Error()}
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(result)
	// }

	// // delete user and company relationship
	// userCompany := &dynamodb.DeleteItemInput{
	// 	Key: map[string]*dynamodb.AttributeValue{
	// 		"PK": {
	// 			S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
	// 		},
	// 		"SK": {
	// 			S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
	// 		},
	// 	},
	// 	TableName: aws.String(app.TABLE_NAME),
	// }

	// _, err = app.SVC.DeleteItem(userCompany)
	// if err != nil {
	// 	result["errors"] = []string{err.Error()}
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(result)
	// }

	// make company user relationship inactive
	err = RemoveUserInCompany(userID, companyID)
	if err != nil {
		result["errors"] = []string{err.Error()}
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// loop through groups
	// delete user and groups relationship
	for _, group := range groups {
		usersGroup := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err = app.SVC.DeleteItem(usersGroup)
		if err != nil {
			result["message"] = "Error from DB"
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}
	//generate log
	// var logInfoGroups []models.LogModuleParams
	// for _, group := range groups {
	// 	logInfoGroups = append(logInfoGroups, models.LogModuleParams{
	// 		ID: group.GroupID,
	// 	})
	// }

	// var logs []*models.Logs
	// logs = append(logs, &models.Logs{
	// 	CompanyID: companyID,
	// 	UserID:    userID,
	// 	LogAction: constants.LOG_ACTION_REMOVE_GROUP_USER,
	// 	LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
	// 	LogInfo: &models.LogInformation{
	// 		Company: &models.LogModuleParams{
	// 			ID: companyID,
	// 		},
	// 		User: &models.LogModuleParams{
	// 			Old: map[string]interface{}{
	// 				"Name": user.FirstName + " " + user.LastName,
	// 			},
	// 		},
	// 	},
	// })

	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["message"] = "error while creating logs"
	// }

	var errorMessages []string

	log := &models.Logs{
		CompanyID: companyID,
		UserID:    authorID,
		LogAction: constants.LOG_ACTION_REMOVE_COMPANY_MEMBERS,
		LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			Users: []models.LogModuleParams{
				models.LogModuleParams{
					Temp: models.TempParam{
						Name: user.FirstName + " " + user.LastName,
						ID:   user.UserID,
					},
				},
			},
		},
	}

	logID, err := ops.InsertLog(log)
	if err != nil {
		errorMessages = append(errorMessages, "error while creating logs")
	}

	// create action item
	actionItem := models.ActionItem{
		CompanyID:      companyID,
		LogID:          logID,
		ActionItemType: "REMOVE_USER_TO_COMPANY",
		SearchKey:      user.FirstName + " " + user.LastName + " " + user.Email,
	}
	actionItemID, err := ops.CreateActionItem(actionItem)
	if err != nil {
		errorMessages = append(errorMessages, "error while creating action items")
	}
	_ = actionItemID

	if !removeIntegrationAccounts {
		result["groups"] = groups
		result["user"] = user
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
		return c.RenderJSON(result)
	}

	//get company integrations
	var integrations []models.Integration

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
					},
				},
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_INTEGRATION),
					},
				},
			},
		},
	}

	res, err := app.SVC.Query(params)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &integrations)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	var integrationErrors []map[string]interface{}

	for j, integration := range integrations {
		i, err := GetIntegrationByID(integration.IntegrationID)
		if err == nil {
			integrations[j].IntegrationName = i.IntegrationName
			integrations[j].IntegrationSlug = i.IntegrationSlug
			token, errr := ops.GetCompanyIntegration(companyID, integration.IntegrationID)
			if errr != nil {
				// integErrors =
			} else {
				switch i.IntegrationSlug {
				case constants.INTEG_SLUG_GITHUB:
					integrationToken := token.IntegrationToken.AccessToken
					userUid, err := ops.GetUserIntegrationUID(integration.IntegrationID, userID)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": err.Error()})
					} else {
						if userUid.IntegrationUID != "" {
							org, gitError := githuboperations.GetConnectedOrganization(companyID, integration.IntegrationID)
							if gitError != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": gitError})
							}
							if org != nil || *org == "" {
								integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": "Organization not found"})
							}
							gitError = githuboperations.RemoveMemberToOrganization(*org, userUid.IntegrationUID, "Bearer "+integrationToken)
							if gitError != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": gitError})
							}
						}
					}
				case constants.INTEG_SLUG_OFFICE_365:
					newToken, officeError := officeOperations.RefreshToken(token.IntegrationToken)
					if officeError != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
					} else {
						currentAzureUser, officeError := officeOperations.GetCurretUserInfo(newToken.AccessToken)
						if officeError != nil {
							integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
						}

						domain := utils.GetMsDomain(currentAzureUser.UserPrincipalName)

						azureUser, officeError := officeOperations.SortAzureUser(user.Email, domain, newToken.AccessToken)
						if officeError != nil {
							integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
						}

						if azureUser != nil {
							officeError = officeOperations.DeleteAzureUser(azureUser.ID, newToken.AccessToken)
							if officeError != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
							}
						}
					}
				case constants.INTEG_SLUG_GOOGLE_CLOUD:
					primaryEmails, err := googleoperations.GetPrimaryEmailsByUserEmail(user.Email, token.IntegrationToken, token.IntegrationTokenExtra)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"google-error": err})
					}
					if len(primaryEmails) != 0 {
						for _, email := range primaryEmails {
							googleError := googleoperations.DeleteGoogleUser(token.IntegrationToken, token.IntegrationTokenExtra, email)
							if googleError != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"google-error": googleError})
							}
						}
					}
				case constants.INTEG_SLUG_SALESFORCE:
					//TODO: @JAM
					salesforceToken, err := salesforceoperations.ValidateToken(token.IntegrationSalesforceToken)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": err})
						break
					}
					bearerToken := salesforceToken.TokenType + " " + salesforceToken.AccessToken
					response, userErr := salesforceoperations.GetSalesforceUserDetails(bearerToken, salesforceToken.InstanceUrl, user.Email)
					if userErr != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": userErr})
						break
					}
					if response.StatusCode != 200 {
						integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": userErr})
						break
					}

					body, readErr := ioutil.ReadAll(response.Body)
					if readErr != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": readErr})
						break
					}
					defer response.Body.Close()
					var salesforceUser salesforceModel.SalesforceQueryUsersResponse

					readErr = json.Unmarshal([]byte(body), &salesforceUser)
					if readErr != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": readErr})
						break
					}
					if len(salesforceUser.Records) > 0 && salesforceUser.Records[0].IsActive {
						postBody, _ := json.Marshal(map[string]interface{}{
							"isActive": false,
						})
						_, deactivateErr := salesforceoperations.SalesforceUserUpdate(bearerToken, salesforceToken.InstanceUrl, salesforceUser.Records[0].ID, postBody)
						if deactivateErr != nil {
							integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": deactivateErr})
							break
						}
					}
				case constants.INTEG_SLUG_AWS:
					userUid, err := ops.GetUserIntegrationUID(integration.IntegrationID, userID)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"aws-error": err.Error()})
					} else {
						if userUid.IntegrationUID != "" {
							awsCredentials := awsoperations.GetAWSCredentials(token)
							awsErrors := awsoperations.DeleteIAMUser(userUid.IntegrationUID, awsCredentials)
							if awsErrors != nil {
								var aErrs []string
								for _, aErr := range awsErrors {
									aErrs = append(aErrs, aErr.Error())
								}
								integrationErrors = append(integrationErrors, map[string]interface{}{"aws-error": aErrs})
							}
						}
					}
				case constants.INTEG_SLUG_JIRA:
					appSecretKey, _ := revel.Config.String("app.encryption.key")
					// authorDetails, _ := ops.GetUserByID(authorID)
					jiraToken := token.JiraToken.JiraToken
					jiraDomain := token.JiraToken.JiraDomain

					// decodedData := utils.DecryptString(authorDetails.JiraToken, appSecretKey)
					decodedData := utils.DecryptString(*jiraToken, appSecretKey)

					// resp, err := jiraoperations.GetUser(decodedData, authorDetails.JiraDomain, user.Email)
					resp, err := jiraoperations.GetUser(decodedData, *jiraDomain, user.Email)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
						break
					}
					// integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": resp.Body})
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
						break
					}
					defer resp.Body.Close()

					var jiraUser []jiramodel.JiraUser
					err = json.Unmarshal([]byte(body), &jiraUser)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
						break
					}

					// resp, _ = jiraoperations.DeleteUser(decodedData, authorDetails.JiraDomain, jiraUser[0].AccountID)
					resp, _ = jiraoperations.DeleteUser(decodedData, *jiraDomain, jiraUser[0].AccountID)
					if resp.StatusCode != 204 {
						integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": resp.Status})
					}

				case constants.INTEG_SLUG_DROPBOX:
					//TODO: @Grace

					//get dropboxteamid from admin
					// admin, err := dropboxoperations.GetAuthenticatedAdmin("Bearer " + token.IntegrationToken.AccessToken)
					// if err != nil || token.IntegrationToken == nil {
					// 	c.Response.Status = 401
					// 	return c.RenderJSON("Dropbox admin not found")
					// }
					//dropboxTeamToken := admin.AdminProfile.TeamMemberID

					//1. get dropbox token
					dropboxToken := "Bearer " + token.IntegrationToken.AccessToken

					//2. Get MemberInformation for TeamMemberID
					mem := []map[string]interface{}{
						{
							".tag":  "email",
							"email": user.Email,
						},
					}

					memberInfo, getMemberError := dropboxoperations.GetMemberInfo(dropboxToken, mem)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"dropbox-error": getMemberError})
						break
					}

					//TeamMemberId to be removed
					var TeamMemberIdToRemove *string
					for _, info := range memberInfo.MembersInfo {
						if *info.Tag == "member_info" {
							TeamMemberIdToRemove = info.Profile.TeamMemberID
						}
					}

					//3. Removing User's dropbox account
					_, removeError := dropboxoperations.RemoveMemberAccount(dropboxToken, TeamMemberIdToRemove)
					if removeError != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"dropbox-error": removeError})
						break
					}
				case constants.INTEG_SLUG_ZENDESK:
					zendeskToken := token.ZendeskToken.ZendeskOAuthToken.AccessToken
					zendeskSubdomain := token.ZendeskToken.ZendeskSubdomain

					resp, err := zendeskoperations.GetUserByEmail(zendeskToken, user.Email, zendeskSubdomain)
					if err != nil {
						integrationErrors = append(integrationErrors, map[string]interface{}{"zendesk-error": err})
						break
					} else {
						//get user id
						var idToDelete int
						for _, user := range resp.Users {
							idToDelete = user.Id
						}

						//check if exists(true: delete user on zendesk)
						if idToDelete != 0 {
							deleteUserError := zendeskoperations.DeleteUser(zendeskToken, idToDelete, zendeskSubdomain) //resp.Users[0].Id (recent userId that is passed)
							if deleteUserError != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"zendesk-error": deleteUserError})
							}
						} else {
							break
						}
					}
				default:
					result[i.IntegrationSlug] = i.IntegrationSlug + " not found in case"
				}
			}
		}
	}

	// return user and user groups
	if len(integrationErrors) > 0 {
		result["integration_errors"] = integrationErrors
	}

	result["groups"] = groups
	result["user"] = user
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

func (c UserController) RemoveMultipleUsers() revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)
	authorID := c.ViewArgs["userID"].(string)

	removeIntegrationAccountsParams := c.Params.Query.Get("remove_integration_accounts")
	removeIntegrationAccounts, err := strconv.ParseBool(removeIntegrationAccountsParams)
	if err != nil {
		removeIntegrationAccounts = false
	}

	userIDs := []string{}
	c.Params.Bind(&userIDs, "users")

	for _, userID := range userIDs {

		user, opsError := ops.GetUserByID(userID)
		if opsError != nil {
			c.Response.Status = opsError.HTTPStatusCode
			return c.RenderJSON(opsError)
		}

		companyUser, err := GetCompanyUser(companyID, userID)
		if err != nil {
			continue
		}

		if companyUser.UserType == "COMPANY_OWNER" {
			continue
		}

		// check user groups
		groups, err := GetUserCompanyGroups(companyID, user.UserID)
		if err != nil {
			c.Response.Status = 400
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		// get group integrations
		// include sub integrations
		for i, group := range groups {
			groupIntegrations, opsError := ops.GetGroupIntegrations(group.GroupID)
			if opsError != nil {
				c.Response.Status = 400
				result["status"] = utils.GetHTTPStatus(err.Error())
			}
			groups[i].GroupIntegrations = *groupIntegrations

			for j, groupIntegration := range groups[i].GroupIntegrations {
				subIntegrations, err := ops.GetGroupSubIntegration(group.GroupID)
				if err != nil {
					c.Response.Status = 400
					result["message"] = "Something went wrong with group sub integrations"
					result["status"] = utils.GetHTTPStatus(err.Error())
					return c.RenderJSON(result)
				}

				subInteg := []models.GroupSubIntegration{}
				for _, sub := range subIntegrations {
					if sub.ParentIntegrationID == groupIntegration.IntegrationID {
						subInteg = append(subInteg, sub)
					}
				}
				groups[i].GroupIntegrations[j].GroupSubIntegrations = subInteg
			}
		}

		// make company user relationship inactive
		err = RemoveUserInCompany(userID, companyID)
		if err != nil {
			c.Response.Status = 400
			result["errors"] = []string{err.Error()}
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		// loop through groups
		// delete user and groups relationship
		for _, group := range groups {
			usersGroup := &dynamodb.DeleteItemInput{
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
					},
				},
				TableName: aws.String(app.TABLE_NAME),
			}

			_, err = app.SVC.DeleteItem(usersGroup)
			if err != nil {
				c.Response.Status = 400
				result["message"] = "Error from DB"
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}
		}

		var errorMessages []string

		log := &models.Logs{
			CompanyID: companyID,
			UserID:    authorID,
			LogAction: constants.LOG_ACTION_REMOVE_COMPANY_MEMBERS,
			LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
			LogInfo: &models.LogInformation{
				Company: &models.LogModuleParams{
					ID: companyID,
				},
				Users: []models.LogModuleParams{
					models.LogModuleParams{
						Temp: models.TempParam{
							Name: user.FirstName + " " + user.LastName,
							ID:   user.UserID,
						},
					},
				},
			},
		}

		logID, err := ops.InsertLog(log)
		if err != nil {
			c.Response.Status = 400
			errorMessages = append(errorMessages, "error while creating logs")
		}

		// create action item
		actionItem := models.ActionItem{
			CompanyID:      companyID,
			LogID:          logID,
			ActionItemType: "REMOVE_USER_TO_COMPANY",
			SearchKey:      user.FirstName + " " + user.LastName + " " + user.Email,
		}
		actionItemID, err := ops.CreateActionItem(actionItem)
		if err != nil {
			c.Response.Status = 400
			errorMessages = append(errorMessages, "error while creating action items")
		}
		_ = actionItemID

		if removeIntegrationAccounts {
			//get company integrations
			var integrations []models.Integration

			params := &dynamodb.QueryInput{
				TableName: aws.String(app.TABLE_NAME),
				KeyConditions: map[string]*dynamodb.Condition{
					"PK": {
						ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
							},
						},
					},
					"SK": {
						ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String(constants.PREFIX_INTEGRATION),
							},
						},
					},
				},
			}

			res, err := app.SVC.Query(params)
			if err != nil {
				c.Response.Status = 400
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}

			err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &integrations)
			if err != nil {
				c.Response.Status = 400
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
				return c.RenderJSON(result)
			}

			var integrationErrors []map[string]interface{}

			for j, integration := range integrations {
				i, err := GetIntegrationByID(integration.IntegrationID)
				if err == nil {
					c.Response.Status = 400
					integrations[j].IntegrationName = i.IntegrationName
					integrations[j].IntegrationSlug = i.IntegrationSlug
					token, errr := ops.GetCompanyIntegration(companyID, integration.IntegrationID)
					if errr != nil {
						// integErrors =
					} else {
						switch i.IntegrationSlug {
						case constants.INTEG_SLUG_GITHUB:
							integrationToken := token.IntegrationToken.AccessToken
							userUid, err := ops.GetUserIntegrationUID(integration.IntegrationID, userID)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": err.Error()})
							} else {
								if userUid.IntegrationUID != "" {
									org, gitError := githuboperations.GetConnectedOrganization(companyID, integration.IntegrationID)
									if gitError != nil {
										integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": gitError})
									}
									if org != nil || *org == "" {
										integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": "Organization not found"})
									}
									gitError = githuboperations.RemoveMemberToOrganization(*org, userUid.IntegrationUID, "Bearer "+integrationToken)
									if gitError != nil {
										integrationErrors = append(integrationErrors, map[string]interface{}{"github-error": gitError})
									}
								}
							}
						case constants.INTEG_SLUG_OFFICE_365:
							newToken, officeError := officeOperations.RefreshToken(token.IntegrationToken)
							if officeError != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
							} else {
								currentAzureUser, officeError := officeOperations.GetCurretUserInfo(newToken.AccessToken)
								if officeError != nil {
									integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
								}

								domain := utils.GetMsDomain(currentAzureUser.UserPrincipalName)

								azureUser, officeError := officeOperations.SortAzureUser(user.Email, domain, newToken.AccessToken)
								if officeError != nil {
									integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
								}

								if azureUser != nil {
									officeError = officeOperations.DeleteAzureUser(azureUser.ID, newToken.AccessToken)
									if officeError != nil {
										integrationErrors = append(integrationErrors, map[string]interface{}{"office-error": officeError.Status})
									}
								}
							}
						case constants.INTEG_SLUG_GOOGLE_CLOUD:
							primaryEmails, err := googleoperations.GetPrimaryEmailsByUserEmail(user.Email, token.IntegrationToken, token.IntegrationTokenExtra)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"google-error": err})
							}
							if len(primaryEmails) != 0 {
								for _, email := range primaryEmails {
									googleError := googleoperations.DeleteGoogleUser(token.IntegrationToken, token.IntegrationTokenExtra, email)
									if googleError != nil {
										integrationErrors = append(integrationErrors, map[string]interface{}{"google-error": googleError})
									}
								}
							}
						case constants.INTEG_SLUG_SALESFORCE:
							//TODO: @JAM
							salesforceToken, err := salesforceoperations.ValidateToken(token.IntegrationSalesforceToken)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": err})
								break
							}
							bearerToken := salesforceToken.TokenType + " " + salesforceToken.AccessToken
							response, userErr := salesforceoperations.GetSalesforceUserDetails(bearerToken, salesforceToken.InstanceUrl, user.Email)
							if userErr != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": userErr})
								break
							}
							if response.StatusCode != 200 {
								integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": userErr})
								break
							}

							body, readErr := ioutil.ReadAll(response.Body)
							if readErr != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": readErr})
								break
							}
							defer response.Body.Close()
							var salesforceUser salesforceModel.SalesforceQueryUsersResponse

							readErr = json.Unmarshal([]byte(body), &salesforceUser)
							if readErr != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": readErr})
								break
							}
							if len(salesforceUser.Records) > 0 && salesforceUser.Records[0].IsActive {
								postBody, _ := json.Marshal(map[string]interface{}{
									"isActive": false,
								})
								_, deactivateErr := salesforceoperations.SalesforceUserUpdate(bearerToken, salesforceToken.InstanceUrl, salesforceUser.Records[0].ID, postBody)
								if deactivateErr != nil {
									integrationErrors = append(integrationErrors, map[string]interface{}{"salesforce-error": deactivateErr})
									break
								}
							}
						case constants.INTEG_SLUG_AWS:
							userUid, err := ops.GetUserIntegrationUID(integration.IntegrationID, userID)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"aws-error": err.Error()})
							} else {
								if userUid.IntegrationUID != "" {
									awsCredentials := awsoperations.GetAWSCredentials(token)
									awsErrors := awsoperations.DeleteIAMUser(userUid.IntegrationUID, awsCredentials)
									if awsErrors != nil {
										var aErrs []string
										for _, aErr := range awsErrors {
											aErrs = append(aErrs, aErr.Error())
										}
										integrationErrors = append(integrationErrors, map[string]interface{}{"aws-error": aErrs})
									}
								}
							}
						case constants.INTEG_SLUG_JIRA:
							appSecretKey, _ := revel.Config.String("app.encryption.key")
							// authorDetails, _ := ops.GetUserByID(authorID)
							jiraToken := token.JiraToken.JiraToken
							jiraDomain := token.JiraToken.JiraDomain

							// decodedData := utils.DecryptString(authorDetails.JiraToken, appSecretKey)
							decodedData := utils.DecryptString(*jiraToken, appSecretKey)

							// resp, err := jiraoperations.GetUser(decodedData, authorDetails.JiraDomain, user.Email)
							resp, err := jiraoperations.GetUser(decodedData, *jiraDomain, user.Email)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
								break
							}
							// integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": resp.Body})
							body, err := ioutil.ReadAll(resp.Body)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
								break
							}
							defer resp.Body.Close()

							var jiraUser []jiramodel.JiraUser
							err = json.Unmarshal([]byte(body), &jiraUser)
							if err != nil {
								integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": err.Error()})
								break
							}

							// resp, _ = jiraoperations.DeleteUser(decodedData, authorDetails.JiraDomain, jiraUser[0].AccountID)
							resp, _ = jiraoperations.DeleteUser(decodedData, *jiraDomain, jiraUser[0].AccountID)
							if resp.StatusCode != 204 {
								integrationErrors = append(integrationErrors, map[string]interface{}{"jira-error": resp.Status})
							}

						case constants.INTEG_SLUG_DROPBOX:
							//TODO: @Grace

							//get dropboxteamid from admin
							// admin, err := dropboxoperations.GetAuthenticatedAdmin("Bearer " + token.IntegrationToken.AccessToken)
							// if err != nil || token.IntegrationToken == nil {
							// 	c.Response.Status = 401
							// 	return c.RenderJSON("Dropbox admin not found")
							// }
							//dropboxTeamToken := admin.AdminProfile.TeamMemberID

							//1. get dropbox token
							dropboxToken := "Bearer " + token.IntegrationToken.AccessToken

							//2. Get MemberInformation for TeamMemberID
							mem := []map[string]interface{}{
								{
									".tag":  "email",
									"email": user.Email,
								},
							}

							memberInfo, getMemberError := dropboxoperations.GetMemberInfo(dropboxToken, mem)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"dropbox-error": getMemberError})
								break
							}

							//TeamMemberId to be removed
							var TeamMemberIdToRemove *string
							for _, info := range memberInfo.MembersInfo {
								if *info.Tag == "member_info" {
									TeamMemberIdToRemove = info.Profile.TeamMemberID
								}
							}

							//3. Removing User's dropbox account
							_, removeError := dropboxoperations.RemoveMemberAccount(dropboxToken, TeamMemberIdToRemove)
							if removeError != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"dropbox-error": removeError})
								break
							}
						case constants.INTEG_SLUG_ZENDESK:
							zendeskToken := token.ZendeskToken.ZendeskOAuthToken.AccessToken
							zendeskSubdomain := token.ZendeskToken.ZendeskSubdomain

							resp, err := zendeskoperations.GetUserByEmail(zendeskToken, user.Email, zendeskSubdomain)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"zendesk-error": err})
								break
							} else {
								//get user id
								var idToDelete int
								for _, user := range resp.Users {
									idToDelete = user.Id
								}

								//check if exists(true: delete user on zendesk)
								if idToDelete != 0 {
									deleteUserError := zendeskoperations.DeleteUser(zendeskToken, idToDelete, zendeskSubdomain) //resp.Users[0].Id (recent userId that is passed)
									if deleteUserError != nil {
										integrationErrors = append(integrationErrors, map[string]interface{}{"zendesk-error": deleteUserError})
									}
								} else {
									break
								}
							}
						case constants.INTEG_SLUG_DOCUSIGN:
							docuSignToken := token.IntegrationToken.AccessToken
							baseUrl, ok := c.ViewArgs["docusignBaseURL"].(string)
							if !ok {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"docusign-error": ok})
								break
							}

							resp, err := docusignoperations.GetDocuSignUserByEmail(docuSignToken, baseUrl, user.Email)
							if err != nil {
								c.Response.Status = 400
								integrationErrors = append(integrationErrors, map[string]interface{}{"docusign-error": err})
								break
							} else {
								var idToDelete string
								if len(resp.Users) != 0 {
									for _, user := range resp.Users {
										idToDelete = user.UserID
									}

									postBody, _ := json.Marshal(map[string]interface{}{
										"users": []map[string]interface{}{
											{
												"userId": idToDelete,
											},
										},
									})

									if idToDelete != "" {
										_, deleteUserError := docusignoperations.DeleteUser(baseUrl, docuSignToken, postBody)
										if deleteUserError != nil {
											integrationErrors = append(integrationErrors, map[string]interface{}{"docusign-error": deleteUserError})
										}
										//
									} else {
										break
									}
								} else {
									break
								}
							}
						default:
							result[i.IntegrationSlug] = i.IntegrationSlug + " not found in case"
						}
					}
				}
			}
			if len(integrationErrors) > 0 {
				result["integration_errors"] = integrationErrors
			}
		}

		// if !removeIntegrationAccounts {
		// 	result["groups"] = groups
		// 	result["user"] = user
		// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
		// 	return c.RenderJSON(result)
		// }

		// return user and user groups

	}

	c.Response.Status = 200
	return c.RenderJSON(userIDs)
}

/*
RemoveUserJiraTokens
*/
func (c UserController) RemoveUserJiraTokens() revel.Result {
	result := make(map[string]interface{})

	userID := c.ViewArgs["userID"].(string)

	// check if user exists
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	// update user details, include google access token
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tk": {
				S: aws.String(""),
			},
			":dm": {
				S: aws.String(""),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(user.PK),
			},
			"SK": {
				S: aws.String(user.SK),
			},
		},
		UpdateExpression: aws.String("SET JiraToken = :tk, JiraDomain = :dm"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["message"] = "removed!"
	return c.RenderJSON(result)
}

func (c UserController) IntegrationEmail(users []mail.Recipient) revel.Result {
	result := make(map[string]interface{})

	// fName := c.Params.Form.Get("f_name")
	// lName := c.Params.Form.Get("l_name")
	// altMail := c.Params.Form.Get("alt_mail")
	// mainMail := c.Params.Form.Get("main_mail")
	// passWord := c.Params.Form.Get("pass_word")
	// integrationName := c.Params.Form.Get("integration_name")

	subject := "You have a new account"
	if users[0].IntegrationName != "" {
		subject = "You have a new " + users[0].IntegrationName + " account"
	}
	var recipient []mail.Recipient
	for _, user := range users {
		var redirectURL = ""

		if user.IntegrationName == "Microsoft" {
			redirectURL = "https://login.microsoftonline.com/"
		} else {
			redirectURL = "https://accounts.google.com/signin/v2"
		}

		recipient = append(recipient, mail.Recipient{
			Name:            user.Name,
			Email:           user.Email,
			RedirectLink:    redirectURL,
			AccountEmail:    user.AccountEmail,
			AccountPassword: user.AccountPassword,
			IntegrationName: user.IntegrationName,
		})
	}

	jobs.Now(mail.SendEmail{
		Subject:    subject,
		Recipients: recipient,
		Template:   "new_user_integration.html",
	})

	//

	// if fName == "" || lName == "" || altMail == "" || mainMail == "" || passWord == "" {
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
	// } else {
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	// }

	return c.RenderJSON(result)
}

// SkipSetupWizard
func (c UserController) SkipSetupWizard(companyID string) revel.Result {
	result := make(map[string]interface{})

	// userID := c.ViewArgs["userID"].(string)
	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		c.Response.Status = 404
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":us": {
				S: aws.String("DONE"),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(company.PK),
			},
			"SK": {
				S: aws.String(company.SK),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET SetupWizardStatus = :us"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		c.Response.Status = 500
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	return c.RenderJSON(result)
}

// FinishTour
func (c UserController) FinishTour() revel.Result {

	userid := c.Params.Form.Get("user_id")
	result := make(map[string]interface{})

	if len(userid) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	user, opsError := ops.GetUserByID(userid)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":us": {
				S: aws.String(constants.BOOL_TRUE),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
			},
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userid)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET IsTourDone = :us"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		c.Response.Status = 500
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	userr, errs := ops.GetUserByID(userid)
	if errs != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["user"] = userr
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// Change User Validate Status
func (c UserController) ConfirmVerification() revel.Result {

	userid := c.Params.Form.Get("user_id")
	result := make(map[string]interface{})

	if len(userid) == 0 {
		c.Response.Status = 422
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	user, opsError := ops.GetUserByID(userid)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":us": {
				S: aws.String(constants.BOOL_TRUE),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
			},
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userid)),
			},
		},
		ReturnValues:     aws.String("ALL_NEW"),
		UpdateExpression: aws.String("SET IsVerified = :us, UpdatedAt = :ua"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		c.Response.Status = 500
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

type RestoreUsersInput struct {
	Users []string `json:"user_ids,omitempty"`
}

func (c UserController) RestoreUsers() revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)
	createdBy := c.ViewArgs["userID"].(string)

	input := RestoreUsersInput{}
	c.Params.BindJSON(&input)

	if len(input.Users) == 0 {
		// return 422
	}

	var validUsers []models.CompanyUser

	// loop in input.Users
	for _, userID := range input.Users {
		// check if user exists in company
		companyUser, err := GetCompanyUser(companyID, userID)
		if err == nil {
			user, err := ops.GetUserByID(companyUser.UserID)
			if err == nil {
				companyUser.UserToken = user.UserToken // temporary fix for determining if admin or not
				validUsers = append(validUsers, companyUser)
			}
		}
		// check if user exists on the app?
	}

	if len(validUsers) == 0 {
		// reutrn 422
		// todo error
		return nil
	}

	var logInfoUsers []models.LogModuleParams

	for _, user := range validUsers {
		// s := constants.ITEM_STATUS_PENDING
		// if user.UserToken == "DEFAULT_USER" {
		// 	s = constants.ITEM_STATUS_DEFAULT
		// }
		previousStatus := user.PreviousStatus
		if previousStatus == "" {
			previousStatus = constants.ITEM_STATUS_DEFAULT
		}

		// update company user to active
		updateInput := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":s": {
					S: aws.String(previousStatus),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			ExpressionAttributeNames: map[string]*string{
				"#s": aws.String("Status"),
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(user.PK),
				},
				"SK": {
					S: aws.String(user.SK),
				},
			},
			UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(updateInput)
		if err == nil {
			// todo handle error

			// generate log
			logInfoUsers = append(logInfoUsers, models.LogModuleParams{
				ID: user.UserID,
			})
		}

		// send invite?
	}

	// generate log
	var errorMessages []string
	log := &models.Logs{
		CompanyID: companyID,
		UserID:    createdBy,
		LogAction: constants.LOG_ACTION_RESTORE_COMPANY_MEMBERS,
		LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	}

	logID, err := ops.InsertLog(log)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating log for restoring users")
	}
	_ = logID

	// generate action item
	// actionItem := models.ActionItem{
	// 	CompanyID:      companyID,
	// 	LogID:          logID,
	// 	ActionItemType: "ADD_USERS_TO_GROUP",
	// }

	// actionItemID, err := ops.CreateActionItem(actionItem)
	// if err != nil {
	// 	errorMessages = append(errorMessages, "error while action items")
	// }
	// _ = actionItemID

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	return c.RenderJSON(nil)
}

type PermanentlyDeleteUsersInput struct {
	Users []string `json:"user_ids,omitempty"`
}

func (c UserController) PermanentlyDeleteUsers() revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)
	requestUserID := c.ViewArgs["userID"].(string)

	input := RestoreUsersInput{}
	c.Params.BindJSON(&input)

	if len(input.Users) == 0 {
		// return 422
	}

	// check permission
	hasPermission := ops.CheckPermissions(constants.REMOVE_COMPANY_MEMBER, requestUserID, companyID)
	if !hasPermission {
		c.Response.Status = 403
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_403)
		return c.RenderJSON(result)
	}

	var validUsers []models.CompanyUser

	for _, user := range input.Users {
		// check if user exists in company
		companyUser, err := GetCompanyUser(companyID, user)
		if err == nil {
			validUsers = append(validUsers, companyUser)
		}

	}

	var logInfoTmpUsers []models.LogModuleParams

	for _, user := range validUsers {
		// delete company + user
		deleteInput := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(user.PK),
				},
				"SK": {
					S: aws.String(user.SK),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err := app.SVC.DeleteItem(deleteInput)
		if err != nil {
			// handle error
		}

		tmp := models.TempParam{}
		user, opsError := ops.GetUserByID(user.UserID)
		if opsError != nil && user.PK != "" {
			tmp = models.TempParam{
				Name: "USER#" + user.UserID,
				ID:   user.UserID,
			}
		} else {
			tmp = models.TempParam{
				Name: utils.GenerateFullname(user.FirstName, user.LastName),
				ID:   user.UserID,
			}
		}

		logInfoTmpUsers = append(logInfoTmpUsers, models.LogModuleParams{
			Temp: tmp,
		})
	}

	// generate log
	errorMessages := []string{}

	log := &models.Logs{
		CompanyID: companyID,
		UserID:    requestUserID,
		LogAction: constants.LOG_ACTION_PERMANENTLY_REMOVE_COMPANY_MEMBERS,
		LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			Users: logInfoTmpUsers,
		},
	}

	_, err := ops.InsertLog(log)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating logs for permanently deleting users")
	}

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	return c.RenderJSON(result)
}

type SaveUserIntegrationAccountsInput struct {
	Account string `json:"account,omitempty"`
	Slug    string `json:"slug,omitempty"`
	UserId  string `json:"UserId,omitempty"`
}
type SaveUserIntegrationAccountsInputs struct {
	Data []SaveUserIntegrationAccountsInput `json:"data,omitempty"`
}

// *Draft for Saving User Integration Accounts

// @Summary Save User Integration Accounts
// @Description This endpoint saves the integration accounts for a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.SaveUserIntegrationAccountsRequest true "Save user integration accounts body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/profile/account/:userID [put]
func (c UserController) SaveUserIntegrationAccounts(userID string) revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)

	input := SaveUserIntegrationAccountsInput{}
	c.Params.BindJSON(&input)

	opsErr := ops.SaveUserIntegrationAccounts(ops.SaveUserIntegrationAccountsInput{
		CompanyID:   companyID,
		UserID:      userID,
		Account:     input.Account,
		Integration: input.Slug,
	}, c.Controller)
	if opsErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:           constants.HTTP_STATUS_400,
			Message:        "SaveUserIntegrationAccounts",
			HTTPStatusCode: 400,
		})
	}

	companyMember, opsErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
		CompanyID: companyID,
	}, c.Controller)
	if opsErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:           constants.HTTP_STATUS_400,
			Message:        "SaveUserIntegrationAccounts: Error on GetCompanyMember",
			HTTPStatusCode: 400,
		})
	}
	associatedAccounts := companyMember.AssociatedAccounts
	if associatedAccounts == nil {
		associatedAccounts = make(map[string][]string)
	}
	if input.Account != "" {
		associatedAccounts[input.Slug] = []string{input.Account}
	} else {
		associatedAccounts[input.Slug] = []string{}
	}
	companyMember.AssociatedAccounts = associatedAccounts
	updateCompanyUserInfoInTmp(&companyMember)

	result["account"] = associatedAccounts
	result["accounts"] = companyMember.AssociatedAccounts
	result["userId"] = companyMember.UserID
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	c.Response.Status = 200
	return c.RenderJSON(result)
}

// @Summary Save Users' Integration Accounts
// @Description This endpoint saves integration accounts for multiple users, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.SaveUsersIntegrationAccountsRequest true "Save users' integration accounts body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/accounts/:userID [post]
func (c UserController) SaveUsersIntegrationAccounts(userID string) revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)

	inputs := SaveUserIntegrationAccountsInputs{}
	s := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}

	c.Params.BindJSON(&inputs)
	for _, input := range inputs.Data {
		opsErr := ops.SaveUserIntegrationAccounts(ops.SaveUserIntegrationAccountsInput{
			CompanyID:   companyID,
			UserID:      input.UserId,
			Account:     input.Account,
			Integration: input.Slug,
		}, c.Controller)
		if opsErr != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:           constants.HTTP_STATUS_400,
				Message:        "SaveUserIntegrationAccounts",
				HTTPStatusCode: 400,
			})
		}
		companyUser, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    input.UserId,
			CompanyID: companyID,
		}, c.Controller)
		if opsError != nil {
			// return opsError
		}
		associatedAccounts := companyUser.AssociatedAccounts
		if associatedAccounts == nil {
			associatedAccounts = make(map[string][]string)
		}
		if input.Account != "" {
			associatedAccounts[input.Slug] = []string{input.Account}
		} else {
			associatedAccounts[input.Slug] = []string{}
		}
		companyUser.AssociatedAccounts = associatedAccounts
		updateCompanyUserInfoInTmp(&companyUser)
	}

	users, err := GetCompanyUsersNew(GetCompanyUsersInput{
		CompanyID: companyID,
		Status:    s,
	})
	if err != nil {

	}
	usersWithData, err := GetCompanyUsersInformation(companyID, users, c.Controller)

	if err != nil {

	}
	c.Response.Status = 200
	result["users"] = usersWithData
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// func (c UserController) RemoveUserInCompany(userID string) revel.Result {
// 	result := make(map[string]interface{})
// 	companyID := c.ViewArgs["companyID"].(string)
// 	if companyID == "" {
// 		c.Response.Status = 400
// 		result["error"] = []string{"Missing Company ID field"}
// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
// 	}

// 	// check if user exists in company
// 	var companyUser models.CompanyUser
// 	input := &dynamodb.GetItemInput{
// 		TableName: aws.String(app.TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			"PK": {
// 				S: aws.String(constants.PREFIX_COMPANY + companyID),
// 			},
// 			"SK": {
// 				S: aws.String(constants.PREFIX_USER + userID),
// 			},
// 		},
// 	}
// 	queryResult, err := app.SVC.GetItem(input)
// 	if err != nil {
// 	}

// 	if queryResult.Item == nil {
// 		c.Response.Status = 400
// 		result["error"] = []string{"Company user not found"}
// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
// 	}

// 	err = dynamodbattribute.UnmarshalMap(queryResult.Item, &companyUser)
// 	if err != nil {
// 	}

// 	// update company user status
// 	updateInput := &dynamodb.UpdateItemInput{
// 		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
// 			":s": {
// 				S: aws.String(constants.ITEM_STATUS_INACTIVE),
// 			},
// 			":ua": {
// 				S: aws.String(utils.GetCurrentTimestamp()),
// 			},
// 		},
// 		ExpressionAttributeNames: map[string]*string{
// 			"#s": aws.String("Status"),
// 		},
// 		TableName: aws.String(app.TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			"PK": {
// 				S: aws.String(queryResult.PK),
// 			},
// 			"SK": {
// 				S: aws.String(queryResult.SK),
// 			},
// 		},
// 		UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
// 	}

// 	_, err = app.SVC.UpdateItem(updateInput)
// 	if err != nil {
// 		result["errors"] = []string{err.Error()}
// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
// 		return c.RenderJSON(result)
// 	}
// }

func RemoveUserInCompany(userID, companyID string) error {
	if userID == "" || companyID == "" {
		e := errors.New("Missing Company ID or User ID field")
		return e
	}

	companyUser, err := GetCompanyUser(companyID, userID)
	if err != nil {

		return err
	}

	// update company user status
	updateInput := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s": {
				// S: aws.String(constants.ITEM_STATUS_INACTIVE),
				S: aws.String(constants.ITEM_STATUS_DELETED),
			},
			":da": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
			":pt": {
				S: aws.String(companyUser.Status),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#s":  aws.String("Status"),
			"#pt": aws.String("PreviousStatus"),
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(companyUser.PK),
			},
			"SK": {
				S: aws.String(companyUser.SK),
			},
		},
		UpdateExpression: aws.String("SET #s = :s,  #pt = :pt, DeletedAt = :da"),
	}

	_, err = app.SVC.UpdateItem(updateInput)
	if err != nil {
		return err
	}

	return nil
}

// @Summary Get User Active Connected Group
// @Description This endpoint retrieves the active connected group for the current user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param userID query string true "User ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/groups/count [get]
func (c UserController) GetUserActiveConnectedGroup() revel.Result {
	var connectedGroups []models.Group
	// var connectedGroups = []

	companyID := c.ViewArgs["companyID"].(string)

	userID := c.Params.Query.Get("userID")

	//get the users connected groups (groupmember entities)
	groups, err := GetUserCompanyGroupsNew(companyID, userID)
	if err != nil {
		return c.RenderJSON(err.Error())
	}

	for i := range groups {
		resultGroup := models.Group{}

		group := &groups[i]

		params := &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			KeyConditions: map[string]*dynamodb.Condition{
				"PK": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
						},
					},
				},
				"SK": {
					ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(constants.PREFIX_DEPARTMENT),
						},
					},
				},
			},
		}

		result, err := ops.HandleQueryLimit(params)
		if err != nil {
			return c.RenderJSON(err.Error())
		}

		err = dynamodbattribute.UnmarshalMap(result.Items[0], &resultGroup)
		if err != nil {
			return c.RenderJSON(err.Error())
		}

		connectedGroups = append(connectedGroups, resultGroup)

		//filter the connectedgroups

	}

	var activeGroups []models.Group

	for _, group := range connectedGroups {

		if strings.ToUpper(group.Status) == "ACTIVE" {
			activeGroups = append(activeGroups, group)
		}

	}
	return c.RenderJSON(activeGroups)
}

func (c UserController) ManageAccess() revel.Result {

	results := make(map[string]interface{})
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var payload models.ManageAccessPayload

	invited := false

	// companyID := c.ViewArgs["companyID"].(string)

	c.Params.BindJSON(&payload)

	if payload.EnableAccess {
		user, errGetUserByEmail := ops.GetUserByID(payload.User.UserID)
		if errGetUserByEmail != nil {
			c.Response.Status = 462
			results["reason"] = "EMAIL_NOT_FOUND"
			results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
			return c.RenderJSON(results)
		}

		//** If first time invited in any company, create account
		//
		if user.UserToken != "DONE" {

			invited = true

			updateInput := &dynamodb.UpdateItemInput{
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":us": {
						S: aws.String("DONE"),
					},
					":st": {
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
					":ua": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
				},
				TableName: aws.String(app.TABLE_NAME),
				ExpressionAttributeNames: map[string]*string{
					"#s": aws.String("Status"),
				},
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.Email)),
					},
				},
				UpdateExpression: aws.String("SET #s = :st, UserToken = :us, UpdatedAt = :ua"),
			}
			// run update item
			_, errUpdatePassword := app.SVC.UpdateItem(updateInput)
			if errUpdatePassword != nil {
				c.Response.Status = 500
				results["error"] = errUpdatePassword.Error()
				results["reason"] = "ERR_UPDATING_ITEM"
				results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(results)
			}

			// generate new user token for activation
			userToken := utils.GenerateRandomString(8)

			// update user token

			updateCompanyUserInput := &dynamodb.UpdateItemInput{
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":s": {
						S: aws.String(constants.ITEM_STATUS_PENDING),
					},
					// ":s": {
					// 	S: aws.String(constants.ITEM_STATUS_ACTIVE),
					// },
					":ua": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
					":us": {
						S: aws.String(userToken),
					},
				},
				ExpressionAttributeNames: map[string]*string{
					"#s": aws.String("Status"),
				},
				TableName: aws.String(app.TABLE_NAME),
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(payload.User.PK),
					},
					"SK": {
						S: aws.String(payload.User.SK),
					},
				},
				UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua, UserToken = :us"),
			}

			_, err := app.SVC.UpdateItem(updateCompanyUserInput)
			if err != nil {
				c.Response.Status = 500
				results["error"] = err.Error()
				results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(results)
			}

			var recipients []mail.Recipient

			remoteAddr := strings.Split(c.Request.RemoteAddr, ":")[0]
			frontendUrl, _ := revel.Config.String("url.frontend")

			token, _ := utils.EncodeToJwtToken(jwt.MapClaims{
				"userToken":  userToken,
				"remoteAddr": remoteAddr,
				"userID":     user.UserID,
				"companyID":  user.ActiveCompany,
			})

			recipients = append(recipients, mail.Recipient{
				Name:           user.FirstName + " " + user.LastName,
				Email:          user.Email,
				ActivationLink: frontendUrl + "/verify-email?token=" + token,
			})

			jobs.Now(mail.SendEmail{
				Subject:    "Welcome to SaaSConsole!",
				Recipients: recipients,
				Template:   "verify.html",
			})
		} else {

			updateCompanyUserInput := &dynamodb.UpdateItemInput{
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":s": {
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
					":ua": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
				},
				ExpressionAttributeNames: map[string]*string{
					"#s": aws.String("Status"),
				},
				TableName: aws.String(app.TABLE_NAME),
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(payload.User.PK),
					},
					"SK": {
						S: aws.String(payload.User.SK),
					},
				},
				UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
			}
			_, err := app.SVC.UpdateItem(updateCompanyUserInput)
			if err != nil {
				c.Response.Status = 500
				results["error"] = err.Error()
				results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(results)
			}

			company, opsErr := ops.GetCompanyByID(companyID)
			if opsErr != nil {
			}

			notificationContent := models.NotificationContentType{
				RequesterUserID: userID,
				ActiveCompany:   companyID,
				Message:         "Your access to " + company.CompanyName + " has been enabled",
			}

			_, opsError := ops.CreateNotification(ops.CreateNotificationInput{
				UserID:              payload.User.UserID,
				NotificationType:    constants.ROLE_UPDATE,
				NotificationContent: notificationContent,
				Global:              true,
			}, c.Controller)
			if opsError != nil {
			}

			if payload.SendEmail {
				var recipient []mail.Recipient

				frontendUrl, _ := revel.Config.String("url.frontend")

				recipient = append(recipient, mail.Recipient{
					Name:           user.FirstName + " " + user.LastName,
					Email:          user.Email,
					InviteUserLink: frontendUrl + "/companies/" + company.CompanyID,
					CompanyName:    company.CompanyName,
				})
				jobs.Now(mail.SendEmail{
					Subject:    "You have been invited!",
					Recipients: recipient,
					Template:   "manage_access_invite.html",
				})
			}
		}
	} else {
		updateCompanyUserInput := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":s": {
					S: aws.String(constants.ITEM_STATUS_DEFAULT),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			ExpressionAttributeNames: map[string]*string{
				"#s": aws.String("Status"),
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(payload.User.PK),
				},
				"SK": {
					S: aws.String(payload.User.SK),
				},
			},
			UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(updateCompanyUserInput)
		if err != nil {
			c.Response.Status = 500
			results["error"] = err.Error()
			results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(results)
		}

		company, opsErr := ops.GetCompanyByID(companyID)
		if opsErr != nil {
		}

		notificationContent := models.NotificationContentType{
			RequesterUserID: userID,
			ActiveCompany:   companyID,
			Message:         "Your access to " + company.CompanyName + " has been disabled",
		}

		_, opsError := ops.CreateNotification(ops.CreateNotificationInput{
			UserID:              payload.User.UserID,
			NotificationType:    constants.ROLE_UPDATE,
			NotificationContent: notificationContent,
			Global:              true,
		}, c.Controller)
		if opsError != nil {
		}

		if payload.SendEmail {
			var recipient []mail.Recipient

			user, errGetUserByEmail := ops.GetUserByID(payload.User.UserID)
			if errGetUserByEmail != nil {
				c.Response.Status = 462
				results["reason"] = "EMAIL_NOT_FOUND"
				results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
				return c.RenderJSON(results)
			}

			frontendUrl, _ := revel.Config.String("url.frontend")

			recipient = append(recipient, mail.Recipient{
				Name:           user.FirstName + " " + user.LastName,
				Email:          user.Email,
				InviteUserLink: frontendUrl + "/companies/" + company.CompanyID,
				CompanyName:    company.CompanyName,
			})
			jobs.Now(mail.SendEmail{
				Subject:    "Your access has been changed",
				Recipients: recipient,
				Template:   "manage_access_revoke.html",
			})
		}

	}

	c.Response.Status = 200
	results["message"] = payload.User.FirstName + " " + payload.User.LastName + "'s access has been updated"
	results["invited"] = invited
	results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(results)
}

func (c UserController) RequestCreationOfAccount() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var input models.RequestToCreateAccountPayload
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Email, input.IntegrationId}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "RequestCreationOfAccount Error: Missing required parameter - email,integrationId,integrationSlug",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	requesterInfo, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
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

	//get integration info
	integrationInfo, error := ops.GetIntegrationByID(input.IntegrationId)
	if error != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retreive Integration",
		})
	}

	// get company admins.
	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company Admins",
		})
	}

	for _, companyAdmin := range companyAdmins {
		userNotifications, _ := ops.GetUserNotifications(companyAdmin.UserID, companyID)

		// check if companyadmin has the notification request...
		duplicateFound := false
		for _, notif := range userNotifications {
			if notif.NotificationType == constants.REQUEST_TO_CREATE_ACCOUNT && notif.UserID == companyAdmin.UserID && notif.NotificationContent.Integration.IntegrationID == input.IntegrationId && notif.NotificationContent.RequesterUserID == userID && notif.NotificationContent.ActiveCompany == companyID {
				if input.Email != "" && notif.NotificationContent.Integration.Email == input.Email {
					duplicateFound = true
					break
				}
			}
		}

		if duplicateFound {
			continue
		} else {
			sendCreateAccountRequestNotificationToCompanyAdmin(c, companyAdmin.UserID, companyID, requesterInfo, integrationInfo)
		}
	}

	c.Response.Status = 200
	return c.RenderJSON(nil)
}

func sendCreateAccountRequestNotificationToCompanyAdmin(c UserController, companyAdminUserID, companyID string, requesterUserInfo models.CompanyUser, integration models.Integration) revel.Result {

	notificationContent := models.NotificationContentType{
		RequesterUserID: requesterUserInfo.UserID,
		ActiveCompany:   companyID,
		Integration: (models.NotificationIntegration{
			IntegrationID:   integration.IntegrationID,
			IntegrationSlug: integration.IntegrationSlug,
			IntegrationName: integration.IntegrationName,
		}),
	}

	switch integration.IntegrationSlug {
	case "google-cloud":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to be created an account on Google Cloud."
		break
	case "bitbucket":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to be invited in Bitbucket."
		break
	case "jira":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to be created an account on Jira."
		break
	}

	createdNotification, err := ops.CreateNotification(ops.CreateNotificationInput{
		UserID:              companyAdminUserID,
		NotificationType:    constants.REQUEST_TO_CREATE_ACCOUNT,
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

	// utils.PrintJSON(createdNotification)

	return c.RenderJSON(createdNotification)
}

func (c UserController) RequestMatchingOfAccount() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var input models.RequestToMatchAccountPayload
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.RequesterEmail, input.ToMatchEmail, input.IntegrationId}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "RequestCreationOfAccount Error: Missing required parameter - email,integrationId,integrationSlug",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	requesterInfo, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    userID,
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

	//get integration info
	integrationInfo, error := ops.GetIntegrationByID(input.IntegrationId)
	if error != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retreive Integration",
		})
	}

	// get company admins.
	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company Admins",
		})
	}

	for _, companyAdmin := range companyAdmins {
		userNotifications, _ := ops.GetUserNotifications(companyAdmin.UserID, companyID)

		// check if companyadmin has the notification request...
		duplicateFound := false
		for _, notif := range userNotifications {
			if notif.NotificationType == constants.REQUEST_TO_MATCH_ACCOUNT && notif.UserID == companyAdmin.UserID && notif.NotificationContent.Integration.IntegrationID == input.IntegrationId && notif.NotificationContent.RequesterUserID == userID && notif.NotificationContent.ActiveCompany == companyID {
				if input.RequesterEmail != "" && notif.NotificationContent.Integration.Email == input.RequesterEmail && input.ToMatchEmail == notif.NotificationContent.Integration.ToMatchEmail {
					duplicateFound = true
					break
				}
			}
		}

		if duplicateFound {
			continue
		} else {
			sendMatchExistingAccountNotification(c, companyAdmin.UserID, companyID, requesterInfo, integrationInfo, input)
		}
	}

	c.Response.Status = 200
	return c.RenderJSON(nil)
}

func sendMatchExistingAccountNotification(c UserController, companyAdminUserID, companyID string, requesterUserInfo models.CompanyUser, integration models.Integration, input models.RequestToMatchAccountPayload) revel.Result {

	notificationContent := models.NotificationContentType{
		RequesterUserID: requesterUserInfo.UserID,
		ActiveCompany:   companyID,
		Integration: (models.NotificationIntegration{
			IntegrationID:   integration.IntegrationID,
			IntegrationSlug: integration.IntegrationSlug,
			IntegrationName: integration.IntegrationName,
			ToMatchEmail:    input.ToMatchEmail,
		}),
	}

	switch integration.IntegrationSlug {
	case "google-cloud":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to link their Google Cloud account."
		break
	case "bitbucket":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to link their Bitbucket account."
		break
	case "jira":
		notificationContent.Message = requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to link their Jira account."
		break
	}

	createdNotification, err := ops.CreateNotification(ops.CreateNotificationInput{
		UserID:              companyAdminUserID,
		NotificationType:    constants.REQUEST_TO_MATCH_ACCOUNT,
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

	// utils.PrintJSON(createdNotification)

	return c.RenderJSON(createdNotification)
}

type CreateRemoveUserJobPayload struct {
	Users []UsersJobPayload `json:"users"`
}

type UsersJobPayload struct {
	Email              string `json:"email"`
	IntegrationID      string `json:"integrationID"`
	IntegrationAccount string `json:"integrationAccount"`
	NumberOfDays       string `json:"numberOfDays"`
}

func (c *CreateRemoveUserJobPayload) Validate(v *revel.Validation) {
	utils.ValidateRequired(v, c.Users).Key("Users").Message("Users is required.")
}

func (c UserController) CreateRemoveUserJob() revel.Result {
	results := make(map[string]interface{})

	var data CreateRemoveUserJobPayload
	c.Params.BindJSON(&data)

	data.Validate(c.Validation)
	if c.Validation.HasErrors() {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "Validation Error",
			Errors:         c.Validation.Errors,
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	existingRequests, error := ops.GetExistingRemoveUserJob(constants.JOB_DELETE_GOOGLE_ACCOUNT)
	if error != nil {
		c.Response.Status = 500
		results["error"] = error.Error()
		results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(results)
	}

	existingRequestsSet := make(map[string]bool)

	for _, rUser := range existingRequests {
		existingRequestsSet[rUser.JobData.Email] = true
	}

	var jobs []models.Job
	var inputRequest []*dynamodb.WriteRequest
	var jobsInput *dynamodb.BatchWriteItemInput

	for _, user := range data.Users {
		if _, exists := existingRequestsSet[user.Email]; !exists {
			now := time.Now()
			convertNoOfDays, _ := strconv.Atoi(user.NumberOfDays)
			userNumberOfDays := now.Add(time.Duration(convertNoOfDays) * 24 * time.Hour)
			day := time.Date(userNumberOfDays.Year(), userNumberOfDays.Month(), userNumberOfDays.Day(), 0, 0, 0, 0, now.Location())
			unix := day.Unix()
			dateNow := strconv.FormatInt(unix, 10)
			jobUID := uuid.NewV4().String()
			var jobType string

			integration, err := ops.GetIntegrationByID(user.IntegrationID)
			if err == nil {
				//TODO: add other integration cases

				switch integration.IntegrationSlug {
				case constants.INTEG_SLUG_GOOGLE_CLOUD:
					jobType = constants.JOB_DELETE_GOOGLE_ACCOUNT
				default:

				}
			}

			job := models.Job{
				PK: utils.AppendPrefix(constants.PREFIX_JOB, dateNow+"#"+jobUID),
				SK: utils.AppendPrefix(constants.PREFIX_JOB, jobType),
				JobData: models.JobData{
					Email:              user.Email,
					IntegrationAccount: user.IntegrationAccount,
					IntegrationID:      user.IntegrationID,
					NumberOfDays:       user.NumberOfDays,
				},
				Status:    constants.ITEM_STATUS_ACTIVE,
				CreatedAt: utils.GetCurrentTimestamp(),
			}

			item := map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(job.PK),
				},
				"SK": {
					S: aws.String(job.SK),
				},
				"JobData": {
					M: map[string]*dynamodb.AttributeValue{
						"Email": {
							S: aws.String(job.JobData.Email),
						},
						"IntegrationID": {
							S: aws.String(job.JobData.IntegrationID),
						},
						"IntegrationAccount": {
							S: aws.String(job.JobData.IntegrationAccount),
						},
						"NumberOfDays": {
							S: aws.String(job.JobData.NumberOfDays),
						},
					},
				},
				"Status": {
					S: aws.String(job.Status),
				},
				"CreatedAt": {
					S: aws.String(job.CreatedAt),
				},
			}

			inputRequest = append(inputRequest, &dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: item,
				},
			})

			jobs = append(jobs, job)
		}
	}

	if len(inputRequest) == 0 {
		c.Response.Status = 497
		results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_497)
		results["message"] = "Request already exists."
		return c.RenderJSON(results)
	}

	jobsInput = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			app.TABLE_NAME: inputRequest,
		},
	}

	_, err := app.SVC.BatchWriteItem(jobsInput)
	if err != nil {
		c.Response.Status = 500
		results["error"] = err.Error()
		results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(results)
	}

	c.Response.Status = 200
	results["job"] = jobs
	results["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(results)
}

func RemoveScheduledUser(userID, activeCompany string, job models.Job) *models.ErrorResponse {
	if utils.FindEmptyStringElement([]string{activeCompany, userID}) {
		return &models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "GetCompanyMemberAlt Error: Missing required parameter - companyID/userID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		}
	}

	companyUser, err := ops.GetCompanyMemberAlt(ops.GetCompanyMemberParams{UserID: userID, CompanyID: activeCompany})
	if err != nil {
		//TODO: handle error
	}

	if companyUser.Status != constants.ITEM_STATUS_DELETED && companyUser.Type != constants.USER_TYPE_COMPANY_OWNER {
		//?REMOVE USER FROM GROUPS
		// gucg := ops.GetUserCompanyGroupsInput{
		// 	CompanyID: activeCompany,
		// 	UserID:    companyUser.UserID,
		// }
		// groups, err := ops.GetUserCompanyGroupsAlt(gucg)
		// if err == nil {
		// 	for _, group := range groups {
		// 		dgaur := ops.RemoveUserToGroupInput{
		// 			GroupID: group.GroupID,
		// 			UserID:  companyUser.UserID,
		// 		}
		// 		err := ops.DeleteGroupAndUserRelationAlt(dgaur)
		// 		if err != nil {
		// 			fmt.Println("----- RemoveScheduledUser: ERROR ON REMOVE USER FROM GROUPS ----")
		// 		}
		// 	}
		// }

		//?REMOVE USER FROM COMPANY
		// ruic := ops.RemoveUserInCompanyInput{
		// 	CompanID: activeCompany,
		// 	UserID:   companyUser.UserID,
		// }
		// err = ops.RemoveUserInCompanyAlt(ruic)
		// if err != nil {
		// 	fmt.Println("----- RemoveScheduledUser: ERROR ON REMOVE USER FROM COMPANY ----")
		// }

		//?REMOVE INTEGRATION ACCESS
		jobType := strings.Split(job.SK, "#")[1]
		integ, integErr := ops.GetIntegrationByID(job.JobData.IntegrationID)
		if integErr == nil {
			token, _ := ops.GetCompanyIntegration(activeCompany, integ.IntegrationID)
			if wscontroller.HasAnyIntegrationToken(token) {
				switch jobType {
				case constants.JOB_DELETE_GOOGLE_ACCOUNT:
					rutgi := wscontroller.RemoveUserToGoogleIntegrationInput{
						Email:      job.JobData.IntegrationAccount,
						Token:      token.IntegrationToken,
						TokenExtra: token.IntegrationTokenExtra,
					}
					opsError := wscontroller.RemoveUserToGoogleIntegration(rutgi)

					if opsError == nil {
						//?CREATE LOGS
						log := &models.Logs{
							CompanyID: activeCompany,
							UserID:    userID,
							LogAction: constants.LOG_ACTION_REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS,
							LogType:   constants.ENTITY_TYPE_COMPANY_MEMBER,
							LogInfo: &models.LogInformation{
								Company: &models.LogModuleParams{
									ID: activeCompany,
								},
								Integration: &models.LogModuleParams{
									ID: job.JobData.IntegrationID,
								},
								Users: []models.LogModuleParams{
									{
										Temp: models.TempParam{
											Name: companyUser.FirstName + " " + companyUser.LastName,
											ID:   companyUser.UserID,
										},
									},
								},
							},
						}

						logID, _ := ops.InsertLog(log)
						_ = logID

						//?SAVE INTEGRATION ACCOUNT AS EMPTY
						opsErr := ops.SaveUserIntegrationAccountsAlt(ops.SaveUserIntegrationAccountsInput{
							CompanyID:   activeCompany,
							UserID:      userID,
							Account:     "",
							Integration: constants.INTEG_SLUG_GOOGLE_CLOUD,
						})

						if opsErr != nil {

						}
					}
				//TODO: other integrations
				default:

				}
			}
		}

		//?UPDATE JOB ITEM STATUS
		updateErr := ops.UpdateUserJobStatus(job.PK, job.SK)
		if updateErr != nil {
			//TODO: handle error
		}

		fmt.Println("----- RemoveScheduledUser: DONE REMOVING USER ----")
	}

	return nil
}

type OnBoardingUserInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	JobTitle  string `json:"jobTitle,omitempty"`
}

type OnBoardingWorkEmailInput struct {
	Account     string `json:"account"`
	Integration string `json:"integration"`
	EmailTo     string `json:"emailTo"`
}

type OnBoardingAssociatedAccount struct {
	Integration string `json:"integration"`
	Account     string `json:"account"`
}

type OnBoardingPayload struct {
	User               OnBoardingUserInput          `json:"user"`
	StartDate          string                       `json:"startDate"`
	Groups             *[]string                    `json:"groups"`
	WorkEmail          *OnBoardingWorkEmailInput    `json:"workEmail"`
	AssociatedAccounts *OnBoardingAssociatedAccount `json:"associatedAccounts"`
}

// @Summary Add User
// @Description This endpoint creates a new user in the system, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.AddUserNewRequest true "Add user new body"
// @Success 201 {object} models.LoginSuccessResponse
// @Router /users/new [post]
func (c UserController) AddUserNew() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var payload OnBoardingPayload
	err := c.Params.BindJSON(&payload)
	if err != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJSON("Error on params binding JSON: " + err.Error())
	}

	companyUser, err := c.UserOps.GetCompanyUserByEmail(companyID, payload.User.Email)
	if err != nil {
		c.Response.Status = http.StatusConflict
		return c.RenderJSON("Error while validating user. Please try again.")

		// c.Response.Status = http.StatusInternalServerError
		// return c.RenderJSON(err.Error())
	}

	if companyUser.Email != "" {
		c.Response.Status = http.StatusConflict
		return c.RenderJSON("Email already exists.")
	}

	status := constants.ITEM_STATUS_PENDING
	//* Learn more about time packge https://pkg.go.dev/time#pkg-constants
	formattedDate := utils.GetFormattedDate("01/02/2006")

	numberOfDays, err := utils.GetNumberOfDays(payload.StartDate)
	if err != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJSON(err.Error())
	}

	if formattedDate != payload.StartDate {
		status = constants.ITEM_STATUS_SCHEDULED
	}

	companyData, opsErr := ops.GetCompanyByID(companyID)
	if opsErr != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJSON(opsErr)
	}

	caser := cases.Title(language.English)

	//* Set new user model
	userUUID := utils.GenerateTimestampWithUID()
	userToken := utils.GenerateRandomString(8)
	searchKey := fmt.Sprintf("%s %s %s", payload.User.FirstName, payload.User.LastName, payload.User.Email)
	email := payload.User.Email
	firstName := caser.String(payload.User.FirstName)
	lastName := caser.String(payload.User.LastName)

	//* Instantiate New SaaSConsole User Type
	newUserType := ops.NewSCUser(&models.User{
		PK:             utils.AppendPrefix(constants.PREFIX_USER, userUUID),
		SK:             utils.AppendPrefix(constants.PREFIX_USER, payload.User.Email),
		UserID:         userUUID,
		FirstName:      firstName,
		LastName:       lastName,
		JobTitle:       payload.User.JobTitle,
		Email:          payload.User.Email,
		SearchKey:      strings.ToLower(searchKey),
		Status:         status,
		CreatedAt:      utils.GetCurrentTimestamp(),
		ActiveCompany:  companyID,
		Type:           constants.ENTITY_TYPE_USER,
		UserToken:      userToken,
		IsVerified:     constants.BOOL_FALSE,
		IsTourDone:     constants.BOOL_FALSE,
		BookmarkGroups: []string{},
	})

	//* Insert new user and send email invitation
	newUser, newUserErr := newUserType.CreateUser(companyID, userID, companyData.CompanyName, email, status, c.Controller)
	if newUserErr != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJSON(newUserErr.Error())
	}

	//* Set company user model
	companyUserInput := &models.CompanyUser{
		PK:        utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:        utils.AppendPrefix(constants.PREFIX_USER, newUser.UserID),
		GSI_SK:    utils.AppendPrefix(constants.PREFIX_USER, newUser.SearchKey),
		CompanyID: companyID,
		FirstName: firstName,
		LastName:  lastName,
		JobTitle:  newUser.JobTitle,
		Email:     newUser.Email,
		SearchKey: strings.ToLower(searchKey),
		UserType:  constants.USER_TYPE_COMPANY_MEMBER,
		UserID:    newUser.UserID,
		Handler:   userID,
		Status:    status,
		CreatedAt: utils.GetCurrentTimestamp(),
		Type:      constants.ENTITY_TYPE_COMPANY_MEMBER,
	}

	associatedAccount := make(map[string][]string)
	if payload.AssociatedAccounts != nil {
		associatedAccount[payload.AssociatedAccounts.Integration] = []string{payload.AssociatedAccounts.Account}
		companyUserInput.AssociatedAccounts = associatedAccount
	}

	//* Instantiate New SaaSConsole Company User Type
	newCompanyUserType := ops.NewSCCompanyUser(companyUserInput)
	//* Insert company user and create logs
	companyUserErr := newCompanyUserType.AddUserToCompany(companyID, userID, userID, status, newUser)
	if companyUserErr != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJSON(companyUserErr.Error())
	}

	//* Check start date
	if formattedDate != payload.StartDate {
		//* Get the number of days date string
		dateString := utils.GetDateNowUnixFormat(numberOfDays)
		JobUUID := uuid.NewV4().String()

		//* Set on boarding user for updating the status on the cron job date
		var jobOnBoardingUsers []models.UserCronJobData
		jobOnBoardingUsers = append(jobOnBoardingUsers, models.UserCronJobData{
			UserID: newUser.UserID,
			Email:  newUser.Email,
		})

		groups := []string{}
		if payload.Groups != nil {
			groups = *payload.Groups
		}

		//* Set cron job for on boarding user
		cronJobInput := &models.CronJob{
			PK:            utils.AppendPrefix(constants.PREFIX_JOB, dateString+"#"+JobUUID),
			SK:            utils.AppendPrefix(constants.PREFIX_JOB, constants.JOB_GROUP_MEMBERS),
			CompanyID:     companyID,
			Status:        constants.ITEM_STATUS_PENDING,
			SelectedDate:  dateString,
			Type:          constants.JOB_ON_BOARDING_USER,
			Users:         jobOnBoardingUsers,
			CurrentUserID: userID,
			CreatedAt:     utils.GetCurrentTimestamp(),
			NumberOfDays:  strconv.Itoa(numberOfDays),
			// SelectedGroups: *payload.Groups,
			SelectedGroups: groups,
		}

		//* Save cron job user
		cronJobObj := ops.NewCronJob(cronJobInput)
		cronJobObj.CreateCronJob()

		//* Set member cron job data for adding the user to the groups
		var memberJobData []models.MemberCronJobData
		memberJobData = append(memberJobData, models.MemberCronJobData{
			MemberID:   newUser.UserID,
			MemberType: constants.MEMBER_TYPE_USER,
		})

		//* Set cron job for adding the user to the groups
		cronJobGroupInput := &models.CronJob{
			PK:            utils.AppendPrefix(constants.PREFIX_JOB, dateString+"#"+JobUUID),
			SK:            utils.AppendPrefix(constants.PREFIX_JOB, constants.JOB_GROUP_MEMBERS),
			CompanyID:     companyID,
			Status:        constants.ITEM_STATUS_PENDING,
			SelectedDate:  dateString,
			Type:          constants.JOB_ADD_GROUP_MEMBERS,
			Members:       memberJobData,
			CurrentUserID: userID,
			CreatedAt:     utils.GetCurrentTimestamp(),
			NumberOfDays:  strconv.Itoa(numberOfDays),
			// SelectedGroups: *payload.Groups,
			SelectedGroups: groups,
		}

		//* Save cron job group
		cronJobGroupType := ops.NewCronJob(cronJobGroupInput)
		cronJobGroupType.CreateCronJob()

	} else {
		if payload.WorkEmail != nil {
			//* Get connected integration in the company using company ID and integration slug
			integration, err := ops.GetConnectedIntegrations(companyID, payload.WorkEmail.Integration)
			if err != nil {
				c.Response.Status = http.StatusInternalServerError
				return c.RenderJSON(err.Error())
			}

			if payload.WorkEmail.Integration == constants.INTEG_SLUG_GOOGLE_CLOUD {
				config, err := googleoperations.GetGoogleConfig()
				if err != nil {
					c.Response.Status = http.StatusInternalServerError
					return c.RenderJSON("Unable to parse client secret file to config")
				}

				integrationUserAccount := &ops.IntegrationUserAccount{
					Email:     payload.WorkEmail.Account,
					FirstName: firstName,
					LastName:  lastName,
					EmailTo:   payload.WorkEmail.EmailTo,
				}

				googleCloudOAuth := &ops.IntegrationAccountToken{
					Config:     config,
					Token:      integration.IntegrationToken,
					TokenExtra: &integration.IntegrationTokenExtra,
				}

				googleCloudAccountType := ops.NewGoogleCloudAccount(integrationUserAccount, googleCloudOAuth)
				//* Create google account
				googleAdminUser, err := googleCloudAccountType.CreateAccount(companyID, newUser.UserID, associatedAccount, c.Controller)
				if err != nil {
					c.Response.Status = http.StatusInternalServerError
					return c.RenderJSON(map[string]interface{}{
						"message":      "error creating google cloud account",
						"errorMessage": err.Error(),
					})
				}
				_ = googleAdminUser
			} else if payload.WorkEmail.Integration == constants.INTEG_SLUG_OFFICE_365 {
				integrationUserAccount := &ops.IntegrationUserAccount{
					Email:     payload.WorkEmail.Account,
					FirstName: firstName,
					LastName:  lastName,
					FullName:  fmt.Sprintf("%s %s", firstName, lastName),
					EmailTo:   payload.WorkEmail.EmailTo,
				}

				office365Auth := &ops.IntegrationAccountToken{
					Token:      integration.IntegrationToken,
					TokenExtra: &integration.IntegrationTokenExtra,
				}
				//* Instantiate MS Azure Office Account Type
				msOfficeAccountType := ops.NewMSAzureAccount(integrationUserAccount, office365Auth)
				//* Create Azure User
				_, officeErr := msOfficeAccountType.CreateAzureUser(companyID)
				if officeErr != nil {
					c.Response.Status = http.StatusInternalServerError
					return c.RenderJSON(map[string]interface{}{
						"message":      "error creating micosoft azure ad account",
						"errorMessage": officeErr.Error(),
					})
				}
			}
		}

		//* Add user to selected group(s)
		if payload.Groups != nil {
			_, addGroupMemberErr := newUserType.AddUserToGroups(companyID, userID, *payload.Groups)
			if addGroupMemberErr != nil {
				c.Response.Status = http.StatusInternalServerError
				return c.RenderJSON(map[string]interface{}{
					"message":      "error adding user to groups",
					"errorMessage": addGroupMemberErr.Error(),
				})
			}
		}
	}
	_ = companyID
	_ = userID

	c.Response.Status = http.StatusCreated
	return c.RenderJSON(payload)
}

type CreateCronJobUserPayload struct {
	DateOfRemoval string `json:"dateOfRemoval"`
	UserID        string `json:"userID"`
	Email         string `json:"email"`
	Type          string `json:"type"`
}

// @Summary Create Cron Job for User
// @Description This endpoint creates a cron job related to a user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.CreateCronJobForUserRequest true "Create cron job for user body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/cron_job [post]
func (c UserController) CreateCronJobUser() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	var payload CreateCronJobUserPayload
	c.Params.BindJSON(&payload)

	format := "01/02/2006"
	formattedDate := utils.GetFormattedDate(format)

	if formattedDate != payload.DateOfRemoval {
		//* Get the number of days date string
		numberOfDays, err := utils.GetNumberOfDays(payload.DateOfRemoval)
		if err != nil {
			return c.RenderJSON(err.Error())
		}
		dateString := utils.GetDateNowUnixFormat(numberOfDays)
		JobUUID := uuid.NewV4().String()

		var jobOnBoardingUsers []models.UserCronJobData
		jobOnBoardingUsers = append(jobOnBoardingUsers, models.UserCronJobData{
			UserID: payload.UserID,
			Email:  payload.Email,
		})

		//* Set cron job user
		cronJobInput := &models.CronJob{
			PK:            utils.AppendPrefix(constants.PREFIX_JOB, dateString+"#"+JobUUID),
			SK:            utils.AppendPrefix(constants.PREFIX_JOB, constants.JOB_GROUP_MEMBERS),
			CompanyID:     companyID,
			Status:        constants.ITEM_STATUS_PENDING,
			SelectedDate:  dateString,
			Type:          payload.Type,
			Users:         jobOnBoardingUsers,
			CurrentUserID: userID,
			CreatedAt:     utils.GetCurrentTimestamp(),
			NumberOfDays:  strconv.Itoa(numberOfDays),
		}

		//* Save cron job user
		cronJobObj := ops.NewCronJob(cronJobInput)
		cronJobObj.CreateCronJob()

		c.Response.Status = 201
	}

	return nil
}

// @Summary Check if Email is Duplicate
// @Description This endpoint checks if the provided email is already in use by another user, returning a 200 OK upon success.
// @Tags users
// @Produce json
// @Param body body models.CheckEmailDuplicateRequest true "Check email duplicate body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /users/check-email [post]
func (c UserController) CheckEmailDuplicate() revel.Result {
	var payload struct {
		Email string `json:"email"`
	}

	companyID := c.ViewArgs["companyID"].(string)

	c.Params.BindJSON(&payload)

	err := IsEmailUniqueInCompany(payload.Email, companyID, "", "", "", false, []string{})
	if err != nil {
		c.Response.Status = http.StatusConflict
		return c.RenderJSON("Email already exists.")
	}

	c.Response.Status = http.StatusAccepted
	return nil
}
