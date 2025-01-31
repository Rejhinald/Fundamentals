package controllers

import (
	"encoding/json"
	"errors"
	"grooper/app"
	"grooper/app/constants"
	stripeoperations "grooper/app/integrations/stripe"
	"grooper/app/mail"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"mime/multipart"
	"strconv"
	"strings"

	// "time"

	"grooper/app/cdn"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dgrijalva/jwt-go"
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	// uuid "github.com/satori/go.uuid"
)

type CompanyController struct {
	*revel.Controller
	DepartmentController
}

/*
****************
AddCompany()
Creates a company with initial Department and UserMember
Body:
user_id - required
company_name - required
department_name - required
department_description - optional
*need to implement the suggestion of QA for improvements *
****************
*/

// @Summary Create Company
// @Description This endpoint allows the creation of a new company in the system, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param body body models.AddCompanyRequest true "Add company body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies [post]
func (c CompanyController) AddCompany() revel.Result {

	companyID := utils.GenerateTimestampWithUID()
	companyName := utils.TrimSpaces(c.Params.Form.Get("company_name"))
	companyDescription := utils.TrimSpaces(c.Params.Form.Get("company_description"))
	currentUser := c.ViewArgs["userID"].(string)
	image := c.Params.Files["display_photo"]
	setupWizardStatus := utils.TrimSpaces(c.Params.Form.Get("setup_wizard_status"))

	if setupWizardStatus == "" {
		setupWizardStatus = constants.WIZARD_STATUS_DONE
	}
	//Get current timestamp
	currentTime := utils.GetCurrentTimestamp()

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	// result := IsCompanyNameUnique(companyName);
	// if result == true {
	// 	data["errors"] = "The company name already exist"
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_477)
	// 	return c.RenderJSON(data)
	// }

	// check if company exists
	// unique := IsCompanyNameUnique(companyName)
	// if !unique {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_477)
	// 	return c.RenderJSON(data)
	// }

	displayPhoto := ""
	if image != nil {
		photo, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_COMPANY)
		displayPhoto = photo
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
	}

	company := models.Company{
		PK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyName),
		CompanyID:          companyID,
		CompanyName:        companyName,
		CompanyDescription: companyDescription,
		DisplayPhoto:       displayPhoto,
		Status:             constants.ITEM_STATUS_ACTIVE,
		SetupWizardStatus:  setupWizardStatus,
		CreatedAt:          currentTime,
		Type:               constants.ENTITY_TYPE_COMPANY,
		SearchKey:          strings.ToLower(companyName),
	}

	company.Validate(c.Validation, constants.SERVICE_TYPE_ADD_COMPANY)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	// Check user ID exists in grooper table
	user, opsError := ops.GetUserByID(currentUser)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(opsError.Status.Code)
		return c.RenderJSON(data)
	}

	// Checks if no user found
	if user.PK == "" {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}

	av, errCompany := dynamodbattribute.MarshalMap(company)
	if errCompany != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		data["error"] = errCompany.Error()
		return c.RenderJSON(data)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, companyErr := app.SVC.PutItem(input)
	if companyErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		data["error"] = companyErr.Error()
		return c.RenderJSON(data)
	}

	/*****************
	ADD COMPANY USER
	*****************/

	searchKey := user.FirstName + " " + user.LastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)

	companyUser := models.CompanyUser{
		PK:            utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:            utils.AppendPrefix(constants.PREFIX_USER, currentUser),
		GSI_SK:        utils.AppendPrefix(constants.PREFIX_USER, searchKey),
		UserID:        currentUser,
		CompanyID:     companyID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		JobTitle:      user.JobTitle,
		Email:         user.Email,
		ContactNumber: user.ContactNumber,
		DisplayPhoto:  user.DisplayPhoto,
		SearchKey:     searchKey,
		UserType:      constants.USER_TYPE_COMPANY_OWNER,
		Handler:       user.UserID,
		Status:        constants.ITEM_STATUS_ACTIVE,
		CreatedAt:     utils.GetCurrentTimestamp(),
		Type:          constants.ENTITY_TYPE_COMPANY_MEMBER,
	}

	av, err := dynamodbattribute.MarshalMap(companyUser)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		data["error"] = err.Error()
		return c.RenderJSON(data)
	}
	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		data["error"] = err.Error()
		return c.RenderJSON(data)
	}

	//assign user role when new user signed up
	item := models.UserRole{
		PK:        utils.AppendPrefix(constants.PREFIX_USER, user.UserID),
		SK:        utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"), utils.AppendPrefix(constants.PREFIX_COMPANY, companyUser.CompanyID)),
		UserID:    user.UserID,
		RoleID:    "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e",
		CompanyID: companyUser.CompanyID,
		Type:      constants.ENTITY_TYPE_USER_ROLE,
	}

	av, err = dynamodbattribute.MarshalMap(item)
	if err != nil {
		data["error"] = "Error at marshalmap"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		data["error"] = "Cannot assign role due to server error"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith created CompanyX
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		// data["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	data["company"] = company
	// data["companyID"] = company.CompanyID

	return c.RenderJSON(data)
}

// @Summary Create Company with SSO
// @Description This endpoint allows the creation of a new company within the system using Single Sign-On, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param body body models.SSOAddCompanyRequest true "Single sign-on add company body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/sso [post]
func (c CompanyController) SSOAddCompany() revel.Result {
	companyID := utils.GenerateTimestampWithUID()
	companyName := utils.TrimSpaces(c.Params.Form.Get("company_name"))
	companyDescription := utils.TrimSpaces(c.Params.Form.Get("company_description"))
	currentUser := c.ViewArgs["userID"].(string)
	image := c.Params.Files["display_photo"]
	setupWizardStatus := utils.TrimSpaces(c.Params.Form.Get("setup_wizard_status"))

	if setupWizardStatus == "" {
		setupWizardStatus = constants.WIZARD_STATUS_DONE
	}
	//Get current timestamp
	currentTime := utils.GetCurrentTimestamp()

	//Make a data interface to return as JSON
	data := make(map[string]interface{})

	displayPhoto := ""
	if image != nil {
		photo, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_COMPANY)
		displayPhoto = photo
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
	}

	company := models.Company{
		PK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyName),
		CompanyID:          companyID,
		CompanyName:        companyName,
		CompanyDescription: companyDescription,
		DisplayPhoto:       displayPhoto,
		Status:             constants.ITEM_STATUS_ACTIVE,
		SetupWizardStatus:  setupWizardStatus,
		CreatedAt:          currentTime,
		Type:               constants.ENTITY_TYPE_COMPANY,
		SearchKey:          strings.ToLower(companyName),
	}

	company.Validate(c.Validation, constants.SERVICE_TYPE_ADD_COMPANY)
	if c.Validation.HasErrors() {
		data["errors"] = c.Validation.Errors
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(data)
	}

	// Check user ID exists in grooper table
	user, opsError := ops.GetUserByID(currentUser)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(opsError.Status.Code)
		return c.RenderJSON(data)
	}

	// Checks if no user found
	if user.PK == "" {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(data)
	}

	av, errCompany := dynamodbattribute.MarshalMap(company)
	if errCompany != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		data["error"] = errCompany.Error()
		return c.RenderJSON(data)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, companyErr := app.SVC.PutItem(input)
	if companyErr != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		data["error"] = companyErr.Error()
		return c.RenderJSON(data)
	}

	/*****************
	ADD COMPANY USER
	*****************/

	searchKey := user.FirstName + " " + user.LastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)

	companyUser := models.CompanyUser{
		PK:            utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:            utils.AppendPrefix(constants.PREFIX_USER, currentUser),
		GSI_SK:        utils.AppendPrefix(constants.PREFIX_USER, searchKey),
		UserID:        currentUser,
		CompanyID:     companyID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		JobTitle:      user.JobTitle,
		Email:         user.Email,
		ContactNumber: user.ContactNumber,
		DisplayPhoto:  user.DisplayPhoto,
		SearchKey:     searchKey,
		UserType:      constants.USER_TYPE_COMPANY_OWNER,
		Handler:       user.UserID,
		Status:        constants.ITEM_STATUS_ACTIVE,
		CreatedAt:     utils.GetCurrentTimestamp(),
		Type:          constants.ENTITY_TYPE_COMPANY_MEMBER,
	}

	av, err := dynamodbattribute.MarshalMap(companyUser)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		data["error"] = err.Error()
		return c.RenderJSON(data)
	}
	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		data["error"] = err.Error()
		return c.RenderJSON(data)
	}

	//assign user role when new user signed up
	item := models.UserRole{
		PK:        utils.AppendPrefix(constants.PREFIX_USER, user.UserID),
		SK:        utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"), utils.AppendPrefix(constants.PREFIX_COMPANY, companyUser.CompanyID)),
		UserID:    user.UserID,
		RoleID:    "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e",
		CompanyID: companyUser.CompanyID,
		Type:      constants.ENTITY_TYPE_USER_ROLE,
	}

	av, err = dynamodbattribute.MarshalMap(item)
	if err != nil {
		data["error"] = "Error at marshalmap"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		data["error"] = "Cannot assign role due to server error"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs []*models.Logs
	// message: John Smith created CompanyX
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    user.UserID,
		LogAction: constants.LOG_ACTION_ESTABLISH_COMPANY,
		LogType:   constants.ENTITY_TYPE_AUTH,
		LogInfo: &models.LogInformation{
			User: &models.LogModuleParams{
				ID: user.UserID,
			},
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		// data["message"] = "error while creating logs"
	}

	//Put the newly created company to the user
	cac := ops.ChangeActiveCompanyarams{UserID: user.UserID, CompanyID: companyID, Email: user.Email}
	opsError = ops.ChangeActiveCompany(cac, c.Controller)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	//Create a New token containing the newly created company and the user

	user, _, _ = GetCurrentUser(companyID, user.UserID, c.Controller)
	data["user"] = user
	data["token"] = ops.EncodeToken(user)
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	data["company"] = company
	// data["companyID"] = company.CompanyID

	return c.RenderJSON(data)
}

/*
****************
GetCompanies()
- Returns all companies in the DB
****************
*/

// @Summary Get Companies
// @Description This endpoint retrieves a list of companies with various filtering and sorting options, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies [get]
func (c CompanyController) GetAllCompanies() revel.Result {
	data := make(map[string]interface{})
	companies := []models.Company{}

	result, err := ops.GetAll(constants.ENTITY_TYPE_COMPANY, constants.INDEX_NAME_GET_COMPANIES)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(data)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result, &companies)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	for i, company := range companies {
		// include generated DisplayPhoto
		displayPhoto := ""
		if company.DisplayPhoto != "" {
			resizedPhoto := utils.ChangeCDNImageSize(company.DisplayPhoto, "_100") // todo dynamic
			file, err := cdn.GetImageFromStorage(resizedPhoto)
			if err == nil {
				displayPhoto = file
			}
		}
		companies[i].DisplayPhoto = displayPhoto
	}

	data["companies"] = companies
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

// ListAllCompanies - Returns all companies in SaaSConsole, handles the dynamodb query limit
func ListAllCompanies() ([]models.Company, error) {
	var companies []models.Company
	result, err := ops.GetAll(constants.ENTITY_TYPE_COMPANY, constants.INDEX_NAME_GET_COMPANIES)
	if err != nil {
		return companies, err
	}
	err = dynamodbattribute.UnmarshalListOfMaps(result, &companies)
	if err != nil {
		return companies, err
	}
	return companies, nil
}

/*
****************
Get Companies
Fetch companies based on user
Params:
user_id - for fetching users inside a certain company
key - use for searching
last_evaluated_key - use for pagination
limit - limit number of items per page
****************
*/

// @Summary Get User Companies
// @Description This endpoint retrieves a list of companies the user is a part of, returning a 200 OK upon success. If the UserID field is empty, the server responds with a 400 Bad Request.
// @Tags companies
// @Produce json
// @Param userID path string true "User ID"
// @Param include query string false "Include related data: integrations, departments, or users"
// @Param key query string false "Search Key"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param limit query int false "Limit the number of results"
// @Param logo_size query int false "Logo size"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/user/:userID [get]
func (c CompanyController) GetUserCompanies(userID string) revel.Result {

	//Parameters
	include := c.Params.Query.Get("include")
	searchKey := c.Params.Query.Get("key")
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	limit := c.Params.Query.Get("limit")
	logoSize := c.Params.Query.Get("logo_size")

	//Make a data interface to return as JSON
	result := make(map[string]interface{})

	// required user id
	if userID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
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

	companies := []models.UserCompany{}
	lastEvaluatedKey := models.UserCompany{}
	var err error

	companies, lastEvaluatedKey, err = GetCompaniesByUser(searchKey, paramLastEvaluatedKey, userID, pageLimit, logoSize)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}
	if include != "" {
		for i, company := range companies {
			if strings.Contains(include, "integrations") {
				var integrations []models.Integration

				params := &dynamodb.QueryInput{
					TableName: aws.String(app.TABLE_NAME),
					KeyConditions: map[string]*dynamodb.Condition{
						"PK": {
							ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
							AttributeValueList: []*dynamodb.AttributeValue{
								{
									S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, company.CompanyID)),
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
				if len(integrations) != 0 {
					for j, integration := range integrations {
						n, err := GetIntegrationByID(integration.IntegrationID)
						if err == nil {
							integrations[j].IntegrationName = n.IntegrationName
							integrations[j].IntegrationDescription = n.IntegrationDescription
							integrations[j].DisplayPhoto = n.DisplayPhoto
						}
					}
				}

				companies[i].Integrations = integrations
			}
			if strings.Contains(include, "departments") {
				res, err := GetDepartmentsByCompany(company.CompanyID, "", "", "")
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
					return c.RenderJSON(result)
				}
				departments := []models.CompanyDepartment{}
				err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &departments)
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
					return c.RenderJSON(result)
				}
				companies[i].Departments = departments
			}

			if strings.Contains(include, "users") {
				res, err := GetAllCompanyUsers(company.CompanyID, "", "", []string{})
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
					return c.RenderJSON(result)
				}
				users := []models.CompanyUserData{}
				users = res
				companies[i].Users = users
			}
		}
	}
	// check if there's a searchKey and no companies
	if searchKey != "" && len(companies) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	result["companies"] = companies
	result["lastEvaluatedKey"] = lastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
Get all departments of specific company
Can be used for including the departments of the company on the response
****************
*/
func GetDepartmentsByCompany(companyID, status, searchKey, exclusiveStartKey string) (*dynamodb.QueryOutput, error) {
	if status == "" {
		status = constants.ITEM_STATUS_ACTIVE
	}

	searchKey = strings.ToLower(searchKey)

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_DEPARTMENT),
					},
				},
			},
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(status),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"CompanyID": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"SearchKey": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(searchKey),
					},
				},
			},
		},
		IndexName:         aws.String(constants.INDEX_NAME_GET_DEPARTMENTS),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(models.UserCompany{}, exclusiveStartKey),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return nil, e
	}
	return result, nil
}

/*
****************
Get all users of specific company
Can be used for including the departments of the company on the response
****************
*/
func GetAllCompanyUsers(companyID, searchKey, exclusiveStartKey string, status []string) ([]models.CompanyUserData, error) {

	if len(status) == 0 {
		status = []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_DEFAULT}
	}

	searchKey = strings.ToLower(searchKey)
	users := []models.CompanyUserData{}
	users2 := []models.CompanyUserData{}
	lastEvaluatedKey := models.CompanyUserData{}

	s, err := dynamodbattribute.MarshalList(status)
	if err != nil {
		// )
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
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: s,
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return users, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, e
	}

	key := result.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return users, e
	}

	for _, user := range users {
		userData, err := ops.GetUserData(user.UserID, searchKey)
		if err != nil {
			e := errors.New(err.Error())
			return users, e
		}

		if userData.UserID != "" {

			displayPhoto := ""
			if userData.DisplayPhoto != "" {
				resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
				file, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					displayPhoto = file
				}
			}

			users2 = append(users2, models.CompanyUserData{
				PK:           userData.PK,
				UserID:       userData.UserID,
				FirstName:    userData.FirstName,
				LastName:     userData.LastName,
				DisplayPhoto: displayPhoto,
				Email:        userData.Email,
				Status:       user.Status,
			})
		}
	}

	return users2, nil
}

/*
*****************************
Get Companies By User
Used for fetching companies where the user belongs to
Parameters:
exclusiveStartKey
userID
pageLimit
*****************************
*/
func GetCompaniesByUser(searchKey, exclusiveStartKey, userID string, pageLimit int64, logoSize string) ([]models.UserCompany, models.UserCompany, error) {

	companies := []models.UserCompany{}
	companies2 := []models.UserCompany{}
	lastEvaluatedKey := models.UserCompany{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_COMPANY),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: []*dynamodb.AttributeValue{
					// {
					// 	S: aws.String(constants.ITEM_STATUS_DEFAULT),
					// },
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
				},
			},
		},

		IndexName:         aws.String(constants.INDEX_NAME_INVERTED_INDEX),
		Limit:             aws.Int64(pageLimit),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return companies, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &companies)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return companies, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return companies, lastEvaluatedKey, e
	}

	for _, company := range companies {
		companyData, opsError := ops.GetCompanyByID(company.CompanyID)
		if opsError != nil {
			e := errors.New(opsError.Status.Message)
			return companies, lastEvaluatedKey, e
		}

		if companyData.DisplayPhoto != "" {
			resizedPhoto := utils.ChangeCDNImageSize(companyData.DisplayPhoto, logoSize)

			fileName, err := cdn.GetImageFromStorage(resizedPhoto)
			if err != nil {
				e := errors.New(err.Error())
				return companies, lastEvaluatedKey, e
			}
			companyData.DisplayPhoto = fileName
		}

		if companyData.CompanyID != "" {
			companies2 = append(companies2, models.UserCompany{
				CompanyID:          companyData.CompanyID,
				CompanyName:        companyData.CompanyName,
				CompanyDescription: companyData.CompanyDescription,
				DisplayPhoto:       companyData.DisplayPhoto,
				Status:             companyData.Status,
				CreatedAt:          companyData.CreatedAt,
				UpdatedAt:          companyData.UpdatedAt,
				SearchKey:          companyData.SearchKey,
				Integrations:       []models.Integration{},
				Departments:        []models.CompanyDepartment{},
				Users:              []models.CompanyUserData{},
			})
		}
	}

	return companies2, lastEvaluatedKey, nil
}

/*
****************
Get Company By ID
Params:
include - value(s): departments, groups
****************
*/

// @Summary Get Company Info
// @Description This endpoint retrieves detailed information about a specific company, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Param include query string false "Include related data: groups or departments"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID [get]
func (c CompanyController) GetCompany(companyID string) revel.Result {
	//Parameters
	include := c.Params.Query.Get("include")
	logoSize := c.Params.Query.Get("logo_size")
	//searchKey 				:= c.Params.Query.Get("key")
	//paramLastEvaluatedKey 	:= c.Params.Query.Get("last_evaluated_key")
	//limit 					:= c.Params.Query.Get("limit")

	//Make a data interface to return as JSON
	result := make(map[string]interface{})
	// required company id
	if companyID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	// company := models.Company{}
	company, err := ops.GetCompanyByID(companyID)
	if err != nil {
		c.Response.Status = 400
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	if company.PK == "" {
		c.Response.Status = 404
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}
	// company.Departments 	= []models.Department{}
	// company.Groups 			= []models.DepartmentGroup{}
	if company.DisplayPhoto != "" {
		resizedPhoto := utils.ChangeCDNImageSize(company.DisplayPhoto, logoSize)

		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err != nil {
			result["error"] = err.Error()
			return c.RenderJSON(result)
		}

		company.DisplayPhoto = fileName
	}

	if include != "" {
		if strings.Contains(include, "departments") {
			res, err := GetDepartmentsByCompany(company.CompanyID, "", "", "")
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}
			departments := []models.Department{}
			err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &departments)
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}
			company.Departments = departments
		}
		// if company departments is not empty
		if len(company.Departments) != 0 {
			if strings.Contains(include, "groups") {
				groups := []models.DepartmentGroup{}
				for i, department := range company.Departments {

					departmentGroups, opsError := ops.GetGroupsByDepartmentID(department.DepartmentID, companyID)
					if opsError != nil {
						c.Response.Status = opsError.HTTPStatusCode
						return c.RenderJSON(opsError)
					}
					// departmentGroups := []models.DepartmentGroup{}
					// err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &departmentGroups)
					// if err != nil {
					// 	result["status"] = utils.GetHTTPStatus(err.Error())
					// 	return c.RenderJSON(result)
					// }
					company.Departments[i].Groups = *departmentGroups
					for j, group := range *departmentGroups {
						users, err := GetDepartmentGroupUsers("", group.GroupID)
						if err != nil {
							result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
							return c.RenderJSON(result)
						}
						if len(users) != 0 {
							for l, user := range users {
								// userData, err := ops.GetUserData(user.MemberID, "")
								userData, err := ops.GetUserData(user.UserID, "")
								if err != nil {
									result["status"] = utils.GetHTTPStatus(err.Error())
									return c.RenderJSON(result)
								}
								displayPhoto := ""
								if userData.DisplayPhoto != "" {
									resizedPhoto := utils.ChangeCDNImageSize(userData.DisplayPhoto, "_100") // todo dynamic
									file, err := cdn.GetImageFromStorage(resizedPhoto)
									if err == nil {
										displayPhoto = file
									}
								}
								users[l] = models.DepartmentGroupUser{
									UserID: userData.UserID,
									// MemberID:     user.MemberID,
									FirstName:    userData.FirstName,
									LastName:     userData.LastName,
									DisplayPhoto: displayPhoto,
								}
							}
						}

						(*departmentGroups)[j].Users = users
					}
					groups = append(*departmentGroups)
				}
				company.Groups = groups
			}
		}
	}

	result["company"] = company
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)

}

/*
****************
// TODO no design yet if company's status and name can be changed
Updating of Company ORIGINAL
Params:
companyID

Form Body:
company_status -
company_name
****************
*/

// @Summary Update Company
// @Description This endpoint allows a user to update the company details, returning a 200 OK upon success. If the company user does not have permission, a 401 Unauthorized status is returned.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Param body body models.UpdateCompanyRequest true "Update company body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID [put]
func (c CompanyController) UpdateCompany(companyID string) revel.Result {

	data := make(map[string]interface{})

	// companyStatus := c.Params.Form.Get("company_status")
	companyName := utils.TrimSpaces(c.Params.Form.Get("company_name"))
	companyDescription := utils.TrimSpaces(c.Params.Form.Get("company_description"))
	image := c.Params.Files["display_photo"]
	userID := c.ViewArgs["userID"].(string)
	setupWizardStatus := c.Params.Form.Get("setup_wizard_status")

	//CHECKED PERMISSION
	checked := ops.CheckPermissions(constants.EDIT_COMPANY, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(data)
	}

	// get company
	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	//CHECK COMPANY NAME EXISTS in USER"S COMPANIES
	isCompanyUnique := IsCompanyNameUnique(companyName, userID)
	if !isCompanyUnique && company.CompanyName != companyName {
		data["errors"] = "The company name already exists."
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_477)
		return c.RenderJSON(data)
	}

	displayPhoto := ""
	if image != nil {
		// if image file is not empty
		dp, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_COMPANY)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
		displayPhoto = dp
	} else {
		// if image file is empty
		// = c.Params.Form.Get("display_photo")
		displayPhoto = company.DisplayPhoto
	}

	// check if company name is unique
	// isCompanyUnique := IsCompanyNameUnique(companyName)
	// if !isCompanyUnique && company.CompanyName != companyName {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_477)
	// 	return c.RenderJSON(data)
	// }

	wizardStatus := setupWizardStatus
	if wizardStatus == "" {
		wizardStatus = constants.WIZARD_STATUS_DONE
	}

	if companyName != company.CompanyName {
		// delete company data
		PK := constants.PREFIX_COMPANY + companyID
		SK := company.SK

		_, deleteErr := ops.DeleteByPartitionKey(PK, SK)
		if deleteErr != nil {
			data["error"] = "Got error calling DeleteItem"
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
		//structure copy of company data with updated SK
		companyData := models.Company{
			PK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
			SK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyName),
			CompanyID:          companyID,
			CompanyName:        companyName,
			CompanyDescription: companyDescription,
			DisplayPhoto:       displayPhoto,
			Status:             company.Status,
			CreatedAt:          company.CreatedAt,
			Type:               company.Type,
			SetupWizardStatus:  wizardStatus,
			SearchKey:          strings.ToLower(companyName),
			UpdatedAt:          utils.GetCurrentTimestamp(),
		}

		companyData.Validate(c.Validation, constants.SERVICE_TYPE_ADD_COMPANY)
		if c.Validation.HasErrors() {
			data["errors"] = c.Validation.Errors
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
			return c.RenderJSON(data)
		}

		av, errCompany := dynamodbattribute.MarshalMap(companyData)
		if errCompany != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			data["error"] = errCompany.Error()
			return c.RenderJSON(data)
		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(app.TABLE_NAME),
		}

		_, companyErr := app.SVC.PutItem(input)
		if companyErr != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			data["error"] = companyErr.Error()
			return c.RenderJSON(data)
		}
	} else {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":cn": {
					S: aws.String(companyName),
				},
				":cd": {
					S: aws.String(companyDescription),
				},
				":cl": {
					S: aws.String(displayPhoto),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
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
			UpdateExpression: aws.String("SET CompanyName = :cn, CompanyDescription = :cd, DisplayPhoto = :cl, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
	}

	// generate log
	var logs []*models.Logs
	// message: CompanyX has beed updated
	// todo: display changes?
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
		},
	})
	_, err := CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
// TODO no design yet if company's status and name can be changed
Updating of Company Info - name and Description
Params:
companyID

Form Body:
company_status -
company_name
****************
*/

// @Summary Update Company Info
// @Description This endpoint allows a user to update the company details, returning a 200 OK upon success. If the company user does not have permission, a 401 Unauthorized status is returned.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Param body body models.UpdateCompanyInfoRequest true "Update company info body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID [put]
func (c CompanyController) UpdateCompanyInfo(companyID string) revel.Result {

	data := make(map[string]interface{})

	// companyStatus := c.Params.Form.Get("company_status")
	companyName := utils.TrimSpaces(c.Params.Form.Get("company_name"))
	companyDescription := utils.TrimSpaces(c.Params.Form.Get("company_description"))
	//image := c.Params.Files["display_photo"]
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION
	checked := ops.CheckPermissions(constants.EDIT_COMPANY, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(data)
	}

	// get company
	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	displayPhoto := company.DisplayPhoto

	// check if company name is unique
	// isCompanyUnique := IsCompanyNameUnique(companyName)
	// if !isCompanyUnique && company.CompanyName != companyName {
	// 	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_477)
	// 	return c.RenderJSON(data)
	// }

	if companyName != company.CompanyName {
		// delete company data
		PK := constants.PREFIX_COMPANY + companyID
		SK := company.SK

		_, deleteErr := ops.DeleteByPartitionKey(PK, SK)
		if deleteErr != nil {
			data["error"] = "Got error calling DeleteItem"
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
		//structure copy of company data with updated SK
		companyData := models.Company{
			PK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
			SK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyName),
			CompanyID:          companyID,
			CompanyName:        companyName,
			CompanyDescription: companyDescription,
			DisplayPhoto:       displayPhoto,
			Status:             company.Status,
			CreatedAt:          company.CreatedAt,
			Type:               company.Type,
			SetupWizardStatus:  constants.WIZARD_STATUS_DONE, //Reason: after updating the company information in wizard. it automatically redirects to dashboard -> might cause errors in other pages.
			SearchKey:          strings.ToLower(companyName),
			UpdatedAt:          utils.GetCurrentTimestamp(),
		}

		companyData.Validate(c.Validation, constants.SERVICE_TYPE_ADD_COMPANY)
		if c.Validation.HasErrors() {
			data["errors"] = c.Validation.Errors
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
			return c.RenderJSON(data)
		}

		av, errCompany := dynamodbattribute.MarshalMap(companyData)
		if errCompany != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			data["error"] = errCompany.Error()
			return c.RenderJSON(data)
		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(app.TABLE_NAME),
		}

		_, companyErr := app.SVC.PutItem(input)
		if companyErr != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			data["error"] = companyErr.Error()
			return c.RenderJSON(data)
		}
	} else {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":cn": {
					S: aws.String(companyName),
				},
				":cd": {
					S: aws.String(companyDescription),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
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
			UpdateExpression: aws.String("SET CompanyName = :cn, CompanyDescription = :cd, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(data)
		}
	}

	// generate log
	var logs []*models.Logs
	// message: CompanyX has beed updated
	// todo: display changes?
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
		},
	})
	_, err := CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
// TODO no design yet if company's status and name can be changed
Updating of Company Logo Only
Params:
companyID

Form Body:
company_status -
company_name
****************
*/

// @Summary Update Company Logo
// @Description This endpoint allows a user to update the company logo, returning a 200 OK upon success. If the company user does not have permission, a 401 Unauthorized status is returned.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Param body body models.UpdateCompanyLogoRequest true "Update company logo body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/logo/:companyID [patch]
func (c CompanyController) UpdateCompanyLogo(companyID string) revel.Result {

	data := make(map[string]interface{})

	// companyStatus := c.Params.Form.Get("company_status")
	image := c.Params.Files["display_photo"]
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION
	checked := ops.CheckPermissions(constants.EDIT_COMPANY, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(data)
	}

	// get company
	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	displayPhoto := ""
	if image != nil {
		// if image file is not empty
		dp, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_COMPANY)
		if err != nil {
			data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_476)
			return c.RenderJSON(data)
		}
		displayPhoto = dp
	}

	// check if company name is unique

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cl": {
				S: aws.String(displayPhoto),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
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
		UpdateExpression: aws.String("SET DisplayPhoto = :cl, UpdatedAt = :ua"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs []*models.Logs
	// message: CompanyX has beed updated
	// todo: display changes?
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
Update Active Company
Changes a user's email
Body:
email
****************
*/

// @Summary Update User Active Company
// @Description This endpoint allows a user to change active company, returning a 200 OK upon success. If the company is not found, a 404 Not Found status is returned.
// @Tags companies
// @Produce json
// @Param userID path string true "User ID"
// @Param body body models.UpdateUserActiveCompanyRequest true "Update user active company body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/active/:userID [patch]
func (c CompanyController) UpdateUserActiveCompany(userID string) revel.Result {
	result := make(map[string]interface{})

	companyID := c.Params.Form.Get("company_id")

	// check if company exists
	company, opsError := ops.GetCompanyByID(companyID)
	if opsError != nil || company.PK == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	// check if user id exists
	user, err := ops.GetUserByID(userID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
		return c.RenderJSON(result)
	}

	// update
	cac := ops.ChangeActiveCompanyarams{UserID: user.UserID, CompanyID: companyID, Email: user.Email}
	opsError = ops.ChangeActiveCompany(cac, c.Controller)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	user.ActiveCompany = companyID

	result["token"] = ops.EncodeToken(user)
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
Get Active Company
Changes a user's email
Body:
email
****************
*/

// @Summary Get Active Company
// @Description This endpoint retrieves the active company of the user, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/active/:userID [get]
func (c CompanyController) GetActiveCompany(userID string) revel.Result {

	//Create data interface
	data := make(map[string]interface{})

	// check if user id exists
	user, err := ops.GetUserByID(userID)
	if err != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
		return c.RenderJSON(data)
	}

	company, err1 := ops.GetActiveCompany(userID)
	if err1 != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
		return c.RenderJSON(data)
	}

	//

	activeCompany, err2 := ops.GetCompanyByID(company.ActiveCompany)
	if err2 != nil {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_462)
		return c.RenderJSON(data)
	}

	if activeCompany.DisplayPhoto != "" {
		resizedPhoto := utils.ChangeCDNImageSize(activeCompany.DisplayPhoto, constants.IMAGE_SUFFIX_100)
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err != nil {
			data["error"] = err.Error()
			return c.RenderJSON(data)
		}

		activeCompany.DisplayPhoto = fileName
	}

	user.ActiveCompany = company.ActiveCompany

	data["company"] = activeCompany
	data["token"] = ops.EncodeToken(user)

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
IsCompanyNameUnique()
- Returns true or false if a company name already taken / exists in user's companies
****************
*/
func IsCompanyNameUnique(companyName string, userID string) bool {
	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_COMPANY),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
				},
			},
			"CompanyName": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyName)),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return false
	}

	if len(result.Items) != 0 {
		return false
	}
	return true
}

/*
****************
AddBookmarkCompany
URL: PUT /v1/groups/:companyID
Params:
- companyID <string>
- userID <string>
****************
*/
func (c GroupController) AddBookmarkCompany() revel.Result {
	data := make(map[string]interface{})

	// params
	companyID := c.Params.Form.Get("company_id")
	userID := c.ViewArgs["userID"].(string)

	// check if userID exists
	u, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	//prepare list
	CompanyID := &dynamodb.AttributeValue{
		S: aws.String(companyID),
	}

	var listOfCompaniess []*dynamodb.AttributeValue
	listOfCompaniess = append(listOfCompaniess, CompanyID)

	//preparing update query
	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(u.PK),
			},
			"SK": {
				S: aws.String(u.SK),
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":bm": {
				L: listOfCompaniess,
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
			":empty_list": {
				L: []*dynamodb.AttributeValue{},
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET BookmarkCompanies = list_append(if_not_exists(BookmarkCompanies, :empty_list), :bm), UpdatedAt = :ua"),
	}

	//inserting query
	_, err := app.SVC.UpdateItem(input)

	//return 500 if invalid
	if err != nil {
		data["message"] = "Error at inputting"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs []*models.Logs
	// message: GroupX has been bookmarked by UserX
	logs = append(logs, &models.Logs{
		CompanyID: u.ActiveCompany,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_BOOKMARK_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

/*
****************
DeleteBookmarkCompany
URL: PUT /v1/groups/:companyID
Params:
- companyID <string>
****************
*/
func (c GroupController) DeleteBookmarkCompany() revel.Result {
	data := make(map[string]interface{})

	// params
	companyID := c.Params.Form.Get("company_id")
	userID := c.ViewArgs["userID"].(string)

	// check if userID exists
	u, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	//finding index of companyID
	var index string

	for idx, item := range u.BookmarkCompanies {
		if item == companyID {
			//converting type int to string
			index = strconv.Itoa(idx)
		}
	}

	if index == "" {
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_474)
		return c.RenderJSON(data)
	}
	//preparing update query
	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(u.PK),
			},
			"SK": {
				S: aws.String(u.SK),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("REMOVE BookmarkCompanies[" + index + "]"),
	}

	//executing query
	_, err := app.SVC.UpdateItem(input)

	//return 500 if errors
	if err != nil {
		data["message"] = "Error at inputting"
		data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(data)
	}

	// generate log
	var logs []*models.Logs
	// message: GroupX has been un-bookmarked by UserX
	logs = append(logs, &models.Logs{
		CompanyID: u.ActiveCompany,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_DELETE_BOOKMARK_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		data["message"] = "error while creating logs"
	}

	// data["exp"] = expressionString
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

// @Summary Invite Company Admin
// @Description This endpoint sends an invitation for a user to be an admin of the company, returning a 200 OK status upon success. If required fields are missing, the server will respond with a 400 Bad Request; if role name is not unique, the server will respond with a 400 Bad Request.
// @Tags companies
// @Produce json
// @Param body body models.InviteCompanyAdminRequest true "Invite company admin body"
// @Param key query string false "Search Key"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param limit query int false "Limit the number of results"
// @Param status query string false "Filter by company status"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/invite [post]
func (c CompanyController) InviteCompanyAdmin() revel.Result {
	userID := c.ViewArgs["userID"].(string)
	result := make(map[string]interface{})

	// form body
	companyID := c.Params.Form.Get("company_id")
	users := c.Params.Form.Get("users")

	searchKey := c.Params.Query.Get("key")
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	limit := c.Params.Query.Get("limit")
	status := c.Params.Query.Get("status")

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

	// check if user id token is belong to the company
	// add permisions

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

	addUsersResult, err := InviteCompanyAdmin(unmarshalUsers, companyID, searchKey, paramLastEvaluatedKey, limit, status)
	if err != nil {
		if addUsersResult != nil {
			result["errors"] = addUsersResult
		}
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	t, _ := json.Marshal(addUsersResult)
	json.Unmarshal([]byte(t), &unmarshalUsers)

	//DO SEND EMAILS
	for _, userz := range unmarshalUsers {
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

	//Get user
	gcm := ops.GetCompanyMemberParams{UserID: userID, CompanyID: companyID}
	usr, opsError := ops.GetCompanyMember(gcm, c.Controller)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		result["status"] = err.Error()
		return c.RenderJSON(opsError)
	}

	var handler = usr.Handler

	if usr.Handler == "" {
		handler = c.ViewArgs["userID"].(string)
	}

	err = AddUsersToCompany(companyID, addUsersResult.([]models.User), c.ViewArgs["userID"].(string), handler, true)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

func InviteCompanyAdmin(users []models.User, activeCompany, searchKey, paramLastEvaluatedKey, limit, status string) (interface{}, error) {
	var emailExists []string
	isCompanyAdmin := true
	// check duplicate emails
	duplicates := CheckDuplicateEmails(users)
	if len(duplicates) != 0 {
		e := errors.New(constants.HTTP_STATUS_400)
		var result = struct {
			Duplicates []string
		}{duplicates}
		return result, e
	}

	// loop on users, use batch
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest
	for i, user := range users {

		// check if email is unique
		err := IsEmailUniqueInCompany(user.Email, activeCompany, searchKey, paramLastEvaluatedKey, limit, isCompanyAdmin, []string{})
		if err != nil {
			if err.Error() == constants.HTTP_STATUS_473 {
				emailExists = append(emailExists, user.Email)
			} else {
				e := errors.New(err.Error())
				return nil, e
			}
		}

		// set user uuid
		userID := utils.GenerateTimestampWithUID()

		// set search key
		searchKey := user.FirstName + " " + user.LastName + " " + user.Email
		searchKey = strings.ToLower(searchKey)

		// push to batch
		role := []string{constants.USER_ROLE_USER}
		roles, _ := dynamodbattribute.MarshalList(role)

		// set status
		var status string
		if user.Status == "" {
			status = constants.ITEM_STATUS_PENDING
		} else {
			status = user.Status
		}

		// generate token for activation
		userToken := utils.GenerateRandomString(8)

		// push to batch
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
		if i%constants.BATCH_LIMIT == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
		users[i].UserID = userID
		users[i].UserToken = userToken

	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		batches = append(batches, currentBatch)
	}

	if len(emailExists) != 0 {
		e := errors.New(constants.HTTP_STATUS_473)
		return emailExists, e
	}

	_, err := ops.BatchWriteItemHandler(batches)
	if err != nil {
		e := errors.New(err.Error())
		return nil, e
	}

	// send activation link
	var recipients []mail.Recipient
	frontendUrl, _ := revel.Config.String("url.frontend")

	for _, user := range users {
		token, _ := utils.EncodeToJwtToken(jwt.MapClaims{
			"userToken": user.UserToken,
			"userID":    user.UserID,
			"companyID": activeCompany,
			// "nbf":       time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			// "exp": time.Now().Add(time.Hour * 2).Unix(), // expires after 2 hours
		})
		recipients = append(recipients, mail.Recipient{
			Name:            user.FirstName + " " + user.LastName,
			Email:           user.Email,
			InviteAdminLink: frontendUrl + "/verify-email?token=" + token,
		})
	}

	// year, _, _ := time.Now().Date()

	jobs.Now(mail.SendEmail{
		Subject:    "Company Admin Invite",
		Recipients: recipients,
		Template:   "invite_company_admin.html",
		// Year:       year,
	})

	return users, nil
}

// GetCompanyUser()
// returns the company user entity
func GetCompanyUser(companyID, userID string) (models.CompanyUser, error) {
	var companyUser models.CompanyUser
	input := &dynamodb.GetItemInput{
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(constants.PREFIX_COMPANY + companyID),
			},
			"SK": {
				S: aws.String(constants.PREFIX_USER + userID),
			},
		},
	}
	queryResult, err := app.SVC.GetItem(input)
	if err != nil {
		return companyUser, err
	}

	if queryResult.Item == nil {
		e := errors.New("Company user not found")
		return companyUser, e
	}
	if len(queryResult.Item) != 0 {
		err = dynamodbattribute.UnmarshalMap(queryResult.Item, &companyUser)
		if err != nil {
			return companyUser, err
		}
	}

	return companyUser, nil
}

// func (c CompanyController) GetCompanyUserController(userID string) revel.Result {
// 	result := make(map[string]interface{})
// 	companyID := c.ViewArgs["companyID"].(string)

// 	return nil
// }

// ChangeActiveCompany()
// func ChangeActiveCompany(companyID, userID, email string) error {
// 	// update
// 	input := &dynamodb.UpdateItemInput{
// 		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
// 			":ac": {
// 				S: aws.String(companyID),
// 			},
// 			":ua": {
// 				S: aws.String(utils.GetCurrentTimestamp()),
// 			},
// 		},
// 		TableName: aws.String(app.TABLE_NAME),
// 		Key: map[string]*dynamodb.AttributeValue{
// 			"PK": {
// 				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
// 			},
// 			"SK": {
// 				S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, email)),
// 			},
// 		},
// 		UpdateExpression: aws.String("SET ActiveCompany = :ac, UpdatedAt = :ua"),
// 	}

// 	_, err := app.SVC.UpdateItem(input)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// GetUserActiveCompanies
func GetUserActiveCompanies(userID string) ([]models.UserCompany, error) {
	userCompanies := []models.UserCompany{}
	companies := []models.UserCompany{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_COMPANY),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
					{
						S: aws.String(constants.ITEM_STATUS_PENDING),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		return companies, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &userCompanies)
	if err != nil {
		return companies, err
	}

	for _, userCompany := range userCompanies {
		c, opsError := ops.GetCompanyByID(userCompany.CompanyID)
		if err != nil {
			return companies, errors.New(opsError.Status.Code)
		}
		company := models.UserCompany{
			PK:                 c.PK,
			SK:                 c.SK,
			CompanyID:          c.CompanyID,
			CompanyName:        c.CompanyName,
			CompanyDescription: c.CompanyDescription,
			CreatedAt:          c.CreatedAt,
		}
		if c.DisplayPhoto != "" {
			resizedPhoto := utils.ChangeCDNImageSize(c.DisplayPhoto, "_100")
			fileName, err := cdn.GetImageFromStorage(resizedPhoto)
			if err == nil {
				company.DisplayPhoto = fileName
			}
		}
		companies = append(companies, company)
	}

	return companies, nil
}

// @Summary Create Company with Department and Subscription
// @Description This endpoint allows the creation of a company with its respective department and subscription, returning a 200 OK status upon success. If required fields are missing, the server will respond with a 422 Unprocessable Content.
// @Tags companies
// @Produce json
// @Param body body models.CreateCompanyWithDepAndSubscriptionRequest true "Create company with department and subscription body"
// @Param department_name query string false "Department Name"
// @Param department_desc query string false "Department Description"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/create [post]
func (c CompanyController) CreateCompanyWithDepAndSubscription() revel.Result {

	result := make(map[string]interface{})
	currentUser := c.ViewArgs["userID"].(string)

	companyPhoto := c.Params.Files["company_photo"]
	companyName := utils.TrimSpaces(c.Params.Form.Get("company_name"))
	companyDesc := c.Params.Form.Get("company_desc")

	departmentName := c.Params.Get("department_name")
	departmentDesc := c.Params.Get("department_desc")

	cardId := c.Params.Form.Get("card_id")
	priceID := c.Params.Form.Get("price_id")
	quantityGet := c.Params.Form.Get("quantity")

	if utils.FindEmptyStringElement([]string{companyName, priceID, quantityGet}) {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Missing parameters",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	quantity, err := strconv.ParseInt(quantityGet, 10, 64)
	if err != nil {
		c.Response.Status = 422
		return c.RenderJSON(map[string]interface{}{
			"message": "Invalid quantity value",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	/*****************
	CHECK CARD INFORMATION
	*****************/
	companies, err := ops.GetUserSubscribedCompanies(currentUser)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(map[string]interface{}{
			"message": "Can't get user subscribed companies",
			"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	userInfo, stripeError := ops.GetUserCustomerID(currentUser)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	companyCount := len(companies)

	if companyCount > 0 {
		if quantity > 5 {
			if utils.FindEmptyStringElement([]string{cardId}) {
				c.Response.Status = 422
				return c.RenderJSON(map[string]interface{}{
					"message": "Missing card_id",
					"status":  utils.GetHTTPStatus(constants.HTTP_STATUS_422),
				})
			}

			cardExisting, stripeError := stripeoperations.CheckCustomerPaymentMethod(*userInfo.CustomerID, cardId)
			if stripeError != nil {
				c.Response.Status = stripeError.HTTPStatusCode
				return c.RenderJSON(stripeError)
			}

			if *cardExisting {
				cus, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardId)
				if err != nil {
					c.Response.Status = stripeError.HTTPStatusCode
					return c.RenderJSON(stripeError)
				}
				_ = cus
			} else {
				pm, stripeError := stripeoperations.AttachStripeCardToCustomer(*userInfo.CustomerID, cardId)
				if stripeError != nil {
					c.Response.Status = stripeError.HTTPStatusCode
					return c.RenderJSON(stripeError)
				}
				_ = pm
				updatedCustomer, stripeError := stripeoperations.UpdateStripeCustomerDefaultCard(*userInfo.CustomerID, cardId)
				if stripeError != nil {
					c.Response.Status = stripeError.HTTPStatusCode
					return c.RenderJSON(stripeError)
				}
				_ = updatedCustomer
			}
		}
	}

	/*****************
	CREATE SUBSCRIPTION
	*****************/
	company, cusErr := c.CreateCompanyHelper(companyName, companyDesc, companyPhoto)
	if cusErr != nil {
		c.Response.Status = cusErr.HTTPStatusCode
		return c.RenderJSON(cusErr)
	}
	result["company"] = company

	/*****************
	CREATE SUBSCRIPTION
	*****************/

	if departmentName != "" {
		createDept := ops.CreateDepartmentPayload{
			CompanyID:   company.CompanyID,
			Name:        departmentName,
			Description: departmentDesc,
		}
		department, deptErr := ops.CreateDepartment(createDept, c.Validation)
		if deptErr != nil {
			c.Response.Status = deptErr.HTTPStatusCode
			return c.RenderJSON(deptErr)
		}
		result["department"] = department

		// department, cusErr := c.CreateDepartmentHelper(company.CompanyID, departmentName, departmentDesc)
		// if cusErr != nil {
		// 	c.Response.Status = cusErr.HTTPStatusCode
		// 	return c.RenderJSON(cusErr)
		// }

		// result["department"] = department
	}

	/*****************
	SWITCH TO ACTIVE COMPANY
	*****************/
	cac := ops.ChangeActiveCompanyarams{UserID: userInfo.UserID, CompanyID: company.CompanyID, Email: userInfo.Email}
	opsError := ops.ChangeActiveCompany(cac, c.Controller)
	if opsError != nil {
		c.Response.Status = opsError.HTTPStatusCode
		return c.RenderJSON(opsError)
	}

	/*****************
	CREATE SUBSCRIPTION
	*****************/
	_, stripeError = stripeoperations.CreateStripeSubscriptionHelper(*userInfo.CustomerID, priceID, quantity, companyCount)
	if stripeError != nil {
		c.Response.Status = stripeError.HTTPStatusCode
		return c.RenderJSON(stripeError)
	}

	userToToken := models.User{
		UserID:        userInfo.UserID,
		Email:         userInfo.Email,
		ActiveCompany: company.CompanyID,
	}

	//
	//
	//
	//
	//
	//
	//

	c.Response.Status = 200
	result["token"] = ops.EncodeToken(userToToken)
	return c.RenderJSON(result)
}

func (c CompanyController) CreateCompanyHelper(companyName, companyDescription string, image []*multipart.FileHeader) (*models.Company, *models.ErrorResponse) {
	currentUser := c.ViewArgs["userID"].(string)
	companyID := utils.GenerateTimestampWithUID()
	currentTime := utils.GetCurrentTimestamp()

	displayPhoto := ""
	if image != nil {
		companyPhoto, err := cdn.UploadImageToStorage(image, constants.IMAGE_TYPE_COMPANY)
		if err != nil {
			return nil, &models.ErrorResponse{
				HTTPStatusCode: 476,
				Message:        "Error on UploadImageToStorage",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_476),
			}
		}
		displayPhoto = companyPhoto
	}

	company := models.Company{
		PK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:                 utils.AppendPrefix(constants.PREFIX_COMPANY, companyName),
		CompanyID:          companyID,
		CompanyName:        companyName,
		CompanyDescription: companyDescription,
		DisplayPhoto:       displayPhoto,
		Status:             constants.ITEM_STATUS_ACTIVE,
		SetupWizardStatus:  constants.WIZARD_STATUS_DONE,
		CreatedAt:          currentTime,
		Type:               constants.ENTITY_TYPE_COMPANY,
		SearchKey:          strings.ToLower(companyName),
	}

	company.Validate(c.Validation, constants.SERVICE_TYPE_ADD_COMPANY)
	if c.Validation.HasErrors() {
		//
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "Error on company.Validate",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		}
	}

	user, opsError := ops.GetUserByID(currentUser)
	if opsError != nil {
		return nil, opsError
	}

	// Checks if no user found
	if user.PK == "" {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 461,
			Message:        "No user found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_461),
		}
	}

	// Check if Company Name is already taken / exists in user's companies
	isCompanyUnique := IsCompanyNameUnique(companyName, currentUser)
	if !isCompanyUnique {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 477,
			Message:        "The company name already exists.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_477),
		}
	}

	av, errCompany := dynamodbattribute.MarshalMap(company)
	if errCompany != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        errCompany.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, companyErr := app.SVC.PutItem(input)
	if companyErr != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        companyErr.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		}
	}

	/*****************
	ADD COMPANY USER
	*****************/
	searchKey := user.FirstName + " " + user.LastName + " " + user.Email
	searchKey = strings.ToLower(searchKey)

	companyUser := models.CompanyUser{
		PK:            utils.AppendPrefix(constants.PREFIX_COMPANY, companyID),
		SK:            utils.AppendPrefix(constants.PREFIX_USER, currentUser),
		GSI_SK:        utils.AppendPrefix(constants.PREFIX_USER, searchKey),
		UserID:        currentUser,
		CompanyID:     companyID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		JobTitle:      user.JobTitle,
		Email:         user.Email,
		ContactNumber: user.ContactNumber,
		DisplayPhoto:  user.DisplayPhoto,
		SearchKey:     searchKey,
		UserType:      constants.USER_TYPE_COMPANY_OWNER,
		Handler:       user.UserID,
		Status:        constants.ITEM_STATUS_ACTIVE,
		CreatedAt:     utils.GetCurrentTimestamp(),
		Type:          constants.ENTITY_TYPE_COMPANY_MEMBER,
	}

	av, err := dynamodbattribute.MarshalMap(companyUser)
	if err != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}
	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		}
	}

	/*****************
	ASSIGN USER ROLE
	*****************/
	item := models.UserRole{
		PK: utils.AppendPrefix(constants.PREFIX_USER, user.UserID),
		SK: utils.AppendPrefix(utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"), utils.AppendPrefix(constants.PREFIX_COMPANY, companyUser.CompanyID)),
		// SK:        utils.AppendPrefix(constants.PREFIX_ROLE, "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"),
		UserID:    user.UserID,
		RoleID:    "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e",
		CompanyID: companyUser.CompanyID,
		Type:      constants.ENTITY_TYPE_USER_ROLE,
	}

	av, err = dynamodbattribute.MarshalMap(item)
	if err != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		}
	}

	input = &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		return nil, &models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		}
	}

	/*****************
	GENERATE USER LOGS
	*****************/
	var logs []*models.Logs
	// message: John Smith created CompanyX
	logs = append(logs, &models.Logs{
		CompanyID: companyID,
		UserID:    currentUser,
		LogAction: constants.LOG_ACTION_ADD_COMPANY,
		LogType:   constants.ENTITY_TYPE_COMPANY,
		LogInfo: &models.LogInformation{
			Company: &models.LogModuleParams{
				ID: companyID,
			},
			User: &models.LogModuleParams{
				ID: currentUser,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		//
		//
		// )
		//
		//
	}

	return &company, nil
}

type GetCompanyUsersCountOutput struct {
	// Active  int `json:"active"`
	Deleted int `json:"deleted"`
	Active  int `json:"active"`
}

// GetCompanyUsersCount

// @Summary Get Company Users Count
// @Description This endpoint retrieves the amount of users under a company, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID/users/count [get]
func (c CompanyController) GetCompanyUsersCount(companyID string) revel.Result {
	// activeCompanyUsers, err := GetCompanyUsers(companyID, []string{})
	// if err != nil {
	// 	c.Response.Status = 400
	// 	return c.RenderJSON(err.Error()) // todo
	// }
	// deletedCompanyUsers, err := GetCompanyUsers(companyID, []string{constants.ITEM_STATUS_DELETED})
	// if err != nil {
	// 	c.Response.Status = 400
	// 	return c.RenderJSON(err.Error()) // todo
	// }

	if companyID == "" {
		companyID = c.ViewArgs["companyID"].(string)
	}

	deletedCompanyUsers, err := GetCompanyUsersNew(GetCompanyUsersInput{
		CompanyID: companyID,
		Status:    []string{constants.ITEM_STATUS_DELETED},
	})

	activeCompanyUsers, err := GetCompanyUsersNew(GetCompanyUsersInput{
		CompanyID: companyID,
		Status:    []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_DEFAULT, constants.ITEM_STATUS_PENDING},
	})

	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error()) // todo
	}

	output := GetCompanyUsersCountOutput{
		// Active:  len(activeCompanyUsers),
		Deleted: len(deletedCompanyUsers),
		Active:  len(activeCompanyUsers),
	}

	return c.RenderJSON(output)
}

// GetCompanyUsers()
func GetCompanyUsers(companyID string, status []string) ([]models.CompanyUser, error) {
	var companyUsers []models.CompanyUser

	if len(status) == 0 {
		status = []string{constants.ITEM_STATUS_DEFAULT, constants.ITEM_STATUS_PENDING, constants.ITEM_STATUS_ACTIVE}
	}

	s, err := dynamodbattribute.MarshalList(status)
	if err != nil {
		return companyUsers, err
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
		ScanIndexForward: aws.Bool(false),
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: s,
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return companyUsers, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &companyUsers)
	if err != nil {
		return companyUsers, err
	}

	return companyUsers, nil
}

func (c CompanyController) GetCompanyCount(userID string) revel.Result {
	companyList, err := GetCompanyList(userID)

	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}

	output := len(companyList)

	return c.RenderJSON(output)
}

func GetCompanyList(userID string) ([]models.UserCompany, error) {
	var companies []models.UserCompany

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_COMPANY),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_NOT_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_DELETED),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return companies, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &companies)
	if err != nil {
		return companies, err
	}

	return companies, nil
}

// @Summary Get Company Users
// @Description This endpoint retrieves a list of users under a company, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID/users [get]
func (c CompanyController) GetCompanyUsers(companyID string) revel.Result {
	users := []models.CompanyUser{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
		KeyConditions: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
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
		//CHANGES TO ACCOMODATE PENDING AND DEFAULT USERS WITH CONNECTED INTEGRATION
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_DELETED),
					},
					{
						S: aws.String(constants.ITEM_STATUS_PENDING),
					},
					{
						S: aws.String(constants.ITEM_STATUS_DEFAULT),
					},
				},
			},
		},
	}

	res, err := ops.HandleQuery(params)
	if err != nil {
		// return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &users)
	if err != nil {
		// return nil, err
	}

	return c.RenderJSON(users)
}

// @Summary Get Company Groups
// @Description This endpoint retrieves a list of groups under a company, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID/groups [get]
func (c CompanyController) GetCompanyGroups(companyID string) revel.Result {
	users := []models.Group{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
		KeyConditions: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"GSI_SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
				},
			},
		},
	}

	res, err := ops.HandleQuery(params)
	if err != nil {
		// return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &users)
	if err != nil {
		// return nil, err
	}

	return c.RenderJSON(users)
}

// @Summary Enable User Access
// @Description This endpoint allows an admin to enable access for a user, returning a 200 OK upon success.
// @Tags companies
// @Produce json
// @Param companyID path string true "Company ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /companies/:companyID/settings/enable-user-access [post]
func (c CompanyController) EnableUserAccess(companyID string) revel.Result {
	company, err := ops.GetCompanyByID(companyID)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "Company not found.",
		})
	}

	_ = company

	users, getErr := GetCompanyUsersToBeEnabled(companyID)
	if getErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "Can't get users to be migrated.",
		})
	}
	for _, user := range users {
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
					S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, user.CompanyID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
				},
			},
			UpdateExpression: aws.String("SET #s = :s, UpdatedAt = :ua"),
		}
		_, err := app.SVC.UpdateItem(updateCompanyUserInput)
		if err != nil {
			continue
		}
	}

	return c.RenderJSON(nil)
}

func GetCompanyUsersToBeEnabled(companyID string) ([]models.UserRole, error) {
	users := []models.UserRole{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		// IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
		KeyConditions: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_USER),
					},
				},
			},
		},

		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),

		QueryFilter: map[string]*dynamodb.Condition{
			"Type": &dynamodb.Condition{
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_USER_ROLE),
					},
				},
			},
		},
	}

	res, err := ops.HandleQuery(params)
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}
