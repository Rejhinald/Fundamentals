package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"grooper/app"
	"grooper/app/api"
	"grooper/app/cdn"
	"grooper/app/constants"
	"grooper/app/constraints"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	googleoperations "grooper/app/integrations/google"
	googleapplicationoperations "grooper/app/integrations/google/applications"
	jiraoperations "grooper/app/integrations/jira"
	officeOperations "grooper/app/integrations/microsoft"
	"grooper/app/mail"
	"grooper/app/models"
	jiraModel "grooper/app/models/jira"
	officeModels "grooper/app/models/microsoft"
	ops "grooper/app/operations"
	"grooper/app/utils"

	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	"github.com/revel/revel/cache"
	uuid "github.com/satori/go.uuid"

	// googlefunctions "grooper/app/functions/google"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	// uuid "github.com/satori/go.uuid"
)

// GroupController Struct
type GroupController struct {
	*revel.Controller
	departmentOps ops.DepartmentOperations
}

// BranchMembers Struct
type BranchMembers struct {
	Copy []models.GroupMember `json:"copy,omitempty"`
	Move []models.GroupMember `json:"move,omitempty"`
}

/*
GetGroupsByDepartment - GET - v1/groups/
Method: GET
Params:
departmentID - for fetching groups for a particular department
key - for searching groups
include - for including group members, integrations (MEMBERS, INTEGRATIONS)
lastEvaluatedKey - for pagination
limit - to limit number of items returned
*/

// @Summary Get Groups By Department
// @Description This endpoint retrieves groups associated with a specific department, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param departmentID path string true "Department ID"
// @Param key query string false "Search Key"
// @Param include query string false "Include related data: members or integrations"
// @Param limit query int false "Limit the number of results"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups [get]
func (c GroupController) GetGroupsByDepartment() revel.Result {

	//Parameters
	departmentID := c.Params.Query.Get("department_id")
	searchKey := c.Params.Query.Get("key")
	include := c.Params.Query.Get("include")
	limit := c.Params.Query.Get("limit")
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	companyID := c.ViewArgs["companyID"].(string)
	//Make a data interface to return as JSON
	result := make(map[string]interface{})

	if departmentID == "" {
		result["error"] = "Missing required parameters for primary keys."
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

	//Models
	// groups := []models.Group{}
	lastEvaluatedKey := models.Group{}

	var err error

	groups, lastEvaluatedKey, err := GetGroups(departmentID, pageLimit, paramLastEvaluatedKey, searchKey)

	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		result["error"] = err.Error()
		return c.RenderJSON(result)
	}

	if len(include) != 0 {
		for i, group := range groups {
			if strings.Contains(include, "members") {
				memberUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_USER, c.Controller)
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				ownerUsers, err := GetGroupOwners(companyID, group.GroupID, constants.MEMBER_TYPE_OWNER, c.Controller)
				if err != nil {
					c.Response.Status = 400
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				groups[i].GroupMembers = memberUsers
				groups[i].GroupOwners = ownerUsers

			}
			if strings.Contains(include, "integrations") {
				integrationsResult, err := GetGroupIntegrations(group.GroupID)
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				groups[i].GroupIntegrations = integrationsResult
			}
		}
	}

	//TODO use the token to fetch the user's role and add it to the return value

	result["lastEvaluatedKey"] = lastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["groups"] = groups

	return c.RenderJSON(result)
}

/*
GetByID - Fetch by GroupID
route: /v1/group
Method: GET
params:
status = required* (ACTIVE OR INACTIVE)
company_id = required*
--
key = optional*
include = optional*
department_id = optional*
*/
func (c GroupController) GetByID() revel.Result {
	//init
	groupID := c.Params.Query.Get("group_id")
	result := make(map[string]interface{})
	group := models.Group{}

	companyID := c.ViewArgs["companyID"].(string)

	//check if params are empty
	if groupID == "" {
		result["message"] = "MISSING PARAMETERS"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	//fetch groups
	fetch, err := ops.GetGroupByID(groupID)
	if err != nil {
		result["message"] = "Something went wrong with group"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(fetch)
	}

	//is no data in fetch
	if len(fetch.Items) == 0 {
		result["message"] = "No Data"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	//binding data to &group
	err = dynamodbattribute.UnmarshalMap(fetch.Items[0], &group)
	if err != nil {
		result["message"] = "Something went wrong with unmarshalmap"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	//SET NEW = FALSE
	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(group.PK),
			},
			"SK": {
				S: aws.String(group.SK),
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":bf": {
				S: aws.String(constants.BOOL_FALSE),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET NewGroup = :bf, UpdatedAt = :ua"),
	}

	//inserting query
	_, err = app.SVC.UpdateItem(input)
	//return 500 if invalid
	if err != nil {
		result["message"] = "Error at inputting"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	//get group user members
	memberUsers, err := GetGroupMembers(companyID, groupID, constants.MEMBER_TYPE_USER, c.Controller, groupID)
	if err != nil {
		result["message"] = "Something went wrong with group members"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	originalGroupMembers := memberUsers // exclude sub group members

	//get group user owners
	ownerUsers, err := GetGroupOwners(companyID, groupID, constants.MEMBER_TYPE_OWNER, c.Controller, groupID)
	if err != nil {
		c.Response.Status = 400
		result["message"] = "Something went wrong with group members"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// GET SUB GROUP MEMBER
	subGroups, err := GetGroupMembers(companyID, groupID, constants.MEMBER_TYPE_GROUP, c.Controller, groupID)
	if err != nil {
		result["message"] = "Something went wrong with group members"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	//GetGroupmembersCount
	indivudualMembersCount := len(memberUsers)
	subgroupMembersCount := len(subGroups)
	groupMembersTotalCount := indivudualMembersCount + subgroupMembersCount

	//CLONE SUBGROUPS
	newSubList := subGroups

	//initalizeCount
	subGroupCount := 0

	for {

		//INTERATE TO THE CURRENT SUBGROUP
		for _, data := range newSubList {

			//FETCH SUB GROUP OF CURRENT SUB GROUP
			subOfSubGroups, err := GetGroupMembers(companyID, data.MemberID, constants.MEMBER_TYPE_GROUP, c.Controller)
			if err != nil {
				result["message"] = "Something went wrong with group members"
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}

			//ITERATE TO SUBGROUP OF SUBGROUP
			for _, subData := range subOfSubGroups {

				flag := false
				for _, data1 := range newSubList {
					//CHECK IF SUBGROUP OF SUBGROUP IS EXISTING IN THE LIST
					if data1.MemberID == subData.MemberID || groupID == subData.MemberID {
						flag = true
						break
					}
				}

				//INSERT SUBGROUP OF SUBGROUP IF NOT EXIST
				if !flag {
					newSubList = append(newSubList, subData)
				}
			}
		}

		//STOP LOOP IF EQUAL
		if subGroupCount == len(newSubList) {
			break
		}

		//APPEND NEW COUNT
		subGroupCount = len(newSubList)
	}

	//LOOP TO SUBGROUPS TO GET THE GROUP ID
	for _, getSubGroupID := range newSubList {
		//FETCH THE MEMBERS
		subGroupMembers, err := GetGroupMembers(companyID, getSubGroupID.MemberID, constants.MEMBER_TYPE_USER, c.Controller, groupID)
		if err != nil {
			result["message"] = "Something went wrong with group members"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		//LOOP TO MEMBERS
		for _, subGroupMems := range subGroupMembers {
			//CHECK IF MEMBER IS EXISTING IN MAIN GROUP
			flag := false
			for _, checkMember := range memberUsers {
				if checkMember.MemberID == subGroupMems.MemberID {
					flag = true
					break
				}
			}

			if !flag {
				//APPEND TAG ( SUBGROUP )
				memberUsers = append(memberUsers, subGroupMems)
			}
		}
	}
	group.GroupMembers = memberUsers
	group.OriginalGroupMembers = originalGroupMembers
	group.GroupOwners = ownerUsers
	group.SubGroup = subGroups

	//fetching for additional information
	for i, item := range group.SubGroup {
		user, err := GetGroupMembers(companyID, item.MemberID, constants.MEMBER_TYPE_USER, c.Controller)
		if err != nil {
			result["message"] = "Something went wrong with group member information"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		//fetch Sub group members
		group.SubGroup[i].GroupMembers = user

		//fetch sub group integration
		integrationsResult, err := GetGroupIntegrations(item.MemberID)
		if err != nil {
			result["message"] = "Something went wrong with group integrations"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}

		group.SubGroup[i].GroupIntegrations = integrationsResult

		//fetching Subgroup addtional information
		for idx, member := range group.SubGroup[i].GroupMembers {
			//fecthing sub group user information
			// user, opsError := ops.GetUserByID(member.MemberID)
			user, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
				UserID:    member.MemberID,
				CompanyID: group.CompanyID,
			}, c.Controller)
			if opsError != nil {
				result["message"] = "Something went wrong with group member information"
				result["status"] = utils.GetHTTPStatus(opsError.Status.Code)
				return c.RenderJSON(result)
			}
			group.SubGroup[i].GroupMembers[idx].MemberInformation = user
		}
	}

	//fetch group integration
	integrationsResult, err := GetGroupIntegrations(groupID)
	if err != nil {
		result["message"] = "Something went wrong with group integrations"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	group.GroupIntegrations = integrationsResult

	//fetching group sub integration
	for i, groupInteg := range group.GroupIntegrations {
		subIntegrationsResult, err := GetGroupSubIntegration(groupID)
		if err != nil {
			result["message"] = "Something went wrong with group sub integrations"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		// group.GroupIntegrations[i].GroupSubIntegrations = subIntegrationsResult

		subInteg := []models.GroupSubIntegration{}
		for _, sub := range subIntegrationsResult {
			if sub.ParentIntegrationID == groupInteg.IntegrationID {
				subInteg = append(subInteg, sub)
			}
		}

		group.GroupIntegrations[i].GroupSubIntegrations = subInteg
	}

	//fetching GroupMembers information
	for i, member := range group.GroupMembers {

		//fetching user member information
		// user, opsError := ops.GetUserByID(member.MemberID)

		user, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    member.MemberID,
			CompanyID: group.CompanyID,
		}, c.Controller)
		if opsError != nil {
			result["message"] = "Something went wrong with group member information"
			result["status"] = utils.GetHTTPStatus(opsError.Status.Code)
			return c.RenderJSON(result)
		}

		//

		if user.DisplayPhoto != "" {
			resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")

			fileName, err := cdn.GetImageFromStorage(resizedPhoto)
			if err != nil {
				result["logs"] = "error from cdn"
				result["error"] = err.Error()
				return c.RenderJSON(result)
			}

			user.DisplayPhoto = fileName
		}

		for j, originalMember := range group.OriginalGroupMembers {
			if originalMember.MemberID == member.MemberID {
				group.OriginalGroupMembers[j].MemberInformation = user
			}
		}

		group.GroupMembers[i].MemberInformation = user

		/********commented because causing to show github UUID in AWS*************/
		// userUID, err := ops.GetIntegrationUID(member.MemberID)
		// if err != nil {
		// 	result["message"] = "Something went wrong with group member information"
		// 	result["status"] = utils.GetHTTPStatus(err.Error())
		// 	return c.RenderJSON(result)
		// }

		// group.GroupMembers[i].IntegrationUID = userUID.IntegrationUID

		//fetching sub group member information
		userGroups, err := GetUserCompanyGroups(companyID, member.MemberID)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			result["message"] = "Something went wrong with group member groups"
			return c.RenderJSON(result)
		}
		newGroupSlice := []models.Group{}
		for _, item := range userGroups {
			if item.PK != "" {
				newGroupSlice = append(newGroupSlice, item)
			}
		}
		group.GroupMembers[i].MemberInformation.Groups = newGroupSlice
	}
	tmpDepartments := map[string]models.Department{}

	qd := ops.GetDepartmentByIDPayload{
		DepartmentID: group.DepartmentID,
		CheckStatus:  true,
	}
	if val, ok := tmpDepartments[group.DepartmentID]; ok {
		group.Department = val
	} else {
		department, opsError := ops.GetDepartmentByID(qd, c.Validation)
		if opsError == nil {
			group.Department = *department
			tmpDepartments[group.DepartmentID] = *department
		}
	}
	//return
	result["groups"] = group
	result["groupMembersTotalCount"] = groupMembersTotalCount
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// Optimzed version of GetByID()

// @Summary Get Company Group By ID
// @Description This endpoint retrieves a company group by its ID, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/profile [get]
func (c GroupController) GetCompanyGroupByID() revel.Result {
	groupID := c.Params.Query.Get("group_id") // TODO: instead of passing in query params make it as url param /groups/:groupID
	result := make(map[string]interface{})
	// group := models.Group{}

	companyID := c.ViewArgs["companyID"].(string)

	//check if params are empty
	// TODO: remove this when group_id query param has been removed
	if groupID == "" {
		result["message"] = "MISSING PARAMETERS"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	group, resultErr := GetCompanyGroupNew(companyID, groupID)
	if resultErr != nil {
		c.Response.Status = resultErr.HTTPStatusCode
		return c.RenderJSON(resultErr)
	}

	integrations, err := GetGroupIntegrationsNew(groupID, companyID, false)
	if err == nil {
		group.GroupIntegrations = integrations
	}

	// TODO: optimize api calls when calling group sub integ
	for idx, integration := range group.GroupIntegrations {
		subIntegrationsResult, err := GetGroupSubIntegration(groupID)
		if err != nil {
			result["message"] = "Something went wrong with group sub integrations"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		group.GroupIntegrations[idx].GroupSubIntegrations = subIntegrationsResult

		subInteg := []models.GroupSubIntegration{}
		for _, sub := range subIntegrationsResult {
			if sub.ParentIntegrationID == integration.IntegrationID {
				subInteg = append(subInteg, sub)
			}
		}

		group.GroupIntegrations[idx].GroupSubIntegrations = subInteg
	}

	// TODO: run updating of group's NewGroup attribute in the background and transfer to reusable function
	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(group.PK),
			},
			"SK": {
				S: aws.String(group.SK),
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":bf": {
				S: aws.String(constants.BOOL_FALSE),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET NewGroup = :bf, UpdatedAt = :ua"),
	}

	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		result["message"] = err.Error()
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	groupMembersUsers, err := GetGroupMembersNew(companyID, group.GroupID, constants.MEMBER_TYPE_USER)
	if err != nil {
		// TODO: Handle error
	}

	for idx, memberUser := range groupMembersUsers {
		// user := getUserInTmp(memberUser.MemberID)
		// companyUser := getCompanyMemberInTmp(group.CompanyID, memberUser.MemberID, c.Controller)
		user, opsErr := ops.GetUserByID(memberUser.MemberID)
		_ = opsErr
		companyUser, opsErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    memberUser.MemberID,
			CompanyID: group.CompanyID,
		}, c.Controller)
		_ = opsErr
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
		if user.UserID != "" {
			groupMembersUsers[idx].MemberInformation = companyUser
			groupMembersUsers[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)

			userGroups, err := ops.GetUserConnectedGroups(companyID, user.UserID)
			if err != nil {
			}
			groupMembersUsers[idx].GroupsCount = len(userGroups)
		}
	}

	groupMembers, gacgerr := GetAllCompanyGroupMembers(companyID, group.GroupID, c.Controller)
	_ = gacgerr

	group.GroupMembers = groupMembers              // TODO: change attribute name
	group.OriginalGroupMembers = groupMembersUsers // TODO: change attribute name

	// return c.RenderJSON(tst)

	groupOwners, err := GetGroupMembersNew(companyID, group.GroupID, constants.MEMBER_TYPE_OWNER)
	if err != nil {
		// TODO: Handle error
	}

	for idx, groupOwner := range groupOwners {
		user := getUserInTmp(groupOwner.MemberID)
		companyUser := getCompanyMemberInTmp(group.CompanyID, groupOwner.MemberID, c.Controller)
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
		groupOwners[idx].MemberInformation = companyUser
		groupOwners[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)
	}
	group.GroupOwners = groupOwners

	groupMemberGroups, err := GetGroupMembersNew(companyID, group.GroupID, constants.MEMBER_TYPE_GROUP)
	if err != nil {
		// TODO: Handle error
	}
	for idx, memberGroup := range groupMemberGroups {
		g, gErr := GetCompanyGroupNew(companyID, memberGroup.MemberID)
		if gErr == nil {
			groupMemberGroups[idx].Name = g.GroupName
			groupMemberGroups[idx].Bg = g.GroupColor
			groupMemberGroups[idx].AssociatedAccounts = g.AssociatedAccounts
			mUsers, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_USER)
			if err != nil {
				// TODO: Handle error
			}

			mGroups, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_GROUP)
			if err != nil {
				// TODO: Handle error
			}
			groupMemberGroups[idx].MembersCount = len(mUsers) + len(mGroups)

			subGroupIntegrations, err := GetGroupIntegrationsNew(g.GroupID, companyID, false)
			if err == nil {
				groupMemberGroups[idx].GroupIntegrations = subGroupIntegrations
			}
		}

		for i, integration := range groupMemberGroups[idx].GroupIntegrations {
			subIntegrationsResult, err := GetGroupSubIntegration(g.GroupID)
			if err != nil {
				result["message"] = "Something went wrong with group sub integrations"
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}
			groupMemberGroups[idx].GroupIntegrations[i].GroupSubIntegrations = subIntegrationsResult

			subInteg := []models.GroupSubIntegration{}
			for _, sub := range subIntegrationsResult {
				if sub.ParentIntegrationID == integration.IntegrationID {
					subInteg = append(subInteg, sub)
				}
			}

			groupMemberGroups[idx].GroupIntegrations[i].GroupSubIntegrations = subInteg
		}
	}

	group.SubGroup = groupMemberGroups // TODO: change attribute name

	//GetGroupmembersCount
	indivudualMembersCount := len(groupMembersUsers)
	subgroupMembersCount := len(groupMemberGroups)
	groupMembersTotalCount := indivudualMembersCount + subgroupMembersCount

	// department := getDepartmentInTmp(companyID, group.DepartmentID)
	// if department != nil {
	// 	group.Department = *department
	// }
	department, err := c.departmentOps.GetCompanyDepartmentByID(companyID, group.DepartmentID)
	if err != nil {
		c.Response.Status = http.StatusBadRequest
		return c.RenderJSON(err.Error())
	}

	if department != nil {
		group.Department = *department
	}

	result["groups"] = group
	result["groupMembersTotalCount"] = groupMembersTotalCount
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// @Summary Get Group Applications
// @Description This endpoint retrieves applications associated with a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/applications [get]
func (c GroupController) GetGroupApplications(groupID string) revel.Result {
	// TODO: error handling
	result := make(map[string]interface{})
	companyID := c.ViewArgs["companyID"].(string)

	group, resultErr := GetCompanyGroupNew(companyID, groupID)
	if resultErr != nil {
		c.Response.Status = resultErr.HTTPStatusCode
		return c.RenderJSON(resultErr)
	}

	integrations, err := GetGroupIntegrationsNew(groupID, companyID, false)
	if err == nil {
		group.GroupIntegrations = integrations
	}

	for idx, integration := range group.GroupIntegrations {

		companyIntegration, err := ops.GetCompanyIntegration(companyID, integration.IntegrationID)
		if err != nil {
			return c.RenderJSON(err)
			// continue
		}

		// subIntegrations, err := ops.GetSubIntegrations(integration.IntegrationID)
		// if err != nil {}

		_ = companyIntegration

		groupSubIntegrationsResult, err := GetGroupSubIntegration(groupID)
		if err != nil {
			continue
		}

		group.GroupIntegrations[idx].GroupSubIntegrations = groupSubIntegrationsResult
		utils.PrintJSON(companyIntegration, "companyIntegration: ")
		subInteg := []models.GroupSubIntegration{}
		for _, sub := range groupSubIntegrationsResult {
			// get sub integration
			s, err := ops.GetSubIntegration(integration.IntegrationID, sub.IntegrationID)
			if err != nil {

			}
			sub.DisplayPhoto = s.DisplayPhoto

			utils.PrintJSON(s, "GetSubIntegration")

			if sub.ConnectedItemsData == nil {
				sub.ConnectedItemsData = make(map[string]interface{})
			}

			sub.ConnectedItemsData, err = GetGroupApplicationsData(sub, companyIntegration)
			if err != nil {
				c.Response.Status = http.StatusInternalServerError
				return c.RenderJSON(err.Error())
			}
			if sub.ParentIntegrationID == integration.IntegrationID {
				subInteg = append(subInteg, sub)
			}
		}

		group.GroupIntegrations[idx].GroupSubIntegrations = subInteg
	}

	_ = result

	return c.RenderJSON(group.GroupIntegrations)
}

func GetGroupApplicationsData(subIntegration models.GroupSubIntegration, companyIntegration models.CompanyIntegration) (interface{}, error) {
	switch subIntegration.IntegrationSlug {
	case constants.INTEG_SLUG_GOOGLE_ADMIN:
		googleGroups, err := googleoperations.GetGoogleAdminGroups(companyIntegration.IntegrationToken, subIntegration.ConnectedItems)
		if err != nil {
			log.Println("googleAdminError ", err)
			return nil, errors.New(err.Message)
		}
		return googleGroups, nil
	case constants.INTEG_SLUG_GOOGLE_DRIVE:
		googleFiles, err := googleapplicationoperations.GetGoogleDriveFiles(companyIntegration.IntegrationToken, subIntegration.ConnectedItems)
		if err != nil {
			log.Println("googleDriveError ", err)
			return nil, errors.New(err.Message)
		}
		return googleFiles, nil
	case constants.INTEG_SLUG_GCP_PROJECTS, constants.INTEG_SLUG_FIREBASE:
		googleProjects, err := googleapplicationoperations.GetGoogleProjects(companyIntegration.IntegrationToken, companyIntegration.IntegrationTokenExtra, subIntegration.ConnectedItems)
		if err != nil {
			log.Println("gcpProjectsError ", err)
			return nil, errors.New(err.Message)
		}
		return googleProjects, nil
	case constants.INTEG_SLUG_365_GROUPS:
		var azureGroups []officeModels.GetAzureGroupsValue
		accessToken := companyIntegration.IntegrationToken.AccessToken
		azureGroupsData, officeError := officeOperations.GetAzureGroups(accessToken)
		if officeError != nil {
			fmt.Println("officeGroupsError ", officeError.Message)
			return nil, errors.New(officeError.Message)
		}
		for _, item := range subIntegration.ConnectedItems {
			for _, azureGroup := range azureGroupsData.Value {
				if azureGroup.ID == item {
					azureGroups = append(azureGroups, azureGroup)
				}
			}
		}
		return azureGroups, nil
	case constants.INTEG_SLUG_LICENSE:
		var officeLinceses []map[string]interface{}
		accessToken := companyIntegration.IntegrationToken.AccessToken
		officeLicenseResult, officeError := officeOperations.GetOfficeLicense(accessToken)
		if officeError != nil {
			fmt.Println("officeLicenseError ", officeError.Message)
			return nil, errors.New(officeError.Message)
		}
		if officeLicenseResult != nil {
			for _, item := range subIntegration.ConnectedItems {
				for _, license := range officeLicenseResult.Value {
					if license.SkuID == item {
						officeLinceses = append(officeLinceses, map[string]interface{}{
							"id":     license.ID,
							"sku_id": license.SkuID,
							"name":   api.GetLicenseName(license.SkuID),
						})
					}
				}
			}
		}
		return officeLinceses, nil
	case constants.INTEG_SLUG_ONEDRIVE:
		var parentFiles []officeModels.GetOneDriveRootFilesValue
		accessToken := companyIntegration.IntegrationToken.AccessToken
		for _, item := range subIntegration.ConnectedItems {
			fileInfo, officeError := officeOperations.GetFileInfo(item, accessToken)
			if officeError != nil {
				fmt.Println("officeOneDriveError ", officeError.Message)
				return nil, errors.New(officeError.Message)
			}
			if fileInfo != nil {
				parentFiles = append(parentFiles, *fileInfo)
			}
		}
		return parentFiles, nil
	case constants.INTEG_SLUG_JIRA_USER:
		appSecretKey, _ := revel.Config.String("app.encryption.key")
		token := companyIntegration.JiraToken.JiraToken
		identity := companyIntegration.JiraToken.JiraDomain
		decodedData := utils.DecryptString(*token, appSecretKey)

		var connectedJiraGroups []jiraModel.JiraGroupResponse
		for _, item := range subIntegration.ConnectedItems {
			groupResponse, err := jiraoperations.GetJiraGroup(decodedData, *identity, item)
			if err != nil {
				fmt.Println("jiraUserError ", err.Message)
				return nil, errors.New(err.Message)
			}
			connectedJiraGroups = append(connectedJiraGroups, *groupResponse)
		}
		return connectedJiraGroups, nil
	case constants.INTEG_SLUG_JIRA_PROJECT:
		appSecretKey, _ := revel.Config.String("app.encryption.key")
		token := companyIntegration.JiraToken.JiraToken
		identity := companyIntegration.JiraToken.JiraDomain
		decodedData := utils.DecryptString(*token, appSecretKey)

		var connectedJiraProjects []jiraModel.JiraSpecificProject
		for _, item := range subIntegration.ConnectedItems {
			jiraProject, err := jiraoperations.GetJiraProject(decodedData, *identity, item)
			if err != nil {
				fmt.Println("jiraProjectError ", err)
				return nil, errors.New(err.Message)
			}
			connectedJiraProjects = append(connectedJiraProjects, *jiraProject)
		}
		return connectedJiraProjects, nil
	case constants.INTEG_SLUG_BITBUCKET_GROUPS:
		workspace, token := companyIntegration.BitbucketCredentials.Workspace, companyIntegration.BitbucketCredentials.Token
		bitbucketGroups, err := bitbucketOperations.ListGroups(workspace, token)
		if err != nil {
			fmt.Println("bitbucketGroupError ", err)
			return nil, errors.New(err.Message)
		}

		var connectedBitbucketGroups []map[string]interface{}
		for _, groupSlug := range subIntegration.ConnectedItems {
			for _, group := range *bitbucketGroups {
				if group.Slug == groupSlug {
					connectedBitbucketGroups = append(connectedBitbucketGroups, map[string]interface{}{
						"id":    group.Slug,
						"name":  group.Name,
						"value": group.Slug,
						"slug":  group.Slug,
					})
				}
			}
		}
		return connectedBitbucketGroups, nil
	case constants.INTEG_SLUG_BITBUCKET_PROJECTS:
		workspace, token := companyIntegration.BitbucketCredentials.Workspace, companyIntegration.BitbucketCredentials.Token

		var connectedBitbucketProjects []map[string]interface{}

		for _, projectKey := range subIntegration.ConnectedItems {
			getProject, opsErr := bitbucketOperations.GetProject(token, workspace, projectKey)
			if opsErr != nil {
				fmt.Println("bitbucketProjectError ", opsErr)
				return nil, errors.New(opsErr.Message)
			}

			connectedBitbucketProjects = append(connectedBitbucketProjects, map[string]interface{}{
				"name": getProject.Name,
				"slug": getProject.Slug,
			})
		}
		return connectedBitbucketProjects, nil
	case constants.INTEG_SLUG_BITBUCKET_REPOSITORIES:
		workspace, token := companyIntegration.BitbucketCredentials.Workspace, companyIntegration.BitbucketCredentials.Token

		var connectedBitbucketRepo []map[string]interface{}
		for _, repoSlug := range subIntegration.ConnectedItems {
			getRepository, opsErr := bitbucketOperations.GetRepository(token, workspace, repoSlug)
			if opsErr != nil {
				fmt.Println("bitbucketRepoError ", opsErr)
				return nil, errors.New(opsErr.Message)
			}

			connectedBitbucketRepo = append(connectedBitbucketRepo, map[string]interface{}{
				"name": getRepository.Name,
				"slug": getRepository.Slug,
				"id":   getRepository.UUID,
			})
		}

		return connectedBitbucketRepo, nil
	}

	return nil, nil
}

func GetAllCompanyGroupMembers(companyID, groupID string, c *revel.Controller) ([]models.GroupMember, error) {
	var members []models.GroupMember
	var groupMemberIDs []string

	groupMemberIDs = append(groupMemberIDs, groupID)
	g, gErr := GetCompanyGroupNew(companyID, groupID)
	if gErr != nil {
		return members, nil
	}
	if gErr == nil {
		mUsers, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_USER)
		if err != nil {
			// TODO: Handle error
		}

		for idx, memberUser := range mUsers {
			user, opsErr := ops.GetUserByID(memberUser.MemberID)
			if opsErr != nil {
				// TODO: Handle error
			}
			companyUser, opsErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
				UserID:    memberUser.MemberID,
				CompanyID: companyID,
			}, c)
			if opsErr != nil {
				// TODO: Handle error
			}
			// user := getUserInTmp(memberUser.MemberID)
			// companyUser := getCompanyMemberInTmp(companyID, memberUser.MemberID, c)
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
			if companyUser.DisplayPhoto != "" {
				resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, "_100")
				fileName, err := cdn.GetImageFromStorage(resizedPhoto)
				if err == nil {
					companyUser.DisplayPhoto = fileName
				}
			}
			companyUser.Email = user.Email
			companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
			if user.UserID != "" {
				mUsers[idx].MemberInformation = companyUser
				mUsers[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)
				userGroups, err := ops.GetUserCompanyGroupsNew(companyID, memberUser.MemberID)
				if err != nil {
					return nil, err
				}
				mUsers[idx].GroupsCount = len(userGroups)
			}
		}

		members = append(members, mUsers...)

		mGroups, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_GROUP)
		if err != nil {
			// TODO: Handle error
		}

		if len(mGroups) == 0 {
			return members, nil
		}

		for _, group := range mGroups {
			if group.MemberID == groupID {
				continue
			}
			//
			m := GetSubGroupMembers(companyID, group.MemberID, groupID, c)
			// groupMemberIDs = append(groupMemberIDs, group.GroupID)
			// m := GetSubGroupMembers(companyID, group.MemberID, groupMemberIDs)
			members = append(members, m...)
		}
		//
	}

	filteredMembers := FilteredDuplicateGroupMembers(members)
	return filteredMembers, nil
}

type GetAllCompanyGroupMembersWithRoutinesResult struct {
	Members            []models.GroupMember
	DirectMembersCount int
}

func GetAllCompanyGroupMembersWithRoutines(companyID, groupID string, c *revel.Controller) (GetAllCompanyGroupMembersWithRoutinesResult, error) {
	var result GetAllCompanyGroupMembersWithRoutinesResult
	var members []models.GroupMember
	var groupMemberIDs []string

	groupMemberIDs = append(groupMemberIDs, groupID)
	g, gErr := GetCompanyGroupNew(companyID, groupID)

	if gErr != nil {
		return result, nil
	}

	if gErr == nil {
		mUsers, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_USER)
		if err != nil {
			// TODO: Handle error
		}

		// dataChannel := make(chan models.GroupMember)
		// groupDataList := make([]models.Group, len(groups))
		for i, memberUser := range mUsers {
			if companyID != "" && memberUser.UserID != "" {
				// go fetchGroupMemberData(companyID, memberUser, c, dataChannel)
			}
			m := fetchGroupMemberDataNew(companyID, memberUser, c)
			if m != nil {
				mUsers[i] = *m
			}

		}
		// for i := 0; i < len(mUsers); i++ {
		// mUsers[i] = <-dataChannel
		// }

		if len(mUsers) != 0 {
			result.DirectMembersCount = result.DirectMembersCount + len(mUsers)
		}
		members = append(members, mUsers...)

		mGroups, err := GetGroupMembersNew(companyID, g.GroupID, constants.MEMBER_TYPE_GROUP)
		if err != nil {
			// TODO: Handle error
		}

		if len(mGroups) == 0 {
			filteredMembers := FilteredDuplicateGroupMembers(members)
			result.Members = filteredMembers
			return result, nil
		}

		result.DirectMembersCount = result.DirectMembersCount + len(mGroups)

		for _, group := range mGroups {
			if group.MemberID == groupID {
				continue
			}
			//
			m := GetSubGroupMembersWithRoutines(companyID, group.MemberID, groupID, c)
			// groupMemberIDs = append(groupMemberIDs, group.GroupID)
			// m := GetSubGroupMembers(companyID, group.MemberID, groupMemberIDs)
			members = append(members, m...)

		}
		//
	}

	filteredMembers := FilteredDuplicateGroupMembers(members)
	result.Members = filteredMembers

	return result, nil
}

func GetSubGroupMembers(companyID, groupID, excludeGroupID string, c *revel.Controller) []models.GroupMember {
	var members []models.GroupMember

	mUsers, err := GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_USER)
	if err != nil {
		// TODO: Handle error
	}

	members = append(members, mUsers...)

	mGroups, err := GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_GROUP)
	if err != nil {
		return members
	}

	// if len(mGroups) == 0 {
	// 	return members
	// }

	if len(mGroups) != 0 {

		for _, g := range mGroups {
			if g.GroupID == excludeGroupID {
				continue
			}
			m := GetSubGroupMembers(companyID, g.MemberID, excludeGroupID, c)
			// if len(m) == 0 {
			// 	return members
			// }
			members = append(members, m...)
			// if !(utils.StringInSlice(g.GroupID, groupMemberIDs)) {
			// 	groupMemberIDs = append(groupMemberIDs, g.GroupID)
			// 	m := GetSubGroupMembers(companyID, g.MemberID, groupMemberIDs)
			// 	if len(m) == 0 {
			// 		return members
			// 	}
			// 	members = append(members, m...)
			// }
		}
	}

	filteredMembers := FilteredDuplicateGroupMembers(members)

	for idx, memberUser := range filteredMembers {
		user := getUserInTmp(memberUser.MemberID)
		companyUser := getCompanyMemberInTmp(companyID, memberUser.MemberID, c)
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
		if user.UserID != "" {

			filteredMembers[idx].MemberInformation = companyUser
			filteredMembers[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)

			userGroups, err := GetUserCompanyGroupsNew(companyID, memberUser.MemberID)
			if err != nil {
			}
			filteredMembers[idx].GroupsCount = len(userGroups)
		}

	}

	return filteredMembers
}
func GetSubGroupMembersWithRoutines(companyID, groupID, excludeGroupID string, c *revel.Controller) []models.GroupMember {
	var members []models.GroupMember

	mUsers, err := GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_USER)
	if err != nil {
		// TODO: Handle error
	}

	members = append(members, mUsers...)

	mGroups, err := GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_GROUP)
	if err != nil {
		return members
	}

	if len(mGroups) != 0 {

		for _, g := range mGroups {
			if g.GroupID == excludeGroupID {
				continue
			}
			m := GetSubGroupMembers(companyID, g.MemberID, excludeGroupID, c)

			members = append(members, m...)
		}
	}

	filteredMembers := FilteredDuplicateGroupMembers(members)

	// dataChannel := make(chan models.GroupMember)
	for i, memberUser := range filteredMembers {
		m := fetchGroupMemberDataNew(companyID, memberUser, c)
		if m != nil {
			filteredMembers[i] = *m
		}
		// if companyID != "" && memberUser.UserID != "" {
		// 	go fetchGroupMemberData(companyID, memberUser, c, dataChannel)
		// }
	}
	// for i := 0; i < len(mGroups); i++ {
	// 	filteredMembers[i] = <-dataChannel
	// }

	return filteredMembers
}

//GetSubGroupMember
// func GetSubGroupMember (groupID , )

/*
GetAllByCompanyID - Fetch by Company ID
route: /v1/groups/company
Method: GET
params:
status = required* (ACTIVE OR INACTIVE)
company_id = required*
--
key = optional*
include = optional*
department_id = optional*
*/

func (c GroupController) GetAllByCompanyID() revel.Result {
	result := make(map[string]interface{})

	searchKey := c.Params.Query.Get("key")
	status := c.Params.Query.Get("status")
	include := c.Params.Query.Get("include")
	// bookmark := c.Params.Query.Get("bookmark")
	companyID := c.Params.Query.Get("company_id")
	departmentID := c.Params.Query.Get("department_id")
	limit := c.Params.Query.Get("limit")
	// paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")

	groups := []models.Group{}

	// lastEvaluatedKey := models.Group{}

	pageLimit := utils.ConvertStringPageLimitToInt64(limit)
	if searchKey != "" {
		searchKey = strings.ToLower(searchKey)
	}
	if status == "" {
		status = constants.ITEM_STATUS_ACTIVE
	}
	if status != "" {
		status = strings.ToUpper(status)
	}

	// bookmarkGroups := []string{}

	// if bookmark != "" {
	// 	bookMarkErr := json.Unmarshal([]byte(bookmark), &bookmarkGroups)
	// 	if bookMarkErr != nil {
	// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 		result["THIS IS THE ERROR"] = "THIS IS THE ERROR"
	// 		return c.RenderJSON(result)
	// 	}
	// 	for _, group := range bookmarkGroups {
	// 		bookMarkedGroup, err := GetGroupsByGroupID(group, companyID, searchKey, departmentID)
	// 		if err != nil {
	// 			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 			result["THIS IS THE ERROR"] = "THIS IS THE ERROR"
	// 			return c.RenderJSON(result)
	// 		}
	// 		bookMarkedGroups = append(bookMarkedGroups, bookMarkedGroup)
	// 	}
	// }

	// get company user bookmarked groups
	bookMarkedGroups := []models.Group{}

	userID := c.ViewArgs["userID"].(string)
	user, opsError := ops.GetUserByID(userID)
	if opsError != nil {

	}

	bookmarkGroups := user.BookmarkGroups
	for _, group := range bookmarkGroups {
		// TODO: BATCH GET ITEM
		bookMarkedGroup, err := GetGroupsByGroupID(group, companyID, searchKey, departmentID)
		if err == nil && bookMarkedGroup.GroupID != "" {
			bookMarkedGroups = append(bookMarkedGroups, bookMarkedGroup)
		}
	}

	// bookmarkedGroupsAv, err := dynamodbattribute.MarshalList(bookmark)
	// if err != nil {

	// }

	var queryFilter map[string]*dynamodb.Condition
	queryFilter = map[string]*dynamodb.Condition{
		// "GroupID": &dynamodb.Condition{
		// 	ComparisonOperator: aws.String(constants.CONDITION_NOT_EQUAL),
		// 	AttributeValueList: []*dynamodb.AttributeValue{
		// 		{
		// 			L: bookmarkedGroupsAv,
		// 		},
		// 	},
		// },
		"SearchKey": &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(searchKey),
				},
			},
		},
		"Status": &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(status),
				},
			},
		},
	}
	if departmentID != "" {
		queryFilter["DepartmentID"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(departmentID),
				},
			},
		}
	}

	_ = pageLimit

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
		},
		QueryFilter: queryFilter,
		// Limit:            aws.Int64(pageLimit),
		IndexName:        aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		ScanIndexForward: aws.Bool(false),
		// ExclusiveStartKey: utils.MarshalLastEvaluatedKey(models.Group{}, paramLastEvaluatedKey),
	}

	// res, err := ops.HandleQuery(params)
	res, err := ops.HandleQueryLimit(params)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// key := res.LastEvaluatedKey
	// for len(key) != 0 {
	// 	// if int64(len(res.Items)) >= pageLimit {
	// 	// 	break
	// 	// }
	// 	params.ExclusiveStartKey = key
	// 	r, err := ops.HandleQuery(params)
	// 	if err != nil {
	// 		break
	// 	}
	// 	res.Items = append(res.Items, r.Items...)
	// 	key = r.LastEvaluatedKey
	// }

	// err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	// if err == nil {
	// 	result["lastEvaluatedKey"] = lastEvaluatedKey
	// }

	// return c.RenderJSON(res)

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &groups)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		result["THIS IS THE ERROR"] = err.Error()
		return c.RenderJSON(result)
	}

	groups = append(bookMarkedGroups, groups...)
	groups = FilterDuplicatedGroups(groups)
	// if strings.Contains(include, "more") {
	// 	groups = FilterBookmarkGroups(groups, bookmarkGroups)
	// } else {
	// 	groups = FilterDuplicatedGroups(groups)
	// }

	if len(include) != 0 {
		for i, group := range groups {
			_ = i
			_ = group
			if strings.Contains(include, "members") {
				// TODO: Enhance get group members query
				memberUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_USER, c.Controller)
				if err == nil {
					groups[i].OriginalGroupMembers = memberUsers
					groups[i].GroupMembers = memberUsers
				}

				// TODO: Remove owners
				ownerUsers, err := GetGroupOwners(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_OWNER, c.Controller)
				if err == nil {
					groups[i].GroupOwners = ownerUsers
				}

				// /***********START********/
				// //1. GET SUB GROUP MEMBER
				// subGroups, err := GetGroupMembers(companyID, group.GroupID, constants.MEMBER_TYPE_GROUP)
				// if err != nil {
				// 	result["message"] = "Something went wrong with group members"
				// 	result["status"] = utils.GetHTTPStatus(err.Error())
				// 	return c.RenderJSON(result)
				// }

				// //1.2 FETCHED LIST OF SUBGROUPS OF THE CREATED GROUP
				// newSubList := subGroups

				// //1.3 INITIAL COUNT OF SUBGROUPS THAT WERE FETCHED
				// subGroupCount := 0

				// for {

				// 	//2. LOOP THE FETCHED SUBGROUPS
				// 	for _, data := range newSubList {

				// 		//2.2 FETCH SUB GROUP OF THE CURRENT SUBGROUP
				// 		subOfSubGroups, err := GetGroupMembers(companyID, data.MemberID, constants.MEMBER_TYPE_GROUP)
				// 		if err != nil {
				// 			result["message"] = "Something went wrong with group members"
				// 			result["status"] = utils.GetHTTPStatus(err.Error())
				// 			return c.RenderJSON(result)
				// 		}

				// 		//2.3 ITERATE TO SUBGROUP OF SUBGROUP
				// 		for _, subData := range subOfSubGroups {

				// 			flag := false

				// 			//2.4 ITERATE THE INITIAL LIST OF SUBGROUPS
				// 			for _, clonedSubList := range newSubList {
				// 				//2.5 CHECK IF SUBGROUP OF SUBGROUP IS EXISTING IN THE LIST
				// 				if clonedSubList.MemberID == subData.MemberID || group.GroupID == subData.MemberID {
				// 					flag = true
				// 					break
				// 				}
				// 			}

				// 			//2.6 INSERT SUBGROUP OF SUBGROUP IF IT DOES NOT EXIST ON INITIAL LIST OF SUBGROUPS ON STEP 1.2
				// 			if !flag {
				// 				newSubList = append(newSubList, subData)
				// 			}
				// 		}
				// 	}

				// 	//3. STOP LOOP IF EQUAL
				// 	if subGroupCount == len(newSubList) {
				// 		break
				// 	}

				// 	//4. APPEND NEW COUNT
				// 	subGroupCount = len(newSubList)
				// }

				// //5. LOOP THE UPDATED SUBGROUPS TO GET THE GROUP ID
				// for _, getSubGroupID := range newSubList {
				// 	// 5.3 FETCH THE MEMBERS
				// 	subGroupMembers, err := GetGroupMembers(companyID, getSubGroupID.MemberID, constants.MEMBER_TYPE_USER, group.GroupID)
				// 	if err != nil {
				// 		result["message"] = "Something went wrong with group members"
				// 		result["status"] = utils.GetHTTPStatus(err.Error())
				// 		return c.RenderJSON(result)
				// 	}
				// 	// 5.2 LOOP TO MEMBERS
				// 	for _, subGroupMems := range subGroupMembers {
				// 		// 5.3 CHECK IF MEMBER IS EXISTING IN THE MAIN GROUP
				// 		flag := false
				// 		for _, checkMember := range memberUsers {
				// 			if checkMember.MemberID == subGroupMems.MemberID {
				// 				flag = true
				// 				break
				// 			}
				// 		}

				// 		if !flag {
				// 			//INSERT MEMBER FROM SUBGROUPS IF IT DOES NOT EXIST ON INITIAL LIST OF MEMBERS
				// 			memberUsers = append(memberUsers, subGroupMems)
				// 		}
				// 	}
				// }
				// /***********END*********/

				groups[i].GroupMembers = memberUsers
				groups[i].GroupOwners = ownerUsers
			}

			if strings.Contains(include, "integrations") {
				// TODO: Enhance query
				integrationsResult, err := GetGroupIntegrations(group.GroupID)
				if err == nil {
					groups[i].GroupIntegrations = integrationsResult
				}
			}
		}

		// TODO: Remove sub integrations
		for _, group := range groups {
			for i, groupInteg := range group.GroupIntegrations {
				subIntegrationResult, err := GetGroupSubIntegration(group.GroupID)
				if err == nil {
					subInteg := []models.GroupSubIntegration{}
					for _, sub := range subIntegrationResult {
						if sub.ParentIntegrationID == groupInteg.IntegrationID {
							subInteg = append(subInteg, sub)
						}
					}

					group.GroupIntegrations[i].GroupSubIntegrations = subInteg
				}
			}
		}

		//  TODO: Remove members or limit the member calls up to 5
		tmpUsersInformation := map[string]models.CompanyUser{}
		tmpOriginalMembersInformation := map[string]models.CompanyUser{}
		tmpUsersUID := map[string]string{}

		for _, group := range groups {
			for i, member := range group.GroupMembers {
				if val, ok := tmpUsersInformation[member.MemberID]; ok {
					group.GroupMembers[i].MemberInformation = val

					for j, originalMember := range group.OriginalGroupMembers {
						if originalMember.MemberID == member.MemberID {
							if om, ok := tmpOriginalMembersInformation[member.MemberID]; ok {
								group.OriginalGroupMembers[j].MemberInformation = om
							}
						}
					}

					if uid, ok := tmpUsersUID[member.MemberID]; ok {
						group.GroupMembers[i].IntegrationUID = uid
					}
				} else {
					// user, err := ops.GetUserByID(member.MemberID)
					user, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
						UserID:    member.MemberID,
						CompanyID: companyID,
					}, c.Controller)
					if err == nil {
						if user.DisplayPhoto != "" {
							resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")
							fileName, err := cdn.GetImageFromStorage(resizedPhoto)
							if err == nil {
								user.DisplayPhoto = fileName
							}
						}

						group.GroupMembers[i].MemberInformation = user
						tmpUsersInformation[member.MemberID] = user

						for j, originalMember := range group.OriginalGroupMembers {
							if originalMember.MemberID == member.MemberID {
								group.OriginalGroupMembers[j].MemberInformation = user
								tmpOriginalMembersInformation[member.MemberID] = user
							}
						}

						userUID, err := ops.GetIntegrationUID(member.MemberID)
						if err == nil {
							group.GroupMembers[i].IntegrationUID = userUID.IntegrationUID
							tmpUsersUID[member.MemberID] = userUID.IntegrationUID
						}
					}
				}
				// user, err := ops.GetUserByID(member.MemberID)
				// if err == nil {
				// 	if user.DisplayPhoto != "" {
				// 		resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")
				// 		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
				// 		if err == nil {
				// 			user.DisplayPhoto = fileName
				// 		}
				// 	}

				// 	group.GroupMembers[i].MemberInformation = user
				// 	for j, originalMember := range group.OriginalGroupMembers {
				// 		if originalMember.MemberID == member.MemberID {
				// 			group.OriginalGroupMembers[j].MemberInformation = user
				// 		}
				// 	}
				// 	userUID, err := ops.GetIntegrationUID(member.MemberID)
				// 	if err == nil {
				// 		group.GroupMembers[i].IntegrationUID = userUID.IntegrationUID
				// 	}
				// }
			}

		}

		tmpDepartments := map[string]models.Department{}
		for i, group := range groups {
			// TODO: Remove sub group members and add the count on the `GroupMembers`
			subGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_GROUP, c.Controller)
			if err == nil {
				groups[i].SubGroup = subGroups
			}
			qd := ops.GetDepartmentByIDPayload{
				DepartmentID: groups[i].DepartmentID,
				CheckStatus:  true,
			}
			if val, ok := tmpDepartments[groups[i].DepartmentID]; ok {
				groups[i].Department = val
			} else {
				// TODO: Enhance query
				department, opsError := ops.GetDepartmentByID(qd, c.Validation)
				if opsError == nil {
					groups[i].Department = *department
					tmpDepartments[groups[i].DepartmentID] = *department
				}
			}
			// department, opsError := ops.GetDepartmentByID(qd, c.Validation)
			// if opsError == nil {
			// 	groups[i].Department = *department
			// }
			// groups[i].Department = *department // COMMENTED UNTIL FURTHER NOTICE
			// groups[i].Department.DepartmentName = ""

			// TODO: Remove subgroups
			for indx, item := range subGroups {
				user, err := GetGroupMembers(c.ViewArgs["companyID"].(string), item.MemberID, constants.MEMBER_TYPE_USER, c.Controller)
				if err == nil {
					groups[i].SubGroup[indx].GroupMembers = user
				}

				integrationsResult, err := GetGroupIntegrations(item.MemberID)
				if err == nil {
					groups[i].SubGroup[indx].GroupIntegrations = integrationsResult
				}

				for idx, member := range user {
					if val, ok := tmpUsersInformation[member.MemberID]; ok {
						groups[i].SubGroup[indx].GroupMembers[idx].MemberInformation = val
					} else {
						// user, err := ops.GetUserByID(member.MemberID)
						user, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
							UserID:    member.MemberID,
							CompanyID: companyID,
						}, c.Controller)
						if err == nil {
							if user.DisplayPhoto != "" {
								resizedPhoto := utils.ChangeCDNImageSize(user.DisplayPhoto, "_100")
								fileName, err := cdn.GetImageFromStorage(resizedPhoto)
								if err == nil {
									user.DisplayPhoto = fileName
								}
							}

							groups[i].SubGroup[indx].GroupMembers[idx].MemberInformation = user
							tmpUsersInformation[member.MemberID] = user
						}
					}
				}
			}
		}
	}
	result["groups"] = groups
	// result["lastEvaluatedKey"] = lastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// Refactored version of GetAllByCompanyID()

// @Summary Get Company Groups
// @Description This endpoint retrieves groups associated with a specific company, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param search_key query string false "Search Key"
// @Param status query string false "Filter by company status"
// @Param include query string false "Include related data: integrations"
// @Param companyID path string true "Company ID"
// @Param departmentID path string false "Department ID"
// @Param sort query string false "Sort order: ascending/descending"
// @Param limit query int false "Limit the number of results"
// @Param bookmark query string false "Bookmark"
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param all query bool false "All condition"
// @Param with_members_info query bool false "With members info"
// @Param isFetchMore query bool false "Is fetch more"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/company/ [get]
func (c GroupController) GetCompanyGroups() revel.Result {
	result := make(map[string]interface{})

	searchKey := c.Params.Query.Get("key")
	status := c.Params.Query.Get("status")
	include := c.Params.Query.Get("include")
	companyID := c.Params.Query.Get("company_id")
	departmentID := c.Params.Query.Get("department_id")
	sort := c.Params.Query.Get("sort")
	limit := c.Params.Query.Get("limit")
	bookmark := c.Params.Query.Get("bookmark")
	lastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	allCondition := c.Params.Query.Get("all")
	withMembersInfo := c.Params.Query.Get("with_members_info")
	isFetchMore := c.Params.Query.Get("isFetchMore")
	_ = withMembersInfo
	var pageLimit int64
	pageLimit = 30
	all, _ := strconv.ParseBool(allCondition)
	groups := []models.Group{}
	groupsLastEvaluatedKey := models.Group{}
	bookMarkedGroups := []models.Group{}
	if searchKey != "" {
		searchKey = strings.ToLower(searchKey)
	}
	if status == "" {
		status = constants.ITEM_STATUS_ACTIVE
	}
	if status != "" {
		status = strings.ToUpper(status)
	}
	if isFetchMore != "" {
		num, _ := strconv.ParseInt(limit, 10, 64)
		pageLimit = num
	}
	bookmarkGroups := []string{}
	if bookmark != "" {
		bookMarkErr := json.Unmarshal([]byte(bookmark), &bookmarkGroups)
		if bookMarkErr != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "GetCompanyGroups: " + bookMarkErr.Error(),
				Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_400),
			})
		}
		if isFetchMore == "" {
			for _, group := range bookmarkGroups {
				bookMarkedGroup, err := GetCompanyGroupWithFilter(companyID, group, departmentID, searchKey)
				if err != nil {
					c.Response.Status = 400
					return c.RenderJSON(models.ErrorResponse{
						Code:    "400",
						Message: "GetCompanyGroups: " + err.Message,
						Status:  utils.GetHTTPStatus(constants.HTTP_STATUS_400),
					})
				}
				if bookMarkedGroup.GroupID != "" {
					bookMarkedGroups = append(bookMarkedGroups, bookMarkedGroup)

					if len(include) != 0 {
						// bookmarkGroupDataChannel := make(chan models.Group)
						for i, group := range bookMarkedGroups {
							g := fetchGroupDataNew(companyID, include, group, c.Controller)
							if g != nil {
								bookMarkedGroups[i] = *g
							}
							// go fetchGroupData(companyID, include, group, c.Controller, bookmarkGroupDataChannel)
						}
						// for i := 0; i < len(bookMarkedGroups); i++ {
						// 	bookMarkedGroups[i] = <-bookmarkGroupDataChannel
						// }
					}
				}
			}
		}
	}
	queryFilter := map[string]*dynamodb.Condition{
		"Status": &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(status),
				},
			},
		},
	}
	if departmentID != "" {
		queryFilter["DepartmentID"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(departmentID),
				},
			},
		}
	}
	if searchKey != "" {
		queryFilter["SearchKey"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(searchKey),
				},
			},
		}
	}
	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
		QueryFilter: queryFilter,
		Limit:       aws.Int64(pageLimit),
		IndexName:   aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
	}

	if sort == "desc" {
		params.ScanIndexForward = aws.Bool(false)
	}
	if lastEvaluatedKey != "" {
		params.ExclusiveStartKey = utils.MarshalLastEvaluatedKey(models.Group{}, lastEvaluatedKey)
	}
	res, err := ops.HandleQueryWithLimit(params, int(pageLimit), all)
	if err != nil {

	}
	key := res.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &groupsLastEvaluatedKey)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &groups)
	if err != nil {
		// TODO: Standardized errors
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		result["THIS IS THE ERROR"] = err.Error()
		return c.RenderJSON(result)
	}
	var finalGroups []models.Group
	for _, group := range groups {
		if !utils.StringInSlice(group.GroupID, bookmarkGroups) {
			finalGroups = append(finalGroups, group)
		}
	}
	if len(include) != 0 {
		// groupDataChannel := make(chan models.Group)
		for i, group := range finalGroups {
			// go fetchGroupData(companyID, include, group, c.Controller, groupDataChannel)
			g := fetchGroupDataNew(companyID, include, group, c.Controller)
			if g != nil {
				finalGroups[i] = *g
			}
		}
		// for i := 0; i < len(finalGroups); i++ {
		// 	finalGroups[i] = <-groupDataChannel
		// }
		// for i, group := range groups {
		// if strings.Contains(include, "members") {
		// membersUsers, err := GetGroupMembersNew(companyID, group.GroupID, constants.MEMBER_TYPE_USER)
		// if err != nil {
		// 	// TODO: Handle error
		// }

		// memberGroups, err := GetGroupMembersNew(companyID, group.GroupID, constants.MEMBER_TYPE_GROUP)
		// if err != nil {
		// 	// TODO: Handle error
		// }

		// 	members, err := GetAllCompanyGroupMembers(companyID, group.GroupID, c.Controller)
		// 	if err != nil {
		// 		c.Response.Status = 500
		// 		return c.RenderJSON(err)
		// 	}

		// 	// groups[i].MembersCount = len(membersUsers) + len(memberGroups)
		// 	groups[i].MembersCount = len(members)

		// 	if withMembersInfo == constants.BOOL_TRUE {
		// 		// allMembers := append(allMembers, memberGroups...)
		// 		groups[i].GroupMembers = members
		// 	}
		// }

		// if strings.Contains(include, "integrations") {
		// 	integrationsResult, err := GetGroupIntegrationsNew(group.GroupID, companyID)
		// 	if err == nil {
		// 		groups[i].GroupIntegrations = integrationsResult
		// 	}
		// }
		// }
	}
	if isFetchMore == "" {
		result["bookmarkGroups"] = bookMarkedGroups
	} else {
		result["bookmarkGroups"] = ""
	}
	result["groups"] = finalGroups
	result["lastEvaluatedKey"] = groupsLastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

type CreateCronJobGroupMembersPayload struct {
	SelectedDate   string   `json:"selectedDate,omitempty"`
	NumberOfDays   int      `json:"numberOfDays,omitempty"`
	Users          []string `json:"users,omitempty"`
	SubGroups      []string `json:"subGroups,omitempty"`
	GroupID        string   `json:"groupId,omitempty"`
	CronJobType    string   `json:"cronJobType,omitempty"`
	SelectedGroups []string `json:"selectedGroups,omitempty"`
}

// @Summary Create Cron Job For Group Members
// @Description This endpoint creates a cron job for managing group members, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.CreateCronJobGroupMembersRequest true "Create cron job group members body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/cron_job [post]
func (c GroupController) CreateCronJobGroupMembers() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)
	var payload CreateCronJobGroupMembersPayload
	c.Params.BindJSON(&payload)

	dateString := utils.GetDateNowUnixFormat(payload.NumberOfDays)

	var membersCronJobData []models.MemberCronJobData
	for _, userId := range payload.Users {
		membersCronJobData = append(membersCronJobData, models.MemberCronJobData{
			MemberID:   userId,
			MemberType: constants.MEMBER_TYPE_USER,
		})
	}

	for _, subGroupId := range payload.SubGroups {
		membersCronJobData = append(membersCronJobData, models.MemberCronJobData{
			MemberID:   subGroupId,
			MemberType: constants.MEMBER_TYPE_GROUP,
		})
	}

	JobUUID := uuid.NewV4().String()

	groupMemberCronJob := &models.CronJob{
		PK:            utils.AppendPrefix(constants.PREFIX_JOB, dateString+"#"+JobUUID),
		SK:            utils.AppendPrefix(constants.PREFIX_JOB, constants.JOB_GROUP_MEMBERS),
		CompanyID:     companyID,
		GroupID:       payload.GroupID,
		Status:        constants.ITEM_STATUS_PENDING,
		SelectedDate:  dateString,
		Type:          payload.CronJobType,
		Members:       membersCronJobData,
		CurrentUserID: userID,
		CreatedAt:     utils.GetCurrentTimestamp(),
		NumberOfDays:  strconv.Itoa(payload.NumberOfDays),
	}

	if payload.CronJobType == constants.JOB_MOVE_GROUP_MEMBERS {
		groupMemberCronJob.SelectedGroups = payload.SelectedGroups
	}

	cronJob := ops.NewCronJob(groupMemberCronJob)
	cronJob.CreateCronJob()

	c.Response.Status = 201
	return nil
}

// @Summary Move Members To Another Group
// @Description This endpoint moves members from one group to another, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.MoveMembersToAnotherGroupRequest true "Move members to another group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/move [post]
func (c GroupController) MoveMembersToAnotherGroups() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)
	var payload CreateCronJobGroupMembersPayload
	c.Params.BindJSON(&payload)

	var members []models.MemberCronJobData
	for _, individualID := range payload.Users {
		members = append(members, models.MemberCronJobData{
			MemberID:   individualID,
			MemberType: constants.MEMBER_TYPE_USER,
		})
	}

	for _, groupID := range payload.SubGroups {
		members = append(members, models.MemberCronJobData{
			MemberID:   groupID,
			MemberType: constants.MEMBER_TYPE_GROUP,
		})
	}

	// Pass skipLogging as false for immediate moves (to create logs)
	err := MoveMembersCronJobHandler(companyID, payload.GroupID, userID, members, payload.SelectedGroups, false)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err)
	}

	c.Response.Status = 201
	return nil
}

func ExecuteMoveMembersCronJob() error {
	now := time.Now()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	unix := day.Unix()
	dateNow := strconv.FormatInt(unix, 10)

	onBoardingCronJobs, err := GetCronJobsGroupMembers(dateNow, constants.JOB_MOVE_GROUP_MEMBERS)
	if err != nil {
		fmt.Println("JOB ERROR ", err.Error())
		return err
	}
	fmt.Println("TEST FETCHING ON BOARDING JOBS", len(onBoardingCronJobs))
	var cronJobErrors []error
	for _, cronJob := range onBoardingCronJobs {
		if cronJob.Status == constants.ITEM_STATUS_PENDING && cronJob.Type == constants.JOB_MOVE_GROUP_MEMBERS {
			// Pass skipLogging as false for scheduled moves (to create logs)
			err := MoveMembersCronJobHandler(cronJob.CompanyID, cronJob.GroupID, cronJob.CurrentUserID, cronJob.Members, cronJob.SelectedGroups, false)
			if err != nil {
				return err
			}

			//* Update cron job status
			err = UpdateDoneCronJob(cronJob.PK, cronJob.SK, dateNow)
			if err != nil {
				cronJobErrors = append(cronJobErrors, err)
			}
		}
	}
	log.Println(cronJobErrors)
	return nil
}

func MoveMembersCronJobHandler(companyID, groupID, userID string, members []models.MemberCronJobData, groupsToMoveIn []string, skipLogging bool) error {
	//* Remove members to group and to connected sub applications
	errRemoveMember := RemoveMembersCronJobHandler(groupID, companyID, userID, members, true) // Keep skipLogging as true to prevent remove logs
	if errRemoveMember != nil {
		return errRemoveMember
	}

	//* Get groups data
	individuals, subGroups := GetMembersCronJobDataType(members)
	for _, selectedGroupID := range groupsToMoveIn {
		// Add members to new group with skipLogging true to prevent add logs
		err := AddIndividualMembersCronJobHandler(selectedGroupID, companyID, userID, "", individuals, true)
		if err != nil {
			return err
		}

		err = AddSubGroupsCronJobHandler(companyID, selectedGroupID, userID, subGroups, true)
		if err != nil {
			return err
		}

		// Create only move logs
		if !skipLogging {
			// Get source group name
			sourceGroup, errSource := GetGroupByID(groupID)
			if errSource != nil {
				return errSource
			}

			// Get destination group name
			destGroup, errDest := GetGroupByID(selectedGroupID)
			if errDest != nil {
				return errDest
			}

			var logInfoUsers []models.LogModuleParams
			for _, member := range members {
				logInfoUsers = append(logInfoUsers, models.LogModuleParams{
					ID:   member.MemberID,
					Type: member.MemberType,
				})
			}

			// Create company level move log
			companyLog := models.Logs{
				CompanyID: companyID,
				UserID:    userID,
				LogAction: constants.LOG_ACTION_MOVE_GROUP_MEMBERS,
				LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
				LogInfo: &models.LogInformation{
					Users: logInfoUsers,
					SourceGroup: &models.LogModuleParams{
						ID:   groupID,
						Name: sourceGroup.GroupName,
					},
					DestinationGroup: &models.LogModuleParams{
						ID:   selectedGroupID,
						Name: destGroup.GroupName,
					},
				},
			}

			// Create group level move log
			groupLog := models.Logs{
				GroupID:   selectedGroupID,
				UserID:    userID,
				LogAction: constants.LOG_ACTION_MOVE_GROUP_MEMBERS,
				LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
				LogInfo: &models.LogInformation{
					Users: logInfoUsers,
					SourceGroup: &models.LogModuleParams{
						ID:   groupID,
						Name: sourceGroup.GroupName,
					},
					DestinationGroup: &models.LogModuleParams{
						ID:   selectedGroupID,
						Name: destGroup.GroupName,
					},
				},
			}

			var errorMessages []string

			_, err = ops.InsertLog(companyLog)
			if err != nil {
				errorMessages = append(errorMessages, "Error while creating company logs for moving members")
			}

			_, err = ops.InsertLog(groupLog)
			if err != nil {
				errorMessages = append(errorMessages, "Error while creating group logs for moving members")
			}

			if len(errorMessages) > 0 {
				return errors.New(strings.Join(errorMessages, ", "))
			}
		}
	}
	return nil
}

// @Summary Get Group Integrations
// @Description This endpoint retrieves integrations associated with a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/integrations/:groupID [get]
func (c GroupController) GetGroupIntegrations(groupID string) revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	group, resultErr := GetCompanyGroupNew(companyID, groupID)
	if resultErr != nil {
		c.Response.Status = resultErr.HTTPStatusCode
		return c.RenderJSON(resultErr)
	}

	integrations, err := GetGroupIntegrationsNew(groupID, companyID, false)
	if err == nil {
		group.GroupIntegrations = integrations
	}

	for idx, integration := range group.GroupIntegrations {
		subIntegrations, err := GetSubIntegrations(integration.IntegrationID)
		if err != nil {
			return nil
		}

		subIntegrationsResult, err := GetGroupSubIntegration(groupID)
		if err != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        err.Error(),
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
			})
		}

		//group.GroupIntegrations[idx].GroupSubIntegrations = subIntegrationsResult

		groupSubInteg := []models.GroupSubIntegration{}
		for _, sub := range subIntegrationsResult {
			if sub.ParentIntegrationID == integration.IntegrationID {
				//* Attached display photo and integration name
				for _, subInteg := range subIntegrations {
					if subInteg.IntegrationID == sub.IntegrationID {
						sub.DisplayPhoto = subInteg.DisplayPhoto
						sub.IntegrationName = subInteg.IntegrationName
					}
				}
				groupSubInteg = append(groupSubInteg, sub)
			}
		}

		group.GroupIntegrations[idx].GroupSubIntegrations = groupSubInteg
	}

	if group.GroupIntegrations == nil {
		c.Response.Status = 404
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 404,
			Message:        "No connected integrations to this group.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		})
	}

	return c.RenderJSON(group.GroupIntegrations)
}

func UpdateDoneCronJob(PK, SK, dateNow string) error {
	// Update cron job status to DONE
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":st": {
				S: aws.String(constants.ITEM_STATUS_DONE),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
		ExpressionAttributeNames: map[string]*string{
			"#s":  aws.String("Status"),
			"#ua": aws.String("UpdatedAt"),
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(PK),
			},
			"SK": {
				S: aws.String(SK),
			},
		},
		UpdateExpression: aws.String("SET #s = :st, #ua = :ua"),
	}

	// run update item
	_, updateItemErr := app.SVC.UpdateItem(input)
	if updateItemErr != nil {
		return updateItemErr
	}

	return nil
}

func ExecuteAddMembersCronJob() error {
	now := time.Now()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	unix := day.Unix()
	dateNow := strconv.FormatInt(unix, 10)

	onBoardingCronJobs, err := GetCronJobsGroupMembers(dateNow, constants.JOB_ADD_GROUP_MEMBERS)
	if err != nil {
		fmt.Println("JOB ERROR ", err.Error())
		return err
	}
	fmt.Println("TEST FETCHING ON BOARDING JOBS", len(onBoardingCronJobs))

	var individualsData []models.MemberCronJobData
	var subGroupsData []models.MemberCronJobData
	var cronJobErrors []error
	for _, cronJob := range onBoardingCronJobs {
		if cronJob.Status == constants.ITEM_STATUS_PENDING && cronJob.Type == constants.JOB_ADD_GROUP_MEMBERS {
			for _, memberData := range cronJob.Members {
				if memberData.MemberType == constants.MEMBER_TYPE_USER {
					individualsData = append(individualsData, memberData)
				} else {
					subGroupsData = append(subGroupsData, memberData)
				}
			}

			//* Save individual members
			err := AddIndividualMembersCronJobHandler(cronJob.GroupID, cronJob.CompanyID, cronJob.CurrentUserID, cronJob.Type, individualsData, false)
			if err != nil {
				cronJobErrors = append(cronJobErrors, err)
			}
			//* Save sub groups
			err = AddSubGroupsCronJobHandler(cronJob.CompanyID, cronJob.GroupID, cronJob.CurrentUserID, subGroupsData, false) // Added false for skipLogging
			if err != nil {
				cronJobErrors = append(cronJobErrors, err)
			}
			//* Update cron job status
			err = UpdateDoneCronJob(cronJob.PK, cronJob.SK, dateNow)
			if err != nil {
				cronJobErrors = append(cronJobErrors, err)
			}
		}
	}
	log.Println(cronJobErrors)
	return nil
}

func ExecuteRemoveMembersCronJob() error {
	now := time.Now()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	unix := day.Unix()
	dateNow := strconv.FormatInt(unix, 10)

	removeMembersCronJob, err := GetCronJobsGroupMembers(dateNow, constants.JOB_REMOVE_GROUP_MEMBERS)
	if err != nil {
		fmt.Println("JOB ERROR ", err.Error())
		return err
	}
	fmt.Println("TEST FETCHING remove members cron JOBS", len(removeMembersCronJob))
	var cronJobErrors []error
	for _, cronJob := range removeMembersCronJob {
		if cronJob.Status == constants.ITEM_STATUS_PENDING && cronJob.Type == constants.JOB_REMOVE_GROUP_MEMBERS {
			err := RemoveMembersCronJobHandler(cronJob.GroupID, cronJob.CompanyID, cronJob.CurrentUserID, cronJob.Members, false) // Added false for skipLogging
			if err != nil {
				return err
			}

			//* Update cron job status
			err = UpdateDoneCronJob(cronJob.PK, cronJob.SK, dateNow)
			if err != nil {
				cronJobErrors = append(cronJobErrors, err)
			}
		}
	}
	log.Println(cronJobErrors)
	return nil
}

func GetGroupIntegrationsCronJob(groupID, companyID string) (*models.Group, error) {
	group, resultErr := GetCompanyGroupNew(companyID, groupID)
	if resultErr != nil {
		return nil, fmt.Errorf("error code: %q, error message: %q", resultErr.Code, resultErr.Message)
	}
	integrations, err := GetGroupIntegrationsNew(groupID, companyID, false)
	if err == nil {
		group.GroupIntegrations = integrations
	}

	for idx, integration := range group.GroupIntegrations {
		subIntegrationsResult, err := GetGroupSubIntegration(groupID)
		if err != nil {
			return nil, err
		}
		group.GroupIntegrations[idx].GroupSubIntegrations = subIntegrationsResult

		subInteg := []models.GroupSubIntegration{}
		for _, sub := range subIntegrationsResult {
			if sub.ParentIntegrationID == integration.IntegrationID {
				subInteg = append(subInteg, sub)
			}
		}

		group.GroupIntegrations[idx].GroupSubIntegrations = subInteg
	}

	return &group, nil
}

func GetMembersCronJobDataType(members []models.MemberCronJobData) ([]models.MemberCronJobData, []models.MemberCronJobData) {
	var individuals []models.MemberCronJobData
	var subGroups []models.MemberCronJobData

	for _, member := range members {
		if member.MemberType == constants.MEMBER_TYPE_USER {
			individuals = append(individuals, member)
		} else {
			subGroups = append(subGroups, member)
		}
	}

	return individuals, subGroups
}

func RemoveMembersCronJobHandler(groupID, companyID, userID string, members []models.MemberCronJobData, skipLogging bool) error {
	//* Get group integrations and remove members to connected applications
	group, err := GetGroupIntegrationsCronJob(groupID, companyID)
	if err != nil {
		return nil
	}
	//* Get group members any type
	groupMembers, err := ops.GetGroupMembersAnyType(companyID, groupID)
	if err != nil {
		return err
	}

	//* Google Config
	config, err := googleoperations.GetGoogleConfig()
	if err != nil {
		return err
	}

	groupIntegrationOps := &ops.GroupIntegraitonOperations{Config: config}

	//* Remove group member to connected sub applications
	for _, member := range groupMembers {
		//* Get individual's data
		user, userErr := ops.GetUserByIDNew(member.MemberID)
		if userErr != nil {
			return errors.New(strings.ToLower(userErr.Message))
		}
		err := groupIntegrationOps.RemoveMemberToConnectedSubApplications(companyID, user.Email, user.UserID, member, *group)
		if err != nil {
			return errors.New(err.Message)
		}
	}

	logSearchKey := group.GroupName + " "
	var departmentMembersToRemove []models.DepartmentMember
	var recipients []mail.Recipient

	for _, member := range members {
		userData, opsErr := ops.GetUserData(member.MemberID, "")
		if userData.UserID == "" || opsErr != nil {
			return err
		}

		groupMember := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, member.MemberID)),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err := app.SVC.DeleteItem(groupMember)
		if err != nil {
			return err
		}

		departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, group.DepartmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, member.MemberID),
			DepartmentID: group.DepartmentID,
			UserID:       member.MemberID,
		})

		recipients = append(recipients, mail.Recipient{
			Name:       userData.FirstName + " " + userData.LastName,
			Email:      userData.Email,
			GroupName:  group.GroupName,
			ActionType: "removed",
		})
		logSearchKey += userData.FirstName + " " + userData.LastName + " "
	}

	if len(departmentMembersToRemove) != 0 {
		var filteredMembersToRemove []models.DepartmentMember
		for _, deptMember := range departmentMembersToRemove {
			skip := false
			for _, u := range filteredMembersToRemove {
				if deptMember.PK == u.PK && deptMember.SK == u.SK {
					skip = true
					break
				}
			}
			if !skip {
				filteredMembersToRemove = append(filteredMembersToRemove, deptMember)
			}
		}

		deptUserRemoveErr := RemoveDepartmentUsers(filteredMembersToRemove)
		if deptUserRemoveErr != nil {
			return errors.New("error while removing users from a department.")
		}
	}

	//send email for removed members
	jobs.Now(mail.SendEmail{
		Subject:    "You have been removed from a group",
		Recipients: recipients,
		Template:   "notify_group_member.html",
	})

	// Only create logs if skipLogging is false
	if !skipLogging {
		var logInfoUsers []models.LogModuleParams
		for _, member := range members {
			logInfoUsers = append(logInfoUsers, models.LogModuleParams{
				ID: member.MemberID,
			})
		}

		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    userID,
			LogAction: constants.LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: groupID,
				},
			},
		}

		groupLog := models.Logs{
			GroupID:   groupID,
			UserID:    userID,
			LogAction: constants.LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: groupID,
				},
			},
		}

		var errorMessages []string

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "error while creating company log")
		}

		_, err = ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "error while creating group log")
		}

		// Check if user has other groups and create action item if needed
		hasGroup := false
		for _, member := range members {
			userGroups, err := GetUserCompanyGroups(companyID, member.MemberID)
			if err != nil {
				return errors.New("something went wrong with group member groups")
			}
			if len(userGroups) > 0 {
				hasGroup = true
				break
			}
		}

		if !hasGroup {
			actionItem := models.ActionItem{
				CompanyID:      companyID,
				LogID:          companyLogID,
				ActionItemType: "ADD_REMOVE_USER",
				SearchKey:      logSearchKey,
			}
			_, err := ops.CreateActionItem(actionItem)
			if err != nil {
				errorMessages = append(errorMessages, "error while creating action items")
			}
		}

		if len(errorMessages) > 0 {
			return errors.New(strings.Join(errorMessages, ", "))
		}
	}

	return nil
}

func AddIndividualMembersCronJobHandler(groupID, companyID, userID, cronJobType string, membersData []models.MemberCronJobData, skipLogging bool) error {
	group, err := GetGroupByID(groupID)
	if err != nil {
		return err
	}

	departmentID := group.DepartmentID

	var members []models.GroupMember
	var inputRequest []*dynamodb.WriteRequest
	var membersInput *dynamodb.BatchWriteItemInput
	var departmentMembers []models.DepartmentMember
	var recipients []mail.Recipient

	for _, member := range membersData {
		userData, err := ops.GetUserData(member.MemberID, "")
		if userData.UserID == "" || err != nil {
			return err
		}

		members = append(members, models.GroupMember{
			PK:         utils.AppendPrefix(constants.PREFIX_GROUP, groupID),
			SK:         utils.AppendPrefix(constants.PREFIX_USER, member.MemberID),
			CompanyID:  companyID,
			GroupID:    groupID,
			MemberID:   member.MemberID,
			Status:     constants.ITEM_STATUS_ACTIVE,
			MemberType: member.MemberType,
			MemberRole: constants.MEMBER_TYPE_USER,
			CreatedAt:  utils.GetCurrentTimestamp(),
			UpdatedAt:  utils.GetCurrentTimestamp(),
			Type:       constants.ENTITY_TYPE_GROUP_MEMBER,
		})

		departmentMembers = append(departmentMembers, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, userData.UserID),
			DepartmentID: departmentID,
			UserID:       userData.UserID,
		})

		recipients = append(recipients, mail.Recipient{
			Name:       userData.FirstName + " " + userData.LastName,
			Email:      userData.Email,
			GroupName:  group.GroupName,
			ActionType: "added",
		})
	}

	for _, member := range members {
		inputRequest = append(inputRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK":         {S: aws.String(member.PK)},
				"SK":         {S: aws.String(member.SK)},
				"CompanyID":  {S: aws.String(member.CompanyID)},
				"GroupID":    {S: aws.String(member.GroupID)},
				"MemberID":   {S: aws.String(member.MemberID)},
				"Status":     {S: aws.String(member.Status)},
				"MemberType": {S: aws.String(member.MemberType)},
				"MemberRole": {S: aws.String(member.MemberRole)},
				"CreatedAt":  {S: aws.String(member.CreatedAt)},
				"UpdatedAt":  {S: aws.String(member.UpdatedAt)},
				"Type":       {S: aws.String(member.Type)},
			},
		}})
	}

	membersInput = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			app.TABLE_NAME: inputRequest,
		},
	}

	batchRes, err := app.SVC.BatchWriteItem(membersInput)
	_ = batchRes
	if err != nil {
		return err
	}

	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptAddUsersErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptAddUsersErr != nil {
		return errors.New("error adding users to department")
	}

	// Only create logs if skipLogging is false
	if !skipLogging {
		logSearchKey := group.GroupName + " "
		var logInfoUsers []models.LogModuleParams
		for _, member := range members {
			u := getUserInTmp(member.MemberID)
			logSearchKey += u.FirstName + " " + u.LastName + " "
			logInfoUsers = append(logInfoUsers, models.LogModuleParams{
				ID: member.MemberID,
			})
		}

		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    userID,
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: groupID,
				},
			},
		}

		groupLog := models.Logs{
			GroupID:   groupID,
			UserID:    userID,
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: groupID,
				},
			},
		}

		var errorMessages []string

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "error while creating company logs")
		}

		_, err = ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "error while creating group logs")
		}

		if len(errorMessages) > 0 {
			return errors.New(strings.Join(errorMessages, ", "))
		}

		groupList, err := GetGroupList(companyID)
		if err != nil {
			return err
		}

		output := len(groupList)

		// generate action items?
		if output >= 2 {
			actionItem := models.ActionItem{
				CompanyID:      companyID,
				LogID:          companyLogID,
				ActionItemType: "ADD_USERS_TO_GROUP",
				SearchKey:      logSearchKey,
			}

			actionItemID, err := ops.CreateActionItem(actionItem)
			if err != nil {
				return errors.New("error while creating action items")
			}
			_ = actionItemID
		}

		//* Google Config
		config, err := googleoperations.GetGoogleConfig()
		if err != nil {
			return err
		}

		groupIntegrationOps := &ops.GroupIntegraitonOperations{Config: config}

		//* Add group members to connected integrations
		for _, member := range members {
			//* Get individual's data
			user, userErr := ops.GetUserByIDNew(member.MemberID)
			if userErr != nil {
				return errors.New(strings.ToLower(userErr.Message))
			}
			err := groupIntegrationOps.AddMemberToConnectedSubApplications(companyID, user.Email, user.UserID, user.FirstName, user.LastName, member.MemberID, member.MemberType, group)
			if err != nil {
				return errors.New(err.Message)
			}
		}
	}

	return nil
}

func AddSubGroupsCronJobHandler(companyID, groupID, userID string, membersData []models.MemberCronJobData, skipLogging bool) error {
	group, err := GetGroupByID(groupID)
	if err != nil {
		return err
	}

	var memberGroups []models.GroupMember
	var groups []models.Group
	for _, memberData := range membersData {
		group, err := GetGroupByID(memberData.MemberID)
		if err != nil {
			return err
		}
		groups = append(groups, group)
	}

	marshalGroups, err := json.Marshal(groups)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(marshalGroups), &memberGroups)
	if err != nil {
		return err
	}

	var departmentMembers []models.DepartmentMember
	for _, group := range memberGroups {
		for _, user := range group.GroupMembers {
			userData, err := ops.GetUserData(user.MemberID, "")

			if userData.UserID == "" || err != nil {
				return err
			}

			departmentMembers = append(departmentMembers, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, group.DepartmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, userData.UserID),
				DepartmentID: group.DepartmentID,
				UserID:       userData.UserID,
			})
		}
	}

	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptAddUsersErr := AddDepartmentUsers(filteredDepartmentMembers, group.DepartmentID)
	if deptAddUsersErr != nil {
		return errors.New("error adding users to department")
	}

	batchLimit := 25
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for _, member := range memberGroups {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string))),
				},
				"GroupID": {
					S: aws.String(group.GroupID),
				},
				"GroupName": {
					S: aws.String(member.GroupName),
				},
				"MemberID": {
					S: aws.String(utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string)),
				},
				"DepartmentID": {
					S: aws.String(member.DepartmentID),
				},
				"CompanyID": {
					S: aws.String(companyID),
				},
				"Status": {
					S: aws.String(constants.ITEM_STATUS_ACTIVE),
				},
				"MemberType": {
					S: aws.String(constants.MEMBER_TYPE_GROUP),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
				},
				"CreatedAt": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
		}})

		if len(memberGroups)%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError := ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		e := errors.New(batchError.Error())
		return e
	}

	// generate log
	var logInfoGroups []models.LogModuleParams
	for _, member := range memberGroups {
		logInfoGroups = append(logInfoGroups, models.LogModuleParams{
			ID:   member.GroupID,
			Name: member.Name,
		})
	}

	companyLog := models.Logs{
		CompanyID: companyID,
		UserID:    userID,
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Groups: logInfoGroups,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    userID,
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Groups: logInfoGroups,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	var errorMessages []string

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "error while creating company logs")
	}
	_ = companyLogID

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "error while creating group logs")
	}
	_ = groupLogID

	return nil
}

func AddGroupMember(groupMember models.GroupMember) error {
	groupMemberData, err := dynamodbattribute.MarshalMap(groupMember)
	if err != nil {
		return err
	}

	groupMemberInput := &dynamodb.PutItemInput{
		Item:      groupMemberData,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(groupMemberInput)
	if err != nil {
		return err
	}

	return nil
}

func GetCronJobsGroupMembers(dateNow, cronJobType string) ([]models.CronJob, error) {
	var membersOnBoarding []models.CronJob

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_JOB, dateNow)),
					},
				},
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_JOB, constants.JOB_GROUP_MEMBERS)),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_PENDING),
					},
				},
			},
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(cronJobType),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
	}

	res, err := ops.HandleQuery(params)
	if err != nil {
		return membersOnBoarding, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &membersOnBoarding)
	if err != nil {
		return membersOnBoarding, err
	}

	return membersOnBoarding, nil
}

func fetchGroupData(companyId, include string, group models.Group, c *revel.Controller, groupDataChannel chan<- models.Group) {
	if strings.Contains(include, "members") {

		membersResult, err := GetAllCompanyGroupMembersWithRoutines(companyId, group.GroupID, c)
		if err != nil {

		}
		group.GroupMembers = membersResult.Members
		group.MembersCount = membersResult.DirectMembersCount
	}
	if strings.Contains(include, "integrations") {
		isSubIntegrationInclude := false
		if strings.Contains(include, "subIntegrations") {
			isSubIntegrationInclude = true
		}

		integrationsResult, err := GetGroupIntegrationsNew(group.GroupID, companyId, isSubIntegrationInclude)
		if err != nil {

		}
		group.GroupIntegrations = integrationsResult
	}

	var department models.Department

	qd := ops.GetDepartmentByIDPayload{
		CompanyID:    companyId,
		DepartmentID: group.DepartmentID,
		CheckStatus:  true,
	}
	// if errCache := cache.Get("department_"+group.DepartmentID, &department); errCache != nil {
	departmentResult, err := ops.GetDepartmentByIDNew(qd, c.Validation)
	if err == nil {
		// go cache.Set("department_"+group.GroupID, departmentResult, 30*time.Minute)
		group.Department = *departmentResult

	}
	// }
	if department.DepartmentID != "" {
		group.Department = department
	}

	groupDataChannel <- group
}

func fetchGroupDataNew(companyId, include string, group models.Group, c *revel.Controller) *models.Group {
	if strings.Contains(include, "members") {

		membersResult, err := GetAllCompanyGroupMembersWithRoutines(companyId, group.GroupID, c)
		if err != nil {

		}
		group.GroupMembers = membersResult.Members
		group.MembersCount = membersResult.DirectMembersCount
	}
	if strings.Contains(include, "integrations") {
		isSubIntegrationInclude := false
		if strings.Contains(include, "subIntegrations") {
			isSubIntegrationInclude = true
		}

		integrationsResult, err := GetGroupIntegrationsNew(group.GroupID, companyId, isSubIntegrationInclude)
		if err != nil {

		}
		group.GroupIntegrations = integrationsResult
	}

	var department models.Department

	qd := ops.GetDepartmentByIDPayload{
		CompanyID:    companyId,
		DepartmentID: group.DepartmentID,
		CheckStatus:  true,
	}
	// if errCache := cache.Get("department_"+group.DepartmentID, &department); errCache != nil {
	departmentResult, err := ops.GetDepartmentByIDNew(qd, c.Validation)
	if err == nil {
		// go cache.Set("department_"+group.GroupID, departmentResult, 30*time.Minute)
		group.Department = *departmentResult

	}
	// }
	if department.DepartmentID != "" {
		group.Department = department
	}

	return &group
}
func fetchGroupMemberData(companyId string, memberUser models.GroupMember, c *revel.Controller, memberDataChannel chan<- models.GroupMember) {
	memberCache := models.GroupMember{}
	// if errCache := cache.Get("member_"+memberUser.MemberID, &memberCache); errCache != nil {
	user, opsErr := ops.GetUserByID(memberUser.MemberID)
	if opsErr != nil {
		// TODO: Handle error
	}
	companyUser, opsErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    memberUser.MemberID,
		CompanyID: companyId,
	}, c)
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
	if companyUser.DisplayPhoto != "" {
		resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, "_100")
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err == nil {
			companyUser.DisplayPhoto = fileName
		}
	}
	companyUser.Email = user.Email
	companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
	if user.UserID != "" {
		memberUser.MemberInformation = companyUser
		memberUser.Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)

		userGroups, err := ops.GetUserConnectedGroups(companyId, memberUser.MemberID)
		if err != nil {
		}
		memberUser.GroupsCount = len(userGroups)

		// go cache.Set("member_"+memberUser.MemberID, memberUser, 30*time.Minute)

	}
	// }

	if memberCache.PK != "" {
		memberDataChannel <- memberCache

	} else {
		memberDataChannel <- memberUser
	}

}

func fetchGroupMemberDataNew(companyId string, memberUser models.GroupMember, c *revel.Controller) *models.GroupMember {

	user, opsErr := ops.GetUserByID(memberUser.MemberID)
	if opsErr != nil {
		// TODO: Handle error
	}
	companyUser, opsErr := ops.GetCompanyMember(ops.GetCompanyMemberParams{
		UserID:    memberUser.MemberID,
		CompanyID: companyId,
	}, c)
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
	if companyUser.DisplayPhoto != "" {
		resizedPhoto := utils.ChangeCDNImageSize(companyUser.DisplayPhoto, "_100")
		fileName, err := cdn.GetImageFromStorage(resizedPhoto)
		if err == nil {
			companyUser.DisplayPhoto = fileName
		}
	}
	companyUser.Email = user.Email
	companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
	if user.UserID != "" {
		memberUser.MemberInformation = companyUser
		memberUser.Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)

		userGroups, err := ops.GetUserConnectedGroups(companyId, memberUser.MemberID)
		if err != nil {
		}
		memberUser.GroupsCount = len(userGroups)
	}

	return &memberUser

}
func fetchGroupIntegrationsData(companyId string, integration models.GroupIntegration, dataChannel chan<- models.GroupIntegration) {

	var integrationDetails models.GroupIntegration

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_INTEGRATIONS),
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_INTEGRATION),
					},
				},
			},
			"IntegrationID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(integration.IntegrationID),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		// e := errors.New(constants.HTTP_STATUS_500)
		// return integrations, e
	}
	if len(result.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(result.Items[0], &integrationDetails)
		if err != nil {
			// e := errors.New(constants.HTTP_STATUS_400)
			// return integrations, e
		}
	}

	integration.IntegrationName = integrationDetails.IntegrationName
	integration.IntegrationSlug = integrationDetails.IntegrationSlug
	integration.DisplayPhoto = integrationDetails.DisplayPhoto

	dataChannel <- integration
}

func fetchGroupIntegrationsDataNew(companyId string, integration models.GroupIntegration) *models.GroupIntegration {
	var integrationDetails models.GroupIntegration

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_INTEGRATIONS),
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_INTEGRATION),
					},
				},
			},
			"IntegrationID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(integration.IntegrationID),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		// e := errors.New(constants.HTTP_STATUS_500)
		// return integrations, e
	}
	if len(result.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(result.Items[0], &integrationDetails)
		if err != nil {
			// e := errors.New(constants.HTTP_STATUS_400)
			// return integrations, e
		}
	}

	integration.IntegrationName = integrationDetails.IntegrationName
	integration.IntegrationSlug = integrationDetails.IntegrationSlug
	integration.DisplayPhoto = integrationDetails.DisplayPhoto

	return &integration
}

/*
GetAll - GET - v1/groups/all
Params:
key - for searching groups
include - for including group memebers, integrations
lastEvaluatedKey - for pagination
limit - to limit number of items returned
*/

// @Summary Get Groups
// @Description This endpoint retrieves a list of all groups, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param last_evaluated_key query string false "Pagination key for fetching the next set of results"
// @Param key query string false "Search Key"
// @Param include query string false "Include related data: members or integrations"
// @Param limit query int false "Limit the number of results"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/all [get]
func (c GroupController) GetAll() revel.Result {

	//Parameters
	paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")
	searchKey := c.Params.Query.Get("key")
	include := c.Params.Query.Get("include")
	limit := c.Params.Query.Get("limit")

	//Make a data interface to return as JSON
	result := make(map[string]interface{})

	if searchKey != "" {
		searchKey = strings.ToLower(searchKey)
	}

	var pageLimit int64
	if limit != "" {
		pageLimit = utils.ToInt64(limit)
	} else {
		pageLimit = constants.DEFAULT_PAGE_LIMIT
	}

	//Pagination variables
	pageCount := 1
	totalItems := 0

	//Models
	// groups := []models.Group{}
	lastEvaluatedKey := models.Group{}

	var err error

	groups, lastEvaluatedKey, err := GetAllGroups(constants.ENTITY_TYPE_GROUP, pageLimit, paramLastEvaluatedKey, searchKey)

	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		result["error"] = err.Error()
		return c.RenderJSON(result)
	}

	if len(paramLastEvaluatedKey) != 0 {

		groups, lastEvaluatedKey, err = GetAllGroups(constants.ENTITY_TYPE_GROUP, pageLimit, paramLastEvaluatedKey, searchKey)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			result["error"] = err.Error()
			return c.RenderJSON(result)
		}

		if lastEvaluatedKey.GroupID == "" {
			lastEvaluatedKey = models.Group{}
		}
	}

	if len(include) != 0 {
		for i, group := range groups {
			if strings.Contains(include, "members") {
				// memberUsers, err := GetGroupMembers(group.GroupID, constants.MEMBER_TYPE_USER)
				// 	if err != nil {
				// 		result["status"] = utils.GetHTTPStatus(err.Error())
				// 	}
				// 	groups[i].GroupMembers = memberUsers
				memberUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_USER, c.Controller)
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				ownerUsers, err := GetGroupOwners(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_OWNER, c.Controller)
				if err != nil {
					c.Response.Status = 400
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				groups[i].GroupMembers = memberUsers
				groups[i].GroupOwners = ownerUsers

			}
			if strings.Contains(include, "integrations") {
				integrationsResult, err := GetGroupIntegrations(group.GroupID)
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
				}

				groups[i].GroupIntegrations = integrationsResult
			}
		}
	}

	//TODO use the token to fetch the user's role and add it to the return value

	totalItems = totalItems + len(groups)

	result["total_items"] = totalItems
	result["total_pages"] = pageCount
	result["lastEvaluatedKey"] = lastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["groups"] = groups

	return c.RenderJSON(result)
}

/*
GetGroupMembers function
Params:
groupID
*/
func GetGroupMembers(companyID, groupID, memberType string, c *revel.Controller, mainGroupID ...string) ([]models.GroupMember, error) {

	members := []models.GroupMember{}

	var sk string
	switch memberType {
	case constants.MEMBER_TYPE_USER:
		sk = constants.PREFIX_USER
	case constants.MEMBER_TYPE_GROUP:
		sk = constants.PREFIX_GROUP
	}

	status := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_DEFAULT, constants.ITEM_STATUS_PENDING}
	statusAv, err := dynamodbattribute.MarshalList(status)
	if err != nil {

	}

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
						S: aws.String(sk),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Status": {
				// ComparisonOperator: aws.String(constants.CONDITION_NOT_EQUAL),
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: statusAv,
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

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return members, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return members, e
	}
	for i, member := range members {
		// append user and group details

		switch member.MemberType {
		case constants.MEMBER_TYPE_USER:
			user, err := ops.GetUserByID(member.MemberID)
			companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
				UserID:    member.MemberID,
				CompanyID: companyID,
			}, c)
			if companyUser.FirstName == "" {
				companyUser.FirstName = user.FirstName
			}
			if companyUser.LastName == "" {
				companyUser.LastName = user.LastName
			}
			companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
			if err == nil {
				// append name, todo: display photo
				members[i].Name = companyUser.FirstName + " " + companyUser.LastName
				if len(mainGroupID) > 0 {

					isSubGroupMember := mainGroupID[0] == member.GroupID
					members[i].IsSubGroupMember = strconv.FormatBool(isSubGroupMember)
				}
			}
		case constants.MEMBER_TYPE_GROUP:
			group, err := GetGroupByID(member.MemberID)
			if err == nil {
				// append name and bg
				members[i].Name = group.GroupName
				members[i].Bg = group.GroupColor
				members[i].AssociatedAccounts = group.AssociatedAccounts
			}
		}
	}
	return members, nil
}

type GetGroupMembersOutput struct {
	Members GetGroupMembersObjectOutput `json:"members,omitempty"`
}
type GetGroupMembersObjectOutput struct {
	Users  []models.GroupMember `json:"Users,omitempty"`
	Groups []models.GroupMember `json:"Groups,omitempty"`
}

// @Summary Get Group Members
// @Description This endpoint retrieves a list of members associated with a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param includeSubGroup query string false "Includes sub groups"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/members [get]
func (c GroupController) GetGroupMembers(groupID string) revel.Result {
	group, err := GetGroupByID(groupID)
	companyID := c.ViewArgs["companyID"].(string)
	includeSubGroup := c.Params.Query.Get("includeSubGroup")
	if err != nil {

	}
	var includeSubGroupBool bool

	if includeSubGroup == "" || includeSubGroup == "false" {
		includeSubGroupBool = false
	} else {
		includeSubGroupBool = true
	}

	// return c.RenderJSON(group)
	if !includeSubGroupBool {
		memberUsers, err := GetGroupMembersNew(group.CompanyID, group.GroupID, constants.MEMBER_TYPE_USER)
		if err != nil {
			// TODO: Handle error
		}

		for idx, member := range memberUsers {
			user, err := ops.GetUserByID(member.MemberID)
			if err != nil {

			}
			companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
				UserID:    member.MemberID,
				CompanyID: group.CompanyID,
			}, c.Controller)
			if err != nil {

			}

			// user := getUserInTmp(member.MemberID)
			// companyUser := getCompanyMemberInTmp(group.CompanyID, member.MemberID, c.Controller)
			//This is to handle empty value for old accounts
			if companyUser.FirstName == "" {
				companyUser.FirstName = user.FirstName
				companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
			}
			if companyUser.LastName == "" {
				companyUser.LastName = user.LastName
				companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
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
			memberUsers[idx].MemberInformation = companyUser
			memberUsers[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)
		}

		memberGroups, err := GetGroupMembersNew(group.CompanyID, group.GroupID, constants.MEMBER_TYPE_GROUP)
		if err != nil {
			// TODO: Handle error
		}

		for idx, memberGroup := range memberGroups {
			g, gErr := ops.GetCompanyGroupNew(companyID, memberGroup.MemberID)
			if gErr == nil {
				memberGroups[idx].Name = g.GroupName
				memberGroups[idx].Bg = g.GroupColor
				memberGroups[idx].AssociatedAccounts = g.AssociatedAccounts

				subGroupIntegrations, err := ops.GetGroupIntegrationsNew(g.GroupID, companyID, true)
				if err == nil {
					memberGroups[idx].GroupIntegrations = subGroupIntegrations
				}
			}
		}

		return c.RenderJSON(GetGroupMembersOutput{
			Members: GetGroupMembersObjectOutput{
				Users:  memberUsers,
				Groups: memberGroups,
			},
		})
	} else {
		memberUsers, err := ops.GetGroupMembersOld(groupID, c.Controller)
		if err != nil {
			// TODO: Handle error
		}

		for idx, member := range memberUsers {
			// user, err := ops.GetUserByID(member.MemberID)
			// if err != nil {

			// }
			// companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			// 	UserID:    member.MemberID,
			// 	CompanyID: group.CompanyID,
			// }, c.Controller)
			// if err != nil {

			// }

			user := getUserInTmp(member.MemberID)
			companyUser := getCompanyMemberInTmp(group.CompanyID, member.MemberID, c.Controller)
			//This is to handle empty value for old accounts
			if companyUser.FirstName == "" {
				companyUser.FirstName = user.FirstName
				companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
			}
			if companyUser.LastName == "" {
				companyUser.LastName = user.LastName
				companyUser.SearchKey = companyUser.FirstName + " " + companyUser.LastName + " " + user.Email
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
			memberUsers[idx].MemberInformation = companyUser
			memberUsers[idx].Name = utils.GenerateFullname(companyUser.FirstName, companyUser.LastName)
		}

		return c.RenderJSON(GetGroupMembersOutput{
			Members: GetGroupMembersObjectOutput{
				Users: memberUsers,
			},
		})

	}
}

// Refactored version of GetGroupMember()
func GetGroupMembersNew(companyID, groupID, memberType string, mainGroupID ...string) ([]models.GroupMember, error) {

	members := []models.GroupMember{}

	var sk string
	switch memberType {
	case constants.MEMBER_TYPE_USER:
		sk = constants.PREFIX_USER
	case constants.MEMBER_TYPE_GROUP:
		sk = constants.PREFIX_GROUP
	case constants.MEMBER_TYPE_OWNER:
		sk = constants.PREFIX_OWNER
	}

	status := []string{constants.ITEM_STATUS_ACTIVE, constants.ITEM_STATUS_DEFAULT, constants.ITEM_STATUS_PENDING}
	statusAv, err := dynamodbattribute.MarshalList(status)
	if err != nil {

	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
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
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_IN),
				AttributeValueList: statusAv,
			},
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(sk),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return members, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return members, e
	}

	// for i, member := range members {
	// 	// append user and group details

	// 	switch member.MemberType {
	// 	case constants.MEMBER_TYPE_USER:
	// 		user, err := ops.GetUserByID(member.MemberID)
	// 		if err == nil {
	// 			// append name, todo: display photo
	// 			members[i].Name = user.FirstName + " " + user.LastName
	// 			if len(mainGroupID) > 0 {

	// 				isSubGroupMember := mainGroupID[0] == member.GroupID
	// 				members[i].IsSubGroupMember = strconv.FormatBool(isSubGroupMember)
	// 			}
	// 		}
	// 	case constants.MEMBER_TYPE_GROUP:
	// 		group, err := GetGroupByID(member.MemberID)
	// 		if err == nil {
	// 			// append name and bg
	// 			members[i].Name = group.GroupName
	// 			members[i].Bg = group.GroupColor
	// 			members[i].AssociatedAccounts = group.AssociatedAccounts
	// 		}
	// 	}
	// }
	return members, nil
}

/*
GetOwnerMembers function
Params:
groupID
*/
func GetGroupOwners(companyID, groupID, memberType string, c *revel.Controller, mainGroupID ...string) ([]models.GroupMember, error) {

	members := []models.GroupMember{}

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
						S: aws.String(constants.PREFIX_OWNER),
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
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return members, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return members, e
	}
	for i, member := range members {
		// append user and group details
		user, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    member.MemberID,
			CompanyID: companyID,
		}, c)
		if err == nil {
			members[i].Name = user.FirstName + " " + user.LastName
			if len(mainGroupID) > 0 {
				isSubGroupMember := mainGroupID[0] == member.GroupID
				members[i].IsSubGroupMember = strconv.FormatBool(isSubGroupMember)
			}
			gcm := ops.GetCompanyMemberParams{UserID: member.MemberID, CompanyID: companyID}
			companyMember, err := ops.GetCompanyMember(gcm, c)
			if err == nil {
				members[i].Status = companyMember.Status
			}
		}
	}
	return members, nil
}

/*
GroupsJoin
groupID
*/
func GroupsJoined(groupID, memberType string) ([]models.GroupMember, error) {

	members := []models.GroupMember{}

	var sk string
	switch memberType {
	case constants.MEMBER_TYPE_USER:
		sk = constants.PREFIX_USER
	case constants.MEMBER_TYPE_GROUP:
		sk = constants.PREFIX_GROUP
	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
					},
				},
			},
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(sk),
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
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return members, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return members, e
	}

	return members, nil
}

// Refactored version of GetGroupIntegrations()
func GetGroupIntegrationsNew(groupID, companyID string, isSubIntegrationInclude bool) ([]models.GroupIntegration, error) {

	integrations := []models.GroupIntegration{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
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
						S: aws.String(constants.PREFIX_INTEGRATION),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return integrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &integrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return integrations, e
	}
	for i, integration := range integrations {
		var integrationDetails models.GroupIntegration
		var integrationCache models.GroupIntegration
		if err == nil {
			// if errCache := cache.Get("groupIntegration_"+integration.IntegrationID, &integrationCache); errCache != nil {
			params := &dynamodb.QueryInput{
				TableName: aws.String(app.TABLE_NAME),
				IndexName: aws.String(constants.INDEX_NAME_GET_INTEGRATIONS),
				KeyConditions: map[string]*dynamodb.Condition{
					"Type": {
						ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String(constants.ENTITY_TYPE_INTEGRATION),
							},
						},
					},
					"IntegrationID": {
						ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
						AttributeValueList: []*dynamodb.AttributeValue{
							{
								S: aws.String(integration.IntegrationID),
							},
						},
					},
				},
			}

			result, err := ops.HandleQueryLimit(params)
			if err != nil {
				e := errors.New(constants.HTTP_STATUS_500)
				return integrations, e
			}
			// params := &dynamodb.QueryInput{
			// 	TableName: aws.String(app.TABLE_NAME),
			// 	KeyConditions: map[string]*dynamodb.Condition{
			// 		"PK": {
			// 			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			// 			AttributeValueList: []*dynamodb.AttributeValue{
			// 				{
			// 					S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integration.IntegrationID)),
			// 				},
			// 			},
			// 		},
			// 		"SK": {
			// 			ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
			// 			AttributeValueList: []*dynamodb.AttributeValue{
			// 				{
			// 					S: aws.String(constants.PREFIX_INTEGRATION),
			// 				},
			// 			},
			// 		},
			// 	},
			// }

			// result, err := app.SVC.Query(params)
			// if err != nil {
			// 	e := errors.New(constants.HTTP_STATUS_500)
			// 	return integrations, e
			// }

			//
			if len(result.Items) != 0 {
				err = dynamodbattribute.UnmarshalMap(result.Items[0], &integrationDetails)
				if err != nil {
					e := errors.New(constants.HTTP_STATUS_400)
					return integrations, e
				}
			}
			integrations[i].IntegrationName = integrationDetails.IntegrationName
			integrations[i].IntegrationSlug = integrationDetails.IntegrationSlug
			integrations[i].DisplayPhoto = integrationDetails.DisplayPhoto
			// go cache.Set("groupIntegration_"+integration.IntegrationID, integrations[i], 30*time.Minute)
			// }
		}
		if integrationCache.IntegrationName != "" {
			integrations[i] = integrationCache
		}

		if isSubIntegrationInclude {
			subGroupIntegration, err := ops.GetGroupSubIntegrationWithIntegrationID(groupID, integration.IntegrationID)
			if err != nil {
			}
			integrations[i].GroupSubIntegrations = subGroupIntegration
		}
	}

	return integrations, nil
}

// Refactored version of GetGroupIntegrations()
func GetGroupIntegrationsWithRoutines(groupID, companyID string) ([]models.GroupIntegration, error) {

	integrations := []models.GroupIntegration{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
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
						S: aws.String(constants.PREFIX_INTEGRATION),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return integrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &integrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return integrations, e
	}
	// dataChannel := make(chan models.GroupIntegration)

	for i, integration := range integrations {
		// go fetchGroupIntegrationsData(companyID, integration, dataChannel)
		integ := fetchGroupIntegrationsDataNew(companyID, integration)
		if integ != nil {
			integrations[i] = *integ
		}
	}
	// for i := 0; i < len(integrations); i++ {
	// 	integrations[i] = <-dataChannel
	// }
	// for i, integration := range integrations{
	// }

	return integrations, nil
}

/*
GetIntegrations function
Params:
groupID
*/
func GetGroupIntegrations(groupID string) ([]models.GroupIntegration, error) {

	integrations := []models.GroupIntegration{}

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
						S: aws.String(constants.PREFIX_INTEGRATION),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return integrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &integrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return integrations, e
	}

	for i, integration := range integrations {
		var integrationDetails models.GroupIntegration
		params := &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			KeyConditions: map[string]*dynamodb.Condition{
				"PK": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integration.IntegrationID)),
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

		result, err := app.SVC.Query(params)
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_500)
			return integrations, e
		}

		//
		if len(result.Items) != 0 {
			err = dynamodbattribute.UnmarshalMap(result.Items[0], &integrationDetails)
			if err != nil {
				e := errors.New(constants.HTTP_STATUS_400)
				return integrations, e
			}
		}

		//
		integrations[i].IntegrationName = integrationDetails.IntegrationName
		integrations[i].IntegrationSlug = integrationDetails.IntegrationSlug
		integrations[i].DisplayPhoto = integrationDetails.DisplayPhoto
	}

	return integrations, nil
}

/*
GetSubIntegrations function
Params:
groupID
*/
func GetGroupSubIntegration(groupID string) ([]models.GroupSubIntegration, error) {

	subIntegrations := []models.GroupSubIntegration{}

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
						S: aws.String(constants.PREFIX_SUB_INTEGRATION),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return subIntegrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &subIntegrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return subIntegrations, e
	}

	return subIntegrations, nil
}

/*
CreateGroup Add Group Web Service
route: /v1/groups
method: POST
params:
department_id - must be exists, required
group_name - required
group_description - optional
member_users <[]GroupMember>- array of users id and role, eg: [{"MemberID": "<id>", "GroupRole": "<role>"}]
member_groups <[]string>- array of group ids, eg: ["<id>"]
*/

// @Summary Create Group
// @Description This endpoint creates a new group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.CreateGroupRequest true "Create group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups [post]
func (c GroupController) CreateGroup() revel.Result {
	result := make(map[string]interface{})

	var memberOwner []string
	var memberGroups []string
	var memberIndividuals []string
	c.Params.Bind(&memberGroups, "group_members")
	c.Params.Bind(&memberOwner, "member_owner")
	c.Params.Bind(&memberIndividuals, "individual_members")
	companyID := c.Params.Form.Get("company_id")
	groupName := strings.TrimSpace(c.Params.Form.Get("group_name"))
	groupEmail := strings.TrimSpace(c.Params.Form.Get("group_email"))
	departmentID := c.Params.Form.Get("department_id")
	groupDescription := c.Params.Form.Get("group_description")
	integrationID := c.Params.Form.Get("integration_id")

	deptExists := CheckIfDepartmentExists(departmentID, c.Validation)
	if !deptExists {
		createDept := ops.CreateDepartmentPayload{
			CompanyID:   companyID,
			Name:        departmentID,
			Description: "",
		}
		department, deptErr := ops.CreateDepartment(createDept, c.Validation)
		if deptErr != nil {
			c.Response.Status = deptErr.HTTPStatusCode
			return c.RenderJSON(deptErr)
		}
		departmentID = department.DepartmentID
	}

	groupUUID := utils.GenerateTimestampWithUID()
	userID := c.ViewArgs["userID"].(string)

	var createGWGroup bool
	c.Params.Bind(&createGWGroup, "create_gw_group")

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.ADD_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	// items, err := GetGroupsByGroupName(groupName, c.ViewArgs["companyID"].(string))
	// if err != nil {
	// 	result["message"] = "Something went wrong with getting groups"
	// 	result["status"] = utils.GetHTTPStatus(err.Error())
	// 	return c.RenderJSON(result)
	// }

	// err = dynamodbattribute.UnmarshalListOfMaps(items, &groups)
	// if err != nil {
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(result)
	// }

	// var groupsNameList []string
	// for _, group := range groups {
	// 	groupsNameList = append(groupsNameList, group.GroupName)
	// }

	groupsNameList, err := ops.GetGroupsByCompany(companyID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// Separated method for generating a unique group name to avoid conflict
	newUniqueGroupName := utils.MakeGroupNameUnique(groupName, groupsNameList)

	//PREPARE GROUP DATA
	group := models.Group{
		PK:               utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
		SK:               utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
		GSI_SK:           utils.AppendPrefix(constants.PREFIX_GROUP, strings.ToLower(groupName)),
		GroupID:          groupUUID,
		DepartmentID:     departmentID,
		CompanyID:        companyID,
		GroupName:        newUniqueGroupName,
		GroupDescription: groupDescription,
		GroupColor:       utils.GetRandomColor(),
		Status:           constants.ITEM_STATUS_ACTIVE,
		Type:             constants.ENTITY_TYPE_GROUP,
		NewGroup:         constants.BOOL_TRUE,
		CreatedAt:        utils.GetCurrentTimestamp(),
		SearchKey:        strings.ToLower(groupName),
	}

	// VALIDATING GROUP FORM
	group.Validate(c.Validation, constants.USER_PERMISSION_ADD_GROUP)
	if c.Validation.HasErrors() {
		result["errors"] = c.Validation.Errors
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}
	if groupEmail != "" {
		group.AssociatedAccounts = map[string][]string{
			"google": []string{groupEmail},
		}
	}
	// CHECK IF GROUPNAME IS UNIQUE
	// uniqueGroup := IsGroupNameUnique(group.CompanyID, group.SearchKey)
	// if uniqueGroup {
	// 	result["message"] = "Group name already exists."
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_490)
	// 	return c.RenderJSON(result)
	// }

	//MARSHAL GROUP DATA
	groupData, err := dynamodbattribute.MarshalMap(group)
	if err != nil {
		result["message"] = "Got error marshalling group"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	//PREPARE FOR INSERT TO AWS
	groupInput := &dynamodb.PutItemInput{
		Item:      groupData,
		TableName: aws.String(app.TABLE_NAME),
	}

	//INSERTING TO AWS
	_, err = app.SVC.PutItem(groupInput)
	if err != nil {
		result["message"] = "Error while saving"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	//PREPARING GROUP MEMBERS
	var members []models.GroupMember
	var groupMembers []models.GroupMember
	var inputRequest []*dynamodb.WriteRequest
	var membersInput *dynamodb.BatchWriteItemInput

	var departmentMembers []models.DepartmentMember

	// PREPARE DEPARTMENT MEMBERS
	for _, item := range memberOwner {
		members = append(members, models.GroupMember{
			PK:         utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
			SK:         utils.AppendPrefix(constants.PREFIX_OWNER, item),
			GroupID:    groupUUID,
			MemberID:   item,
			MemberType: constants.MEMBER_TYPE_OWNER,
			Status:     constants.ITEM_STATUS_ACTIVE,
			MemberRole: constants.MEMBER_TYPE_OWNER,
			CreatedAt:  utils.GetCurrentTimestamp(),
			UpdatedAt:  utils.GetCurrentTimestamp(),
			Type:       constants.ENTITY_TYPE_GROUP_OWNER,
		})

		departmentMembers = append(departmentMembers, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, item),
			DepartmentID: departmentID,
			UserID:       item,
		})
	}
	if len(memberIndividuals) > 0 {
		//PREPARING GROUP MEMBERS
		for _, item := range memberIndividuals {
			members = append(members, models.GroupMember{
				PK:         utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
				SK:         utils.AppendPrefix(constants.PREFIX_USER, item),
				CompanyID:  companyID,
				GroupID:    groupUUID,
				MemberID:   item,
				MemberType: constants.MEMBER_TYPE_USER,
				Status:     constants.ITEM_STATUS_ACTIVE,
				MemberRole: constants.MEMBER_TYPE_USER,
				CreatedAt:  utils.GetCurrentTimestamp(),
				UpdatedAt:  utils.GetCurrentTimestamp(),
				Type:       constants.ENTITY_TYPE_GROUP_MEMBER,
			})

			departmentMembers = append(departmentMembers, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, item),
				DepartmentID: departmentID,
				UserID:       item,
			})
		}
	}

	//COMPILING INFORMATION FOR INSERTION
	for _, member := range members {
		inputRequest = append(inputRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": &dynamodb.AttributeValue{
					S: aws.String(member.PK),
				},
				"SK": &dynamodb.AttributeValue{
					S: aws.String(member.SK),
				},
				"CompanyID": &dynamodb.AttributeValue{
					S: aws.String(companyID),
				},
				"GroupID": &dynamodb.AttributeValue{
					S: aws.String(member.GroupID),
				},
				"MemberID": &dynamodb.AttributeValue{
					S: aws.String(member.MemberID),
				},
				"Name": &dynamodb.AttributeValue{
					S: aws.String(member.Name),
				},
				"MemberType": &dynamodb.AttributeValue{
					S: aws.String(member.MemberType),
				},
				"Status": &dynamodb.AttributeValue{
					S: aws.String(member.Status),
				},
				"MemberRole": &dynamodb.AttributeValue{
					S: aws.String(member.MemberRole),
				},
				"CreatedAt": &dynamodb.AttributeValue{
					S: aws.String(member.CreatedAt),
				},
				"UpdatedAt": &dynamodb.AttributeValue{
					S: aws.String(member.UpdatedAt),
				},
				"Type": &dynamodb.AttributeValue{
					S: aws.String(member.Type),
				},
			},
		}})
	}

	if len(memberGroups) > 0 {
		for _, item := range memberGroups {
			group, errGetGroup := GetGroupByID(item)
			if errGetGroup != nil {
				result["status"] = constants.HTTP_STATUS[errGetGroup.Error()]
				return c.RenderJSON(result)
			}

			groupMembers = append(groupMembers, models.GroupMember{
				PK:           utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
				SK:           utils.AppendPrefix(constants.PREFIX_GROUP, item),
				GroupID:      groupUUID,
				GroupName:    group.GroupName,
				CompanyID:    companyID,
				MemberID:     item,
				DepartmentID: departmentID,
				Status:       constants.ITEM_STATUS_ACTIVE,
				MemberType:   constants.MEMBER_TYPE_GROUP,
				CreatedAt:    utils.GetCurrentTimestamp(),
				UpdatedAt:    utils.GetCurrentTimestamp(),
				Type:         constants.ENTITY_TYPE_GROUP_MEMBER,
			})
		}
		for _, member := range groupMembers {
			inputRequest = append(inputRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(member.PK),
					},
					"SK": {
						S: aws.String(member.SK),
					},
					"GroupID": {
						S: aws.String(member.GroupID),
					},
					"CompanyID": {
						S: aws.String(member.CompanyID),
					},
					"GroupName": {
						S: aws.String(member.GroupName),
					},
					"MemberID": {
						S: aws.String(member.MemberID),
					},
					"Status": {
						S: aws.String(member.Status),
					},
					"MemberType": {
						S: aws.String(member.MemberType),
					},
					"CreatedAt": {
						S: aws.String(member.CreatedAt),
					},
					"UpdatedAt": {
						S: aws.String(member.UpdatedAt),
					},
					"Type": {
						S: aws.String(member.Type),
					},
				},
			}})

			departmentMembers = append(departmentMembers, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, member.MemberID),
				DepartmentID: departmentID,
				UserID:       member.MemberID,
			})
		}
	}

	//SPECIFYING TABLE NAME
	membersInput = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			app.TABLE_NAME: inputRequest,
		},
	}

	//INSERTING DATA
	_, err = app.SVC.BatchWriteItem(membersInput)
	//ERROR AT INSERTING
	if err != nil {
		result["message"] = "Got error in put item (MEMBERS)"
		result["error"] = err.Error()
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// INSERT DEPARTMENT MEMBERS START (NEW 02-21-2022)
	// UPDATED 04-5-2022
	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptUserAddErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptUserAddErr != nil {
		result["departmentError"] = "Error adding users to department."
	}
	// INSERT DEPARTMENT MEMBERS END (NEW 02-21-2022)
	// UPDATED 04-5-2022
	// TODO refactor this web service

	//data
	//new group
	newGroup := models.Group{}
	//fetch groups
	fetch, err := ops.GetGroupByID(groupUUID)
	if err != nil {
		result["message"] = "Something went wrong with group"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(fetch)
	}

	//is no data in fetch
	if len(fetch.Items) == 0 {
		result["message"] = "No Data"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	//binding data to &group
	err = dynamodbattribute.UnmarshalMap(fetch.Items[0], &newGroup)
	if err != nil {
		result["message"] = "Something went wrong with unmarshalmap"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	//get group user members
	memberUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), groupUUID, constants.MEMBER_TYPE_USER, c.Controller, groupUUID)
	if err != nil {
		result["message"] = "Something went wrong with group members"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	for idx, member := range memberUsers {
		//fecthing sub group user information
		// user, opsError := ops.GetUserByID(member.MemberID)
		user, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    member.MemberID,
			CompanyID: companyID,
		}, c.Controller)
		if opsError != nil {
			result["message"] = "Something went wrong with group member information"
			result["status"] = utils.GetHTTPStatus(opsError.Status.Code)
			return c.RenderJSON(result)
		}

		userUID, err := ops.GetIntegrationUID(member.MemberID)
		if err != nil {
			result["message"] = "Something went wrong with group member information"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}

		memberUsers[idx].IntegrationUID = userUID.IntegrationUID

		memberUsers[idx].MemberInformation = user
	}

	newGroup.GroupMembers = memberUsers

	/*********** added 07-01-2022
	START - Fetching memmbers added from subgroups and appending them to the variable "memmber"
	********/
	//1. GET SUB GROUP MEMBER
	subGroups, err := GetGroupMembers(companyID, groupUUID, constants.MEMBER_TYPE_GROUP, c.Controller)
	if err != nil {
		result["message"] = "Something went wrong with group members"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	//1.2 FETCHED LIST OF SUBGROUPS OF THE CREATED GROUP
	newSubList := subGroups

	//1.3 INITIAL COUNT OF SUBGROUPS THAT WERE FETCHED
	subGroupCount := 0

	for {

		//2. LOOP THE FETCHED SUBGROUPS
		for _, data := range newSubList {

			//2.2 FETCH SUB GROUP OF THE CURRENT SUBGROUP
			subOfSubGroups, err := GetGroupMembers(companyID, data.MemberID, constants.MEMBER_TYPE_GROUP, c.Controller)
			if err != nil {
				result["message"] = "Something went wrong with group members"
				result["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(result)
			}

			//2.3 ITERATE TO SUBGROUP OF SUBGROUP
			for _, subData := range subOfSubGroups {

				flag := false

				//2.4 ITERATE THE INITIAL LIST OF SUBGROUPS
				for _, clonedSubList := range newSubList {
					//2.5 CHECK IF SUBGROUP OF SUBGROUP IS EXISTING IN THE LIST
					if clonedSubList.MemberID == subData.MemberID || groupUUID == subData.MemberID {
						flag = true
						break
					}
				}

				//2.6 INSERT SUBGROUP OF SUBGROUP IF IT DOES NOT EXIST ON INITIAL LIST OF SUBGROUPS ON STEP 1.2
				if !flag {
					newSubList = append(newSubList, subData)
				}
			}
		}

		//3. STOP LOOP IF EQUAL
		if subGroupCount == len(newSubList) {
			break
		}

		//4. APPEND NEW COUNT
		subGroupCount = len(newSubList)
	}

	//5. LOOP THE UPDATED SUBGROUPS TO GET THE GROUP ID
	for _, getSubGroupID := range newSubList {
		// 5.3 FETCH THE MEMBERS
		subGroupMembers, err := GetGroupMembers(companyID, getSubGroupID.MemberID, constants.MEMBER_TYPE_USER, c.Controller, groupUUID)
		if err != nil {
			result["message"] = "Something went wrong with group members"
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		// 5.2 LOOP TO MEMBERS
		for _, subGroupMems := range subGroupMembers {
			// 5.3 CHECK IF MEMBER IS EXISTING IN THE MAIN GROUP
			flag := false
			for _, checkMember := range memberUsers {
				if checkMember.MemberID == subGroupMems.MemberID {
					flag = true
					break
				}
			}

			if !flag {
				//INSERT MEMBERS FROM SUBGROUPS IF IT DOES NOT EXIST ON INITIAL LIST OF MEMBERS
				members = append(members, subGroupMems)
			}
		}
	}

	// append members in group
	group.GroupMembers = members
	//* If create group connect to integration
	if integrationID != "" {
		subIntegrations, err := ops.GetIntegrationSubIntegrations(integrationID)
		if err == nil {
			// todo handle error
			googleAdminIntegration := GetGoogleAdminSubIntegrationInList(subIntegrations)

			connectInput := ConnectGroupsToGoogleAdminInput{
				Groups: []GWGroup{
					{
						GroupName:    groupName,
						GroupKey:     groupEmail,
						GroupID:      group.GroupID,
						DepartmentID: group.DepartmentID,
					},
				},
				GoogleAdmin:   googleAdminIntegration,
				IntegrationID: integrationID,
				CompanyID:     companyID,
			}
			err := ConnectGroupsToGoogleAdmin(connectInput)
			if err != nil {
				// continue
				// todo error
			}
		}
	}
	/***********
	END
	*********/

	// //PREPARING INTEGRATION DATA
	// if len(integrationID) > 0 {
	// 	var integrations []models.GroupIntegration
	// 	var integrationRequest []*dynamodb.WriteRequest
	// 	var integrationInput *dynamodb.BatchWriteItemInput

	// 	for index, item := range integrationID {
	// 		integrations = append(integrations, models.GroupIntegration{
	// 			PK:              utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
	// 			SK:              utils.AppendPrefix(constants.PREFIX_INTEGRATION, item),
	// 			GroupID:         groupUUID,
	// 			IntegrationID:   item,
	// 			IntegrationName: integrationName[index],
	// 			DisplayPhoto:    integrationAvatar[index],
	// 			CreatedAt:       utils.GetCurrentTimestamp(),
	// 			UpdatedAt:       utils.GetCurrentTimestamp(),
	// 			Type:            constants.ENTITY_TYPE_GROUP_INTEGRATION,
	// 		})
	// 	}

	// 	//COMPILING INFORMATION FOR INSERTION
	// 	for _, integration := range integrations {
	// 		integrationRequest = append(integrationRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
	// 			Item: map[string]*dynamodb.AttributeValue{
	// 				"PK": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.PK),
	// 				},
	// 				"SK": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.SK),
	// 				},
	// 				"GroupID": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.GroupID),
	// 				},
	// 				"IntegrationID": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.IntegrationID),
	// 				},
	// 				"IntegrationName": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.IntegrationName),
	// 				},
	// 				"DisplayPhoto": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.DisplayPhoto),
	// 				},
	// 				"CreatedAt": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.CreatedAt),
	// 				},
	// 				"UpdatedAt": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.UpdatedAt),
	// 				},
	// 				"Type": &dynamodb.AttributeValue{
	// 					S: aws.String(integration.Type),
	// 				},
	// 			},
	// 		}})

	// 		if len(subID) > 0 {

	// 			subIntegrationsResponse, err := GetGroupSubIntegration(integration.IntegrationID)
	// 			if err != nil {
	// 				result["message"] = "Error while fetching sub integrations"
	// 				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 				return c.RenderJSON(result)
	// 			}

	// 			var subIntegrations []models.GroupSubIntegration
	// 			var subIntegrationRequest []*dynamodb.WriteRequest
	// 			var subIntegrationInput *dynamodb.BatchWriteItemInput

	// 			for _, subIntegrationResponse := range subIntegrationsResponse {
	// 				subIntegrations = append(subIntegrations, models.GroupSubIntegration{
	// 					PK:                  utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
	// 					SK:                  utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, subIntegrationResponse.IntegrationID),
	// 					GroupID:             groupUUID,
	// 					IntegrationID:       subIntegrationResponse.IntegrationID,
	// 					IntegrationName:     subIntegrationResponse.IntegrationName,
	// 					DisplayPhoto:        subIntegrationResponse.DisplayPhoto,
	// 					ParentIntegrationID: subIntegrationResponse.ParentIntegrationID,
	// 					IntegrationSlug:	 subIntegrationResponse.IntegrationSlug,
	// 					CreatedAt:           utils.GetCurrentTimestamp(),
	// 					UpdatedAt:           utils.GetCurrentTimestamp(),
	// 					Type:                constants.ENTITY_TYPE_GROUP_SUB_INTEGRATION,
	// 				})
	// 			}

	// 			//COMPILING INFORMATION FOR INSERTION
	// 			for _, subIntegration := range subIntegrations {
	// 				subIntegrationRequest = append(subIntegrationRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
	// 					Item: map[string]*dynamodb.AttributeValue{
	// 						"PK": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.PK),
	// 						},
	// 						"SK": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.SK),
	// 						},
	// 						"GroupID": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.GroupID),
	// 						},
	// 						"IntegrationID": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.IntegrationID),
	// 						},
	// 						"IntegrationName": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.IntegrationName),
	// 						},
	// 						"IntegrationSlug": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.IntegrationSlug),
	// 						},
	// 						"ParentIntegrationID": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.ParentIntegrationID),
	// 						},
	// 						"DisplayPhoto": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.DisplayPhoto),
	// 						},
	// 						"CreatedAt": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.CreatedAt),
	// 						},
	// 						"UpdatedAt": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.UpdatedAt),
	// 						},
	// 						"Type": &dynamodb.AttributeValue{
	// 							S: aws.String(subIntegration.Type),
	// 						},
	// 					},
	// 				}})
	// 			}

	// 			//SPECIFYING TABLE NAME
	// 			subIntegrationInput = &dynamodb.BatchWriteItemInput{
	// 				RequestItems: map[string][]*dynamodb.WriteRequest{
	// 					app.TABLE_NAME: subIntegrationRequest,
	// 				},
	// 			}

	// 			//INSERTING DATA
	// 			_, err = app.SVC.BatchWriteItem(subIntegrationInput)
	// 			//ERROR AT INSERTING
	// 			if err != nil {
	// 				result["message"] = "Error while saving. Please add sub integration to save"
	// 				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 				return c.RenderJSON(result)
	// 			}
	// 		}
	// 	}

	// 	//SPECIFYING TABLE NAME
	// 	integrationInput = &dynamodb.BatchWriteItemInput{
	// 		RequestItems: map[string][]*dynamodb.WriteRequest{
	// 			app.TABLE_NAME: integrationRequest,
	// 		},
	// 	}

	// 	//INSERTING DATA
	// 	_, err = app.SVC.BatchWriteItem(integrationInput)

	// 	//ERROR AT INSERTING
	// 	if err != nil {
	// 		result["err"] = err
	// 		result["message"] = "Error while saving. Please add integration to save"
	// 		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 		return c.RenderJSON(result)
	// 	}
	// }

	// create group to google workspace
	// if createGWGroup {
	// 	gwGroupEmail := groupEmail
	// 	if gwGroupEmail == "" {
	// 		gwGroupEmail = utils.NameToUsername(newUniqueGroupName)
	// 	}
	// 	gwGroup := admin.Group{
	// 		Name: newUniqueGroupName,
	// 		Email: gwGroupEmail,
	// 		Description: groupDescription,
	// 	}
	// 	createGWGroupInput := googlefunctions.CreateGWGroupInput{
	// 		Group: gwGroup,
	// 	}
	// 	createGWGroupOutput, err := googlefunctions.CreateGWGroup()
	// }
	_ = groupEmail

	// generate log
	var logs []models.Logs
	// message: John Smith created GroupX
	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: group.GroupID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
			Department: &models.LogModuleParams{
				ID: departmentID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["logs"] = "error while creating logs"
	}

	// generate log
	// message: John Smith and John Doe has been added to GroupX
	logSearchKey := group.GroupName + " "
	var companyLogInfoUsers []models.LogModuleParams
	var groupLogInfoUsers []models.LogModuleParams
	for _, member := range members {
		if member.MemberType != constants.MEMBER_TYPE_OWNER {
			u := getUserInTmp(member.MemberID)
			logSearchKey += u.FirstName + " " + u.LastName + " "
			companyLogInfoUsers = append(companyLogInfoUsers, models.LogModuleParams{
				ID: member.MemberID,
			})

			groupLogInfoUsers = append(groupLogInfoUsers, models.LogModuleParams{
				ID: member.MemberID,
			})
		}
	}

	var errorMessages []string

	groupList, err := GetGroupList(companyID)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}
	output := len(groupList)

	if len(companyLogInfoUsers) != 0 {
		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: companyLogInfoUsers,
				Group: &models.LogModuleParams{
					ID: group.GroupID,
				},
			},
		}

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating company logs")
		}
		var actionItem models.ActionItem

		if output >= 2 {
			actionItem = models.ActionItem{
				CompanyID:      companyID,
				LogID:          companyLogID,
				ActionItemType: "ADD_USERS_TO_GROUP",
				SearchKey:      logSearchKey,
			}
		}
		// generate action items

		actionItemID, err := ops.CreateActionItem(actionItem)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating action items")
		}
		_ = actionItemID
	}

	if len(groupLogInfoUsers) != 0 {
		groupLog := models.Logs{
			GroupID:   group.GroupID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: groupLogInfoUsers,
				Group: &models.LogModuleParams{
					ID: group.GroupID,
				},
			},
		}

		groupLogID, err := ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating group logs")
		}
		_ = groupLogID
	}

	if len(errorMessages) != 0 {
		result["errorMessages"] = errorMessages
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	// result["actionItemID"] = actionItemID
	result["groupLogInfoUsers"] = groupLogInfoUsers
	result["group"] = group
	result["groups"] = newGroup
	result["message"] = "Group created"
	return c.RenderJSON(result)
}

type GWGroup struct {
	GroupName    string `json:"group_name"`
	GroupKey     string `json:"group_key"`
	GroupID      string
	DepartmentID string
}
type ConnectGroupsToGoogleAdminInput struct {
	Groups        []GWGroup
	GoogleAdmin   models.Integration
	IntegrationID string
	CompanyID     string
}

func ConnectGroupsToGoogleAdmin(input ConnectGroupsToGoogleAdminInput) error {
	if len(input.Groups) == 0 {
		return nil
	}

	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for _, group := range input.Groups {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, input.IntegrationID)),
				},
				"GroupID": {
					S: aws.String(group.GroupID),
				},
				"IntegrationID": {
					S: aws.String(input.IntegrationID),
				},
				"CompanyID": {
					S: aws.String(input.CompanyID),
				},
			},
		}})

		av, err := dynamodbattribute.MarshalList([]string{group.GroupKey})
		if err != nil {
			// return err
			continue
		}

		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, input.GoogleAdmin.IntegrationID)),
				},
				"GroupID": {
					S: aws.String(group.GroupID),
				},
				"IntegrationID": {
					S: aws.String(input.GoogleAdmin.IntegrationID),
				},
				"ParentIntegrationID": {
					S: aws.String(input.GoogleAdmin.ParentIntegrationID),
				},
				"IntegrationSlug": {
					S: aws.String(input.GoogleAdmin.IntegrationSlug),
				},
				"ConnectedItems": {
					L: av,
				},
				"CompanyID": {
					S: aws.String(input.CompanyID),
				},
			},
		}})

		if len(currentBatch)%constants.BATCH_LIMIT == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
		batches = append(batches, currentBatch)
	}

	batchOutput, err := ops.NewBatchWriteItemHandler(batches)
	if err != nil {
		return err
	}
	_ = batchOutput

	return nil
}

func GetGoogleAdminSubIntegrationInList(subIntegrations []models.Integration) models.Integration {
	var integration models.Integration
	for _, sub := range subIntegrations {
		if sub.IntegrationSlug == constants.INTEG_SLUG_GOOGLE_ADMIN {
			integration = sub
			break
		}
	}
	return integration
}

/*
AddIntegrationToGroup
*/

// @Summary Add Integrations
// @Description This endpoint is used to create integrations for a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddIntegrationsRequest true "Add integrations body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/integrations/add [post]
func (c GroupController) AddIntegrations() revel.Result {
	result := make(map[string]interface{})

	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	integrationID := c.Params.Form.Get("integration_id")
	integrationName := c.Params.Form.Get("integration_name")
	integrationAvatar := c.Params.Form.Get("integration_avatar")
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.ADD_GROUP_INTEGRATION, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	if groupID == "" || integrationID == "" || integrationName == "" || integrationAvatar == "" {
		result["message"] = "missing parameters!"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	//PREPARING INTEGRATION DATA
	integrations := models.GroupIntegration{
		PK:              utils.AppendPrefix(constants.PREFIX_GROUP, groupID),
		SK:              utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationID),
		GroupID:         groupID,
		IntegrationID:   integrationID,
		IntegrationName: integrationName,
		DisplayPhoto:    integrationAvatar,
		CreatedAt:       utils.GetCurrentTimestamp(),
		UpdatedAt:       utils.GetCurrentTimestamp(),
		Type:            constants.ENTITY_TYPE_GROUP_INTEGRATION,
	}

	//MARSHAL GROUP DATA
	integrationData, err := dynamodbattribute.MarshalMap(integrations)
	if err != nil {
		result["message"] = "Got error marshalling integrations"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	//PREPARE FOR INSERT TO AWS
	integrationInput := &dynamodb.PutItemInput{
		Item:      integrationData,
		TableName: aws.String(app.TABLE_NAME),
	}

	//INSERTING TO AWS
	_, err = app.SVC.PutItem(integrationInput)
	if err != nil {
		result["message"] = "Error while saving"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// generate log
	var logs []models.Logs
	// message: GroupX is now connected to IntegrationX
	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP_INTEGRATION,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			// User: &models.LogModuleParams {
			// 	ID: c.ViewArgs["userID"].(string),
			// },
			Integration: &models.LogModuleParams{
				ID: integrationID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["logs"] = "error while creating logs"
	}

	result["message"] = "Added Integrations"
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
DeleteIntegrations
*/

// @Summary Delete Integrations
// @Description This endpoint deletes integrations from a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.DeleteIntegrationsRequest true "Delete integrations body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/integrations/delete [post]
func (c GroupController) DeleteIntegrations() revel.Result {
	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	integrationID := c.Params.Form.Get("integration_id")

	result := make(map[string]interface{})
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.REMOVE_GROUP_INTEGRATION, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	if groupID == "" || integrationID == "" {
		result["params"] = "groupID: " + groupID + " integrationID: " + integrationID
		result["message"] = "Missing parameters"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)

	}

	getSubIntegs, err := GetGroupSubIntegrationsByParentID(groupID, integrationID)
	if err != nil {
		result["message"] = "Error getting sub integrations"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	if len(getSubIntegs) != 0 {
		for _, items := range getSubIntegs {
			subIntegrations := &dynamodb.DeleteItemInput{
				Key: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, items.GroupID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, items.IntegrationID)),
					},
				},
				TableName: aws.String(app.TABLE_NAME),
			}

			_, err := app.SVC.DeleteItem(subIntegrations)
			if err != nil {
				result["message"] = "Error deleting sub integrations"
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}
		}
	}

	integrations := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationID)),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.DeleteItem(integrations)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// // generate log
	// var logs []models.Logs
	// // message: IntegrationX has been disconnected to GroupX
	// logs = append(logs, models.Logs{
	// 	CompanyID: companyID,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
	// 	LogType:   constants.ENTITY_TYPE_GROUP,
	// 	LogInfo: &models.LogInformation{
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 		// User: &models.LogModuleParams {
	// 		// 	ID: c.ViewArgs["userID"].(string),
	// 		// },
	// 		User: &models.LogModuleParams {
	// 			ID: userID,
	// 		},
	// 		Integration: &models.LogModuleParams{
	// 			ID: integrationID,
	// 		},
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["logs"] = "error while creating logs"
	// }

	// create group activity & company log
	var errorMessages []string
	companyLog := models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			Integration: &models.LogModuleParams{
				ID: integrationID,
			},
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			Integration: &models.LogModuleParams{
				ID: integrationID,
			},
		},
	}

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating company logs for removing group and integration connection")
	}
	_ = companyLogID

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating group logs for removing group and integration connection")
	}
	_ = groupLogID

	if len(errorMessages) != 0 {
		result["errorMessages"] = errorMessages
	}

	result["message"] = "successfully deleted "
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(result)

}

/*
DeleteGroup
*/

// @Summary Delete Group
// @Description This endpoint deletes a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.DeleteGroupRequest true "Delete bookmark group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/delete [post]
func (c GroupController) DeleteGroup() revel.Result {
	result := make(map[string]interface{})
	userID := c.ViewArgs["userID"].(string)

	groupID := c.Params.Form.Get("group_id")
	departmentID := c.Params.Form.Get("department_id")

	currentTime := utils.GetCurrentTimestamp()

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.REMOVE_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	if len(groupID) == 0 {
		result["message"] = "Missing parameters gcm2"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	// check if userID exists
	u, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		result["message"] = "Error at Getting users"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(result)
	}

	// finding index of groupID
	var index string

	for idx, item := range u.BookmarkGroups {
		if item == groupID {
			//converting type int to string
			index = strconv.Itoa(idx)
		}
	}

	// preparing update query
	if index != "" {

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
			UpdateExpression: aws.String("REMOVE BookmarkGroups[" + index + "]"),
		}

		//executing query
		_, err := app.SVC.UpdateItem(input)

		//return 500 if errors
		if err != nil {
			result["message"] = "Error at inputting"
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

	}

	group := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":st": {
				S: aws.String(constants.ITEM_STATUS_INACTIVE),
			},
			":ua": {
				S: aws.String(currentTime),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#ST": aws.String("Status"),
		},
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID)),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET #ST = :st, UpdatedAt = :ua"),
	}

	_, err := app.SVC.UpdateItem(group)
	if err != nil {
		result["error"] = err.Error()
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	groupsJoined, err := GroupsJoined(groupID, constants.MEMBER_TYPE_GROUP)
	if err != nil {
		result["message"] = "failed fetching group joined"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}
	subGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), groupID, constants.MEMBER_TYPE_GROUP, c.Controller)
	if err != nil {
		result["message"] = "failed fetching sub groups"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}
	users, err := GetGroupMembers(c.ViewArgs["companyID"].(string), groupID, constants.MEMBER_TYPE_USER, c.Controller)
	if err != nil {
		result["message"] = "failed fetching group members"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	if len(groupsJoined) != 0 {
		for _, group := range groupsJoined {
			_, err := UpdateMemberStatus(group.GroupID, group.SK, constants.ITEM_STATUS_INACTIVE)
			if err != nil {
				result["message"] = "failed updating sub groups "
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}
		}
	}
	var departmentMembersToRemove []models.DepartmentMember
	if len(users) != 0 {
		for _, user := range users {
			_, err := UpdateMemberStatus(user.GroupID, user.SK, constants.ITEM_STATUS_INACTIVE)
			if err != nil {
				result["message"] = "failed updating user "
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}

			// check if member belongs to other Group of the same department
			userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), user.MemberID)
			deptCountOccurrence := 0 // GroupMember.DepartmentID matched to other Group.DepartmentID
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				result["message"] = "Something went wrong fetching group member's groups"
				return c.RenderJSON(result)
			}
			for _, userGroup := range userGroupsList {
				if userGroup.DepartmentID == departmentID && userGroup.GroupID != groupID {
					deptCountOccurrence++
				}
			}
			if deptCountOccurrence == 0 {
				// User doesn't exist to other Groups of the same Department
				departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
					PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
					SK:           utils.AppendPrefix(constants.PREFIX_USER, user.MemberID),
					DepartmentID: departmentID,
					UserID:       user.MemberID,
				})
			}
		}
	}

	if len(subGroups) != 0 {
		for _, subGroup := range subGroups {
			_, err := UpdateMemberStatus(subGroup.GroupID, subGroup.SK, constants.ITEM_STATUS_INACTIVE)
			if err != nil {
				result["message"] = "failed updating user "
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
				return c.RenderJSON(result)
			}

			groupMemberList, err := ops.GetGroupMembersOld(subGroup.GroupID, c.Controller)
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				result["message"] = "Something went wrong fetching group members"
				return c.RenderJSON(result)
			}
			if len(groupMemberList) != 0 {
				for _, grpMember := range groupMemberList {
					if grpMember.MemberType == constants.MEMBER_TYPE_USER {
						// check if member belongs to other Group of the same department
						userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), grpMember.MemberID)
						deptCountOccurrence := 0 // departmentID matched to other userGroup.DepartmentID
						if err != nil {
							result["status"] = utils.GetHTTPStatus(err.Error())
							result["message"] = "Something went wrong fetching group member's groups"
							return c.RenderJSON(result)
						}
						for _, userGroup := range userGroupsList {
							if userGroup.DepartmentID == departmentID && userGroup.GroupID != groupID {
								deptCountOccurrence++
							}
						}
						if deptCountOccurrence == 0 {
							// User doesn't exist to other Groups of the same Department
							departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
								PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
								SK:           utils.AppendPrefix(constants.PREFIX_USER, grpMember.MemberID),
								DepartmentID: departmentID,
								UserID:       grpMember.MemberID,
							})
						}
					}
				}
			}
		}
	}

	// REMOVE DEPARTMENT MEMBERS START
	if len(departmentMembersToRemove) != 0 {
		var filteredMembersToRemove []models.DepartmentMember
		for _, deptMember := range departmentMembersToRemove {
			skip := false
			for _, u := range filteredMembersToRemove {
				if deptMember.PK == u.PK && deptMember.SK == u.SK {
					skip = true
					break
				}
			}
			if !skip {
				filteredMembersToRemove = append(filteredMembersToRemove, deptMember)
			}
		}

		deptUserRemoveErr := RemoveDepartmentUsers(filteredMembersToRemove)
		if deptUserRemoveErr != nil {
			result["message"] = "Error while removing users from a department."
		}
	}

	var logs []models.Logs
	// message: GroupX has been un-bookmarked by UserX
	logs = append(logs, models.Logs{
		CompanyID: u.ActiveCompany,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_DELETE_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			User: &models.LogModuleParams{
				ID: userID,
			},
			Department: &models.LogModuleParams{
				ID: departmentID,
			},
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["message"] = "error while creating logs"
	}

	result["message"] = "successfully deleted "
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(result)
}

func GetGroupSubIntegrationsByParentID(groupID, parentID string) ([]models.GroupSubIntegration, error) {

	subIntegrations := []models.GroupSubIntegration{}

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
						S: aws.String(constants.PREFIX_SUB_INTEGRATION),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"ParentIntegrationID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(parentID),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return subIntegrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &subIntegrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return subIntegrations, e
	}

	return subIntegrations, nil
}

/*
****************
AddToGroup - Members
- Used to add members from the Group's Info page add member modal
- Same functionality with AddIndividualsAsMembers
****************
*/

// @Summary Add To Group
// @Description This endpoint is used to add members from the Group's Info page add member modal, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddToGroupRequest true "Add to group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/add [post]
func (c GroupController) AddToGroup() revel.Result {
	var membersID []string
	c.Params.Bind(&membersID, "members_id")

	result := make(map[string]interface{})
	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	departmentID := c.Params.Form.Get("department_id")
	memberType := c.Params.Form.Get("member_type")
	var memberRole, typeGroup, prefix string
	if memberType == constants.MEMBER_TYPE_OWNER {
		memberRole = constants.MEMBER_TYPE_OWNER
		typeGroup = constants.ENTITY_TYPE_GROUP_OWNER
		prefix = constants.PREFIX_OWNER
	} else {
		memberType = constants.MEMBER_TYPE_USER
		memberRole = constants.MEMBER_TYPE_USER
		typeGroup = constants.ENTITY_TYPE_GROUP_MEMBER
		prefix = constants.PREFIX_USER
	}
	//PREPARING GROUP MEMBERS
	var members []models.GroupMember
	var inputRequest []*dynamodb.WriteRequest
	var membersInput *dynamodb.BatchWriteItemInput
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.ADD_GROUP_MEMBER, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	if len(membersID) == 0 || groupID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		result["message"] = "Missing parameters"
		return c.RenderJSON(result)
	}

	// check if group exists
	group, err := GetGroupByID(groupID)
	if err != nil {
		result["error"] = "Group not exists."
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}
	departmentID = group.DepartmentID

	var departmentMembers []models.DepartmentMember

	var recipients []mail.Recipient

	//PREPARING GROUP MEMBERS
	for _, item := range membersID {

		userData, err := ops.GetUserData(item, "")
		if userData.UserID == "" {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400) // change status code
			return c.RenderJSON(result)
		}
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error()) // change status code
			return c.RenderJSON(result)
		}

		members = append(members, models.GroupMember{
			PK:         utils.AppendPrefix(constants.PREFIX_GROUP, groupID),
			SK:         utils.AppendPrefix(prefix, item),
			CompanyID:  c.ViewArgs["companyID"].(string),
			GroupID:    groupID,
			MemberID:   item,
			Status:     constants.ITEM_STATUS_ACTIVE,
			MemberType: memberType,
			MemberRole: memberRole,
			CreatedAt:  utils.GetCurrentTimestamp(),
			UpdatedAt:  utils.GetCurrentTimestamp(),
			Type:       typeGroup,
		})

		departmentMembers = append(departmentMembers, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, userData.UserID),
			DepartmentID: departmentID,
			UserID:       userData.UserID,
		})

		recipients = append(recipients, mail.Recipient{
			Name:       userData.FirstName + " " + userData.LastName,
			Email:      userData.Email,
			GroupName:  group.GroupName,
			ActionType: "added",
		})
	}
	//COMPILING INFORMATION FOR INSERTION
	for _, member := range members {
		inputRequest = append(inputRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": &dynamodb.AttributeValue{
					S: aws.String(member.PK),
				},
				"SK": &dynamodb.AttributeValue{
					S: aws.String(member.SK),
				},
				"CompanyID": &dynamodb.AttributeValue{
					S: aws.String(member.CompanyID),
				},
				"GroupID": &dynamodb.AttributeValue{
					S: aws.String(member.GroupID),
				},
				"MemberID": &dynamodb.AttributeValue{
					S: aws.String(member.MemberID),
				},
				"Status": &dynamodb.AttributeValue{
					S: aws.String(member.Status),
				},
				"MemberType": &dynamodb.AttributeValue{
					S: aws.String(member.MemberType),
				},
				"MemberRole": &dynamodb.AttributeValue{
					S: aws.String(member.MemberRole),
				},
				"CreatedAt": &dynamodb.AttributeValue{
					S: aws.String(member.CreatedAt),
				},
				"UpdatedAt": &dynamodb.AttributeValue{
					S: aws.String(member.UpdatedAt),
				},
				"Type": &dynamodb.AttributeValue{
					S: aws.String(member.Type),
				},
			},
		}})
		go cache.Set("member_"+member.MemberID, member, 30*time.Minute)

	}

	//send email for added members
	jobs.Now(mail.SendEmail{
		Subject:    "You have been added to a group",
		Recipients: recipients,
		Template:   "notify_group_member.html",
	})

	//SPECIFYING TABLE NAME
	membersInput = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			app.TABLE_NAME: inputRequest,
		},
	}

	//INSERTING DATA
	batchRes, err := app.SVC.BatchWriteItem(membersInput)
	_ = batchRes

	//ERROR AT INSERTING
	if err != nil {
		result["message"] = "Got error in put item (MEMBERS)"
		result["error"] = err.Error()
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// INSERT DEPARTMENT MEMBERS START
	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptAddUsersErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptAddUsersErr != nil {
		result["departmentError"] = "Error adding users to department"
	}
	// INSERT DEPARTMENT MEMBERS END

	// // generate log
	// var logs []models.Logs
	// // message: John Smith and John Doe has been added to GroupX
	// var logInfoUsers []models.LogModuleParams
	// for _, member := range members {
	// 	logInfoUsers = append(logInfoUsers, models.LogModuleParams{
	// 		ID: member.MemberID,
	// 	})
	// }
	// logs = append(logs, models.Logs{
	// 	CompanyID: companyID,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
	// 	LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
	// 	LogInfo: &models.LogInformation{
	// 		Users: logInfoUsers,
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["logs"] = "error while creating logs"
	// }

	// generate log
	//var logs []models.Logs
	// message: John Smith created GroupX
	// Remarks: always added logs that creating a group even if already exists and only adding a new member
	// logs = append(logs, models.Logs{
	// 	CompanyID: companyID,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_ADD_GROUP,
	// 	LogType:   constants.ENTITY_TYPE_GROUP,
	// 	LogInfo: &models.LogInformation{
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 		User: &models.LogModuleParams{
	// 			ID: c.ViewArgs["userID"].(string),
	// 		},
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["logs"] = "error while creating logs"
	// }

	// get suggested groups
	// suggestedGroups, err := GetSuggestedGroups(c.ViewArgs["companyID"].(string), groupID)
	// if err != nil { }

	// generate log
	// message: John Smith and John Doe has been added to GroupX
	logSearchKey := group.GroupName + " "
	var logInfoUsers []models.LogModuleParams
	for _, member := range members {
		u := getUserInTmp(member.MemberID)
		logSearchKey += u.FirstName + " " + u.LastName + " "
		logInfoUsers = append(logInfoUsers, models.LogModuleParams{
			ID: member.MemberID,
		})
	}

	companyLog := models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	var errorMessages []string

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating company logs")
	}

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating group logs")
	}
	_ = groupLogID

	groupList, err := GetGroupList(companyID)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}

	output := len(groupList)

	var actionItem models.ActionItem
	// generate action items?
	if output >= 2 {
		actionItem = models.ActionItem{
			CompanyID:      companyID,
			LogID:          companyLogID,
			ActionItemType: "ADD_USERS_TO_GROUP",
			SearchKey:      logSearchKey,
		}

	}

	actionItemID, err := ops.CreateActionItem(actionItem)
	if err != nil {
		result["logs"] = "error while creating action items"
	}

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["actionItemID"] = actionItemID
	result["group"] = groupID
	// result["suggestedGroups"] = suggestedGroups
	result["message"] = "Members Added"
	return c.RenderJSON(result)
}

// @Summary Get Suggested Groups From Group
// @Description This endpoint retrieves a list of groups suggested from a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param exclude_users_groups query string false "Excludes specified user groups"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/suggested/:groupID [get]
func (c GroupController) GetSuggestedGroupsFromGroup(groupID string) revel.Result {
	result := make(map[string]interface{})
	companyID := c.ViewArgs["companyID"].(string)
	userID := c.ViewArgs["userID"].(string)

	excludeUsersGroups := c.Params.Query.Get("exclude_users_groups")
	excludeUsersGroupsUnmarshalled := []string{}
	err := json.Unmarshal([]byte(excludeUsersGroups), &excludeUsersGroupsUnmarshalled)
	if err != nil {
	}

	group, err := GetGroupByID(groupID)
	if err != nil {
		result["message"] = "Error while getting group"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	_ = group

	groupsOccurence := make(map[string]int)

	// get user groups to be filter
	for _, userKey := range excludeUsersGroupsUnmarshalled {
		groups, err := GetUserCompanyGroups(companyID, userKey)
		if err == nil {
			// items = append(items, groups...)
			// c
			for _, group := range groups {
				if groupID == group.GroupID {
					continue
				}
				if val, ok := groupsOccurence[group.GroupID]; ok {
					groupsOccurence[group.GroupID] = val + 1
					continue
				}
				groupsOccurence[group.GroupID] = 1
			}
		}
	}

	var groupsToExclude []string
	for key, el := range groupsOccurence {
		if el >= len(excludeUsersGroupsUnmarshalled) {
			groupsToExclude = append(groupsToExclude, key)
		}
	}

	groupsToExclude = append(groupsToExclude, userID)

	suggestedGroups, err := GetSuggestedGroups(companyID, groupID, groupsToExclude, excludeUsersGroupsUnmarshalled, c.Controller)
	if err != nil {
		c.Response.Status = 400
		result["message"] = err.Error()
		return c.RenderJSON(result)
	}

	return c.RenderJSON(suggestedGroups)
}

func GetSuggestedGroups(companyID, groupID string, groupsToExclude, usersToExclude []string, c *revel.Controller) ([]models.Group, error) {
	var suggestedGroups []models.Group

	memberUsers, err := GetGroupMembers(companyID, groupID, constants.MEMBER_TYPE_USER, c)
	if err == nil {
	}

	var memberIds []string
	for _, member := range memberUsers {
		if !utils.StringInSlice(member.MemberID, usersToExclude) {
			memberIds = append(memberIds, member.MemberID)
		}
	}

	groupsToExclude = append(groupsToExclude, groupID)

	// get company group members
	companyGroups, err := GetGroupMembersInCompany(GetGroupMembersInCompanyInput{
		CompanyID: companyID,
		Filter: GetGroupMembersInCompanyFilterInput{
			Users:  memberIds,
			Groups: groupsToExclude,
		},
	})
	if err != nil {
		return suggestedGroups, err
	}

	if len(companyGroups) == 0 {
		return suggestedGroups, nil
	}

	// group results by group id
	groupedCompanyGroups := make(map[string][]models.GroupMember)
	for _, cGroup := range companyGroups {
		if val, ok := groupedCompanyGroups[cGroup.GroupID]; ok {
			groupedCompanyGroups[cGroup.GroupID] = append(val, cGroup)
			continue
		}
		groupedCompanyGroups[cGroup.GroupID] = []models.GroupMember{cGroup}
	}

	// filter groups with 85% number of members same with the current group
	percentage := 85 // transfer to constants or in db
	suggestedMembersLen := math.Floor((float64(percentage) / 100) * float64(len(memberIds)))

	for groupKey, groupMembers := range groupedCompanyGroups {
		if len(groupMembers) >= int(suggestedMembersLen) && len(groupMembers) > 1 {
			g, err := GetGroupByID(groupKey)
			if err == nil && g.Status == constants.ITEM_STATUS_ACTIVE {
				g.GroupMembers = groupMembers
				suggestedGroups = append(suggestedGroups, g)
			}
		}
	}

	return suggestedGroups, nil
}

type GetGroupMembersInCompanyFilterInput struct {
	Users  []string
	Groups []string
}

type GetGroupMembersInCompanyInput struct {
	CompanyID string
	Filter    GetGroupMembersInCompanyFilterInput
}

func GetGroupMembersInCompany(input GetGroupMembersInCompanyInput) ([]models.GroupMember, error) {
	var items []map[string]*dynamodb.AttributeValue
	result, err := GetGroupMembersInCompanyQuery(input, nil)
	if err != nil {
		return []models.GroupMember{}, err
	}
	items = append(items, result.Items...)
	key := result.LastEvaluatedKey

	for len(key) != 0 {
		result, err := GetGroupMembersInCompanyQuery(input, key)
		if err != nil {

		}
		items = append(items, result.Items...)
		key = result.LastEvaluatedKey
	}

	var members []models.GroupMember

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		return members, err
	}

	return members, nil
}

func GetGroupMembersInCompanyQuery(input GetGroupMembersInCompanyInput, exclusiveStartKey map[string]*dynamodb.AttributeValue) (*dynamodb.QueryOutput, error) {
	// filterGroups, err := dynamodbattribute.MarshalList(input.Filter.Groups)
	// if err != nil { }

	// filterUsers, err := dynamodbattribute.MarshalList(input.Filter.Users)
	// if err != nil { }

	expressionAttrValues := map[string]*dynamodb.AttributeValue{
		":sk": {
			S: aws.String(constants.PREFIX_OWNER),
		},
	}

	var memberKeys []string
	for idx, userKey := range input.Filter.Users {
		expressionAttrValues[":user"+strconv.Itoa(idx)] = &dynamodb.AttributeValue{
			S: aws.String(userKey),
		}
		memberKeys = append(memberKeys, ":user"+strconv.Itoa(idx))
	}

	var filtMemberExpr string
	if len(memberKeys) != 0 {
		filtMemberExpr = "MemberID in (" + strings.Join(memberKeys, ",") + ") AND "
	}

	var groupKeys []string
	for idx, groupKey := range input.Filter.Groups {
		expressionAttrValues[":group"+strconv.Itoa(idx)] = &dynamodb.AttributeValue{
			S: aws.String(groupKey),
		}
		groupKeys = append(groupKeys, ":group"+strconv.Itoa(idx))
	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(input.CompanyID),
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
		ExpressionAttributeValues: expressionAttrValues,
		ExpressionAttributeNames: map[string]*string{
			"#SK":      aws.String("SK"),
			"#GroupID": aws.String("GroupID"),
		},
		FilterExpression: aws.String(filtMemberExpr + "not contains(#SK, :sk) AND not (#GroupID in (" + strings.Join(groupKeys, ",") + "))"),
		// QueryFilter: map[string]*dynamodb.Condition{
		// 	"MemberID": {
		// 		ComparisonOperator: aws.String(constants.CONDITION_IN),
		// 		AttributeValueList: filterUsers,
		// 	},
		// 	"SK": {
		// 		ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
		// 		AttributeValueList: []*dynamodb.AttributeValue{
		// 			{
		// 				S: aws.String(constants.PREFIX_USER),
		// 			},
		// 		},
		// 	},
		// 	"GroupID": {
		// 		ComparisonOperator: aws.String(constants.CONDITION_NOT_EQUAL),
		// 		AttributeValueList: []*dynamodb.AttributeValue{
		// 			{
		// 				S: aws.String(input.Filter.Groups[0]),
		// 			},
		// 		},
		// 	},
		// },
		IndexName:         aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		ExclusiveStartKey: exclusiveStartKey,
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type AddUsersToGroupsInput struct {
	Groups []string `json:"groups"`
	Users  []string `json:"users"`
}

// @Summary Add Users To Groups
// @Description This endpoint adds users to specified groups, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddUsersToGroupsRequest true "Add users to groups body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/bulk/add [post]
func (c GroupController) AddUsersToGroups() revel.Result {
	result := make(map[string]interface{})
	var input AddUsersToGroupsInput
	c.Params.BindJSON(&input)

	companyID := c.ViewArgs["companyID"].(string)

	if len(input.Groups) == 0 || len(input.Users) == 0 {
		c.Response.Status = 422
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	// remove duplicates
	filteredGroups := utils.RemoveDuplicateStrings(input.Groups)
	filteredUsers := utils.RemoveDuplicateStrings(input.Users)

	// filter existing groups
	var groups []models.Group
	for _, groupKey := range filteredGroups {
		// check if group exists
		group, err := GetGroupByID(groupKey)
		if err == nil {
			groups = append(groups, group)
		}
	}

	// filter existing users
	var users []models.CompanyUser
	for _, userKey := range filteredUsers {
		// check if user exists in company
		user, err := GetCompanyUser(companyID, userKey)
		if err == nil {
			users = append(users, user)
		}
	}

	// loop in filtered groups
	for _, group := range groups {
		var batches [][]*dynamodb.WriteRequest
		var currentBatch []*dynamodb.WriteRequest

		var departmentMembers []models.DepartmentMember
		var logInfoUsers []models.LogModuleParams
		var errorMessages []string

		for i, user := range users {
			currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": &dynamodb.AttributeValue{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
					},
					"SK": &dynamodb.AttributeValue{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, user.UserID)),
					},
					"CompanyID": &dynamodb.AttributeValue{
						S: aws.String(companyID),
					},
					"GroupID": &dynamodb.AttributeValue{
						S: aws.String(group.GroupID),
					},
					"MemberID": &dynamodb.AttributeValue{
						S: aws.String(user.UserID),
					},
					"MemberType": &dynamodb.AttributeValue{ // is it needed?
						S: aws.String(constants.MEMBER_TYPE_USER),
					},
					"Status": &dynamodb.AttributeValue{
						S: aws.String(user.Status),
					},
					"MemberRole": &dynamodb.AttributeValue{
						S: aws.String(constants.MEMBER_TYPE_USER),
					},
					"CreatedAt": &dynamodb.AttributeValue{
						S: aws.String(utils.GetCurrentTimestamp()),
					},
					"UpdatedAt": &dynamodb.AttributeValue{
						S: aws.String(utils.GetCurrentTimestamp()),
					},
					"Type": &dynamodb.AttributeValue{ // is it needed?
						S: aws.String(constants.MEMBER_TYPE_USER),
					},
				},
			}})

			departmentMembers = append(departmentMembers, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, group.DepartmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, user.UserID),
				DepartmentID: group.DepartmentID,
				UserID:       user.UserID,
			})

			logInfoUsers = append(logInfoUsers, models.LogModuleParams{
				ID: user.UserID,
			})

			if i%constants.BATCH_LIMIT == 0 {
				batches = append(batches, currentBatch)
				currentBatch = nil
			}
		}

		if len(currentBatch) > 0 && len(currentBatch) != constants.BATCH_LIMIT {
			batches = append(batches, currentBatch)
		}

		_, err := ops.BatchWriteItemHandler(batches)
		if err != nil {
		} // handle error, need to adjust BatchWriteItemHandler()

		addDepartmentUsersErr := AddDepartmentUsers(departmentMembers, group.DepartmentID)
		if addDepartmentUsersErr != nil {
		} // handle error, need to adjust AddDepartmentUsers()

		// insert log
		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: group.GroupID,
				},
			},
		}

		groupLog := models.Logs{
			GroupID:   group.GroupID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
			LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
			LogInfo: &models.LogInformation{
				Users: logInfoUsers,
				Group: &models.LogModuleParams{
					ID: group.GroupID,
				},
			},
		}

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating company logs")
		}
		_ = companyLogID

		groupLogID, err := ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating group logs")
		}
		_ = groupLogID
	}

	return c.RenderJSON(nil)
}

/*
**********
Upate Member Type from group
**********
*/
func UpdateMemberStatus(groupID, sk, statusItem string) (bool, error) {
	currentTime := utils.GetCurrentTimestamp()

	if groupID == "" || sk == "" || statusItem == "" {
		e := errors.New(constants.HTTP_STATUS_400)
		return false, e
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":st": {
				S: aws.String(statusItem),
			},
			":ua": {
				S: aws.String(currentTime),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#ST": aws.String("Status"),
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
			},
			"SK": {
				S: aws.String(sk),
			},
		},
		TableName:        aws.String(app.TABLE_NAME),
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET #ST = :st, UpdatedAt = :ua"),
	}

	_, err := app.SVC.UpdateItem(input)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return false, e
	}

	return true, nil
}

/*
**********
Delete Member from group
**********
*/

// @Summary Delete Member
// @Description This endpoint removes a member from a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.DeleteMemberRequest true "Delete member body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/delete [post]
func (c GroupController) DeleteMember() revel.Result {
	var membersID []string
	c.Params.Bind(&membersID, "members_id")

	result := make(map[string]interface{})
	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	memberType := c.Params.Form.Get("member_type")
	userID := c.ViewArgs["userID"].(string)
	departmentID := c.Params.Form.Get("department_id")
	prefix := constants.PREFIX_USER
	if memberType == constants.MEMBER_TYPE_OWNER {
		prefix = constants.PREFIX_OWNER
	}

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.REMOVE_GROUP_MEMBER, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}
	//check if groupMembers is empty
	if len(membersID) == 0 {
		result["message"] = "No members to delete"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// check if groupID exists
	group, err := GetGroupByID(groupID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}
	logSearchKey := group.GroupName + " "

	var departmentMembersToRemove []models.DepartmentMember

	//prepare for deleting -original code heere
	var recipients []mail.Recipient

	for _, items := range membersID {
		userData, opsErr := ops.GetUserData(items, "")
		if userData.UserID == "" {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400) // change status code
			return c.RenderJSON(result)
		}
		if opsErr != nil {
			result["status"] = utils.GetHTTPStatus(opsErr.Error()) // change status code
			return c.RenderJSON(result)
		}

		groupMember := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(prefix, items)),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err := app.SVC.DeleteItem(groupMember)
		//error from aws
		if err != nil {
			result["message"] = "Error from DB"
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, items),
			DepartmentID: departmentID,
			UserID:       items,
		})

		recipients = append(recipients, mail.Recipient{
			Name:       userData.FirstName + " " + userData.LastName,
			Email:      userData.Email,
			GroupName:  group.GroupName,
			ActionType: "removed",
		})
		logSearchKey += userData.FirstName + " " + userData.LastName + " "
	}

	// REMOVE DEPARTMENT MEMBERS START
	if len(departmentMembersToRemove) != 0 {
		var filteredMembersToRemove []models.DepartmentMember
		for _, deptMember := range departmentMembersToRemove {
			skip := false
			for _, u := range filteredMembersToRemove {
				if deptMember.PK == u.PK && deptMember.SK == u.SK {
					skip = true
					break
				}
			}
			if !skip {
				filteredMembersToRemove = append(filteredMembersToRemove, deptMember)
			}
		}

		deptUserRemoveErr := RemoveDepartmentUsers(filteredMembersToRemove)
		if deptUserRemoveErr != nil {
			result["message"] = "Error while removing users from a department."
		}
	}

	//send email for removed members
	jobs.Now(mail.SendEmail{
		Subject:    "You have been removed from a group",
		Recipients: recipients,
		Template:   "notify_group_member.html",
	})
	// REMOVE DEPARTMENT MEMBERS END

	// generate log
	var logs []models.Logs
	// message: GroupX has been deleted by UserX
	var deletedGroupMembers []models.LogModuleParams
	for _, m := range membersID {
		deletedGroupMembers = append(deletedGroupMembers, models.LogModuleParams{
			ID: m,
		})
		go cache.Delete("member_" + m)
	}

	hasGroup := false
	var logInfoUsers []models.LogModuleParams
	for _, member := range membersID {
		// CHECK IF THE MEMBER HAVE OTHER GROUP
		userGroups, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), member)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			result["message"] = "Something went wrong with group member groups"
		}

		if len(userGroups) > 0 {
			hasGroup = true
		}

		// need users info for logs/activities to display
		logInfoUsers = append(logInfoUsers, models.LogModuleParams{
			ID: member,
		})
	}

	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			GroupMembers: deletedGroupMembers,
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
			Users: logInfoUsers,
		},
	})

	// check if the user has any group, then return data without action items
	// if hasGroup {
	// 	// createBatch log
	// 	_, err := CreateBatchLog(logs)
	// 	if err != nil {
	// 		result["message"] = "error while creating logs"
	// 	}
	// 	// return data
	// 	result["message"] = "Removed members from group"
	// 	result["group"] = groupID
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	// 	return c.RenderJSON(result)
	// }

	companyLog := models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Users: logInfoUsers,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	var errorMessages []string

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating company log")
	}

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating group log")
	}
	_ = groupLogID

	// check if the user have no group, then return data with action items
	if !hasGroup {

		actionItem := models.ActionItem{
			CompanyID:      companyID,
			LogID:          companyLogID,
			ActionItemType: "ADD_REMOVE_USER",
			SearchKey:      logSearchKey,
		}
		actionItemID, err := ops.CreateActionItem(actionItem)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating action items")
		}
		result["actionItemID"] = actionItemID
	}

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	result["message"] = "Removed members from group"
	result["group"] = groupID
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(result)
}

/*
****************
AddGroup
- Function for creating a new group and members (users, groups)
- Used on CREATE, CLONE, MERGE, BRANCH group
Params:
group <Group> - item details
groups, users <[]GroupMember> - array of item ids and role (if member user)
****************
*/
func (c GroupController) AddGroup(companyID string, group models.Group, groups, users []models.GroupMember) (models.Group, string, error) {

	// generate group color
	qd := ops.GetDepartmentByIDPayload{
		DepartmentID: group.DepartmentID,
		CheckStatus:  true,
	}
	department, opsError := ops.GetDepartmentByID(qd, c.Validation)
	if opsError != nil {
		e := errors.New(constants.HTTP_STATUS_470)
		return models.Group{}, "", e
	}
	// // check if department id exists
	// department, err := ops.GetDepartmentByID(group.DepartmentID)
	// if err != nil {
	// 	e := errors.New(constants.HTTP_STATUS_470)
	// 	return models.Group{}, "", e
	// }

	// check if group name is unique
	grpNameUnique := IsGroupNameUnique(group.CompanyID, group.GroupName)
	if grpNameUnique {
		e := errors.New(constants.HTTP_STATUS_409)
		return models.Group{}, "", e
	}

	// insert group
	av, err := dynamodbattribute.MarshalMap(group)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return models.Group{}, "", e
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return models.Group{}, "", e
	}

	// insert member users
	if len(users) != 0 {
		err := c.AddGroupMembers(companyID, constants.MEMBER_TYPE_USER, group, users, false)
		if err != nil {
			e := errors.New(err.Error())
			return models.Group{}, "", e
		}
	}

	// insert member groups
	if len(groups) != 0 {
		err := c.AddGroupMembers(companyID, constants.MEMBER_TYPE_GROUP, group, groups, false)
		if err != nil {
			e := errors.New(err.Error())
			return models.Group{}, "", e
		}
	}

	return group, department.CompanyID, nil
}

/*
****************
AddGroupMembers
- Add group member based on their member type
Params:
memberType <string> - USER, GROUP
group <Group> - members will be added to this group passed
members <[]GroupMember> - array of item ids and role (if member user)
****************
*/
func (c GroupController) AddGroupMembers(companyID, memberType string, group models.Group, members []models.GroupMember, createLog bool) error {
	// check if type is valid, add
	memberType = strings.ToUpper(memberType)
	if !utils.StringInSlice(memberType, constraints.MEMBER_TYPES) {
		e := errors.New(constants.HTTP_STATUS_400)
		return e
	}

	// check if members length is equal to 0, return nil
	if len(members) == 0 {
		return nil
	}

	// loop insert batch of GROUP_MEMBER ENTITY
	batchLimit := 25
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, member := range members {
		switch memberType {
		case constants.MEMBER_TYPE_USER:
			currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, member.MemberID)),
					},
					"GroupID": {
						S: aws.String(group.GroupID),
					},
					"MemberID": {
						S: aws.String(member.MemberID),
					},
					"CompanyID": {
						S: aws.String(companyID),
					},
					"Status": {
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
					"Type": {
						S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
					},
					"MemberType": {
						S: aws.String(constants.MEMBER_TYPE_USER),
					},
					"CreatedAt": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
				},
			}})
		case constants.MEMBER_TYPE_GROUP:
			currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string))),
					},
					"GroupID": {
						S: aws.String(group.GroupID),
					},
					"GroupName": {
						S: aws.String(member.GroupName),
					},
					"MemberID": {
						S: aws.String(utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string)),
					},
					"DepartmentID": {
						S: aws.String(member.DepartmentID),
					},
					"CompanyID": {
						S: aws.String(companyID),
					},
					"Status": {
						S: aws.String(constants.ITEM_STATUS_ACTIVE),
					},
					"MemberType": {
						S: aws.String(constants.MEMBER_TYPE_GROUP),
					},
					"Type": {
						S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
					},
					"CreatedAt": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
				},
			}})
		}
		if i%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError := ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		e := errors.New(batchError.Error())
		return e
	}

	//Commented out to make logs for clone and branch group only one.
	// // generate log
	// var logs []models.Logs
	// // message: John Smith and John Doe has been added to GroupX
	// var logInfoUsers []models.LogModuleParams
	// var logInfoGroups []models.LogModuleParams
	// for _, member := range members {
	// 	logInfoGroups = append(logInfoGroups, models.LogModuleParams{
	// 		ID: member.MemberID,
	// 		Name: member.GroupName,

	// 	})
	// }
	// subGroups, err := GetGroupMembers(group.GroupID, constants.MEMBER_TYPE_GROUP)
	// if err != nil {
	// 	e := errors.New(err.Error())
	// 	return e
	// }

	// for _, subGroup := range subGroups {
	// 	logInfoUsers = append(logInfoUsers, models.LogModuleParams{
	// 		ID:   subGroup.MemberID,
	// 		Name: subGroup.GroupName,
	// 	})
	// }

	// logInfo := &models.LogInformation{}
	// if memberType == constants.MEMBER_TYPE_USER {
	// 	logInfo.Users = logInfoUsers
	// 	logInfo.Group = &models.LogModuleParams{
	// 		ID: group.GroupID,
	// 	}
	// }
	// // else{
	// // 	logInfo.Groups = logInfoGroups
	// // 	logInfo.Group = &models.LogModuleParams{
	// // 		ID: group.GroupID,
	// // 	}
	// // }

	// logs = append(logs, models.Logs{
	// 	CompanyID: companyID,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
	// 	LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
	// 	// LogInfo: &models.LogInformation{
	// 	// 	Users: logInfoUsers,
	// 	// 	Group: &models.LogModuleParams{
	// 	// 		ID: group.GroupID,
	// 	// 	},
	// 	// 	Groups: logInfoGroups,
	// 	// },
	// 	LogInfo: logInfo,
	// })

	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	return nil
	// }

	return nil
}

/*
*************************
AddMultipleUsersToMultipleGroups
*/

// @Summary Add Multiple Users To Multiple Groups
// @Description This endpoint adds multiple users to multiple groups, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddMultipleUsersToMultipleGroupsRequest true "Add multiple users to multiple groups body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/add/multiple [post]
func (c GroupController) AddMultipleUsersToMultipleGroups(companyID, memberType string, groups []models.Group, members []models.GroupMember, createLog bool) error {
	// check if type is valid, add
	memberType = strings.ToUpper(memberType)
	if !utils.StringInSlice(memberType, constraints.MEMBER_TYPES) {
		e := errors.New(constants.HTTP_STATUS_400)
		return e
	}

	// check if members length is equal to 0, return nil
	if len(members) == 0 {
		return nil
	}

	// loop insert batch of GROUP_MEMBER ENTITY
	batchLimit := 25
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, member := range members {
		for _, group := range groups {
			switch memberType {
			case constants.MEMBER_TYPE_USER:
				currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"PK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
						},
						"SK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, member.MemberID)),
						},
						"GroupID": {
							S: aws.String(group.GroupID),
						},
						"MemberID": {
							S: aws.String(member.MemberID),
						},
						"CompanyID": {
							S: aws.String(companyID),
						},
						"Status": {
							S: aws.String(constants.ITEM_STATUS_ACTIVE),
						},
						"Type": {
							S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
						},
						"MemberType": {
							S: aws.String(constants.MEMBER_TYPE_USER),
						},
						"CreatedAt": {
							S: aws.String(utils.GetCurrentTimestamp()),
						},
					},
				}})
			case constants.MEMBER_TYPE_GROUP:
				currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"PK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
						},
						"SK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string))),
						},
						"GroupID": {
							S: aws.String(group.GroupID),
						},
						"GroupName": {
							S: aws.String(member.GroupName),
						},
						"MemberID": {
							S: aws.String(utils.IfThenElse(member.MemberID != "", member.MemberID, member.GroupID).(string)),
						},
						"DepartmentID": {
							S: aws.String(member.DepartmentID),
						},
						"CompanyID": {
							S: aws.String(companyID),
						},
						"Status": {
							S: aws.String(constants.ITEM_STATUS_ACTIVE),
						},
						"MemberType": {
							S: aws.String(constants.MEMBER_TYPE_GROUP),
						},
						"Type": {
							S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
						},
						"CreatedAt": {
							S: aws.String(utils.GetCurrentTimestamp()),
						},
					},
				}})
			}
			if i%batchLimit == 0 {
				batches = append(batches, currentBatch)
				currentBatch = nil
			}
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError := ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		e := errors.New(batchError.Error())
		return e
	}

	return nil
}

/*
****************
RemoveGroupMembers
Used for adding members by Role (ADMIN, MEMBER, VIEWER)
Params:
members - array of struct { PK, SK string }, eg: [{"PK": "GROUP#12345", "SK": "USER#12345"}]
****************
*/
func RemoveGroupMembers(memberType string, members []models.GroupMember, group models.Group) error {

	batchLimit := 25
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, member := range members {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(member.PK),
				},
				"SK": {
					S: aws.String(member.SK),
				},
			},
		}})
		if i%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, err := ops.BatchWriteItemHandler(batches)
	if err != nil {
		e := errors.New(err.Error())
		return e
	}

	// // generate log
	// var logs []models.Logs
	// // message: John Smith and John Doe has been added to GroupX
	// var logInfoUsers []models.LogModuleParams
	// for _, member := range members {
	// 	logInfoUsers = append(logInfoUsers, models.LogModuleParams{
	// 		ID: member.MemberID,
	// 	})
	// }
	// logs = append(logs, models.Logs {
	// 	CompanyID: companyID,
	// 	UserID: c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
	// 	LogType: constants.ENTITY_TYPE_GROUP_MEMBER,
	// 	LogInfo: &models.LogInformation {
	// 		Users: logInfoUsers,
	// 		Group: &models.LogModuleParams {
	// 			ID: group.GroupID,
	// 		},
	// 	},
	// })
	// _, err := CreateBatchLog(logs)
	// if err != nil {
	// 	// result["message"] = "error while creating logs"
	// }

	return nil
}

/*
****************
CloneGroup
- Generate identical copy of group
Params: groupID
Body:
group_name <string> - required
****************
*/

// @Summary Clone Group
// @Description This endpoint clones a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param body body models.CloneGroupRequest true "Clone group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/clone/:groupID [post]
func (c GroupController) CloneGroup(groupID string) revel.Result {
	result := make(map[string]interface{})

	// get param
	groupName := c.Params.Form.Get("group_name")
	companyID := c.Params.Form.Get("company_id") // new
	groupEmail := c.Params.Form.Get("group_email")
	groupDescription := c.Params.Form.Get("group_description")

	groupUUID := utils.GenerateTimestampWithUID()
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.CLONE_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	// check if group id exists
	group, err := GetGroupByID(groupID)
	if err != nil {
		result["Message"] = "Existing group name"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	tmp := group

	// update values
	group.PK = utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID)
	group.GroupID = groupUUID
	group.GSI_SK = utils.AppendPrefix(constants.PREFIX_GROUP, strings.ToLower(groupName))
	group.GroupName = groupName
	group.NewGroup = constants.BOOL_TRUE
	group.CreatedAt = utils.GetCurrentTimestamp()
	group.UpdatedAt = ""
	group.SearchKey = strings.ToLower(groupName)
	group.GroupDescription = strings.TrimSpace(groupDescription)
	if groupEmail != "" {
		group.AssociatedAccounts = map[string][]string{
			"google": []string{groupEmail},
		}
	}

	// validate form
	group.Validate(c.Validation, constants.USER_PERMISSION_ADD_GROUP)
	if c.Validation.HasErrors() {
		result["errors"] = c.Validation.Errors
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	// get member users
	memberUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), groupID, constants.MEMBER_TYPE_USER, c.Controller)
	if err != nil {
		result["message"] = "ERROR WITH GETTING GROUP MEMBERS (USERS)"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// get subgroups
	memberGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), groupID, constants.MEMBER_TYPE_GROUP, c.Controller)
	if err != nil {
		result["message"] = "ERROR WITH GETTING GROUP MEMBERS (GROUP)"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// add new group
	_, _, err = c.AddGroup(companyID, group, memberGroups, memberUsers)
	if err != nil {
		result["message"] = "ERROR WITH CLONING GROUP"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	res, err := CloneIntegrations(groupID, groupUUID, companyID, []string{}, true)
	if !res {
		result["message"] = "ERROR WHILE CLONING INTEGRATIONS"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// generate log
	var logs []models.Logs

	var newMembers []models.LogModuleParams
	for _, m := range memberUsers {
		newMembers = append(newMembers, models.LogModuleParams{
			ID:   m.MemberID,
			Type: m.MemberType,
		})
	}

	// message: GroupY has been cloned to GroupX

	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_CLONE_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: group.GroupID,
			},
			Origin: &models.LogModuleParams{
				ID: tmp.GroupID,
			},
			Members: newMembers,
		},
	})

	_, err = CreateBatchLog(logs)
	if err != nil {
		result["logs"] = "error while creating logs"
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	// result["logID"] = logID

	return c.RenderJSON(result)
}

/*
CLONE INTEGRATIONS
*/
func CloneIntegrations(oldGroupID, newGroupID string, companyID string, integrationListToCopy []string, isCopyConnectedItems bool) (bool, error) {
	// var filterIntegrations []models.GroupIntegration
	// var filterSubIntegrations []models.GroupIntegration
	getIntegrations, err := GetGroupIntegrations(oldGroupID)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return false, e
	}

	getSubIntegrations, err := GetGroupSubIntegration(oldGroupID)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return false, e
	}

	var integrations []models.GroupIntegration
	var integrationRequest []*dynamodb.WriteRequest
	var integrationInput *dynamodb.BatchWriteItemInput
	var subIntegrations []models.GroupSubIntegration
	var subIntegrationRequest []*dynamodb.WriteRequest
	var subIntegrationInput *dynamodb.BatchWriteItemInput

	if len(getIntegrations) != 0 {
		for _, item := range getIntegrations {
			foundIntegration := true
			if len(integrationListToCopy) != 0 {
				foundIntegration = utils.FindStringInSplice(integrationListToCopy, item.IntegrationID)
			}
			if foundIntegration {
				integrations = append(integrations, models.GroupIntegration{
					PK:              utils.AppendPrefix(constants.PREFIX_GROUP, newGroupID),
					SK:              utils.AppendPrefix(constants.PREFIX_INTEGRATION, item.IntegrationID),
					GroupID:         newGroupID,
					IntegrationID:   item.IntegrationID,
					IntegrationName: item.IntegrationName,
					DisplayPhoto:    item.DisplayPhoto,
					CreatedAt:       utils.GetCurrentTimestamp(),
					UpdatedAt:       utils.GetCurrentTimestamp(),
					Type:            constants.ENTITY_TYPE_GROUP_INTEGRATION,
				})
			}
		}

		//COMPILING INFORMATION FOR INSERTION
		for _, integration := range integrations {
			integrationRequest = append(integrationRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": &dynamodb.AttributeValue{
						S: aws.String(integration.PK),
					},
					"SK": &dynamodb.AttributeValue{
						S: aws.String(integration.SK),
					},
					"GroupID": &dynamodb.AttributeValue{
						S: aws.String(integration.GroupID),
					},
					"IntegrationID": &dynamodb.AttributeValue{
						S: aws.String(integration.IntegrationID),
					},
					"IntegrationName": &dynamodb.AttributeValue{
						S: aws.String(integration.IntegrationName),
					},
					"DisplayPhoto": &dynamodb.AttributeValue{
						S: aws.String(integration.DisplayPhoto),
					},
					"CreatedAt": &dynamodb.AttributeValue{
						S: aws.String(integration.CreatedAt),
					},
					"UpdatedAt": &dynamodb.AttributeValue{
						S: aws.String(integration.UpdatedAt),
					},
					"Type": &dynamodb.AttributeValue{
						S: aws.String(integration.Type),
					},
					"CompanyID": &dynamodb.AttributeValue{
						S: aws.String(companyID),
					},
				},
			}})
		}

		//SPECIFYING TABLE NAME
		integrationInput = &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				app.TABLE_NAME: integrationRequest,
			},
		}

		//INSERTING DATA
		_, err = app.SVC.BatchWriteItem(integrationInput)
		//ERROR AT INSERTING
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_500)
			return false, e
		}
	}

	if isCopyConnectedItems {

		if len(getSubIntegrations) != 0 {
			for _, item := range getSubIntegrations {
				foundIntegration := true
				if len(integrationListToCopy) != 0 {
					foundIntegration = utils.FindStringInSplice(integrationListToCopy, item.ParentIntegrationID)
				}
				if foundIntegration {
					subIntegrations = append(subIntegrations, models.GroupSubIntegration{
						PK:                  utils.AppendPrefix(constants.PREFIX_GROUP, newGroupID),
						SK:                  utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, item.IntegrationID),
						GroupID:             newGroupID,
						IntegrationID:       item.IntegrationID,
						IntegrationName:     item.IntegrationName,
						ParentIntegrationID: item.ParentIntegrationID,
						DisplayPhoto:        item.DisplayPhoto,
						CreatedAt:           utils.GetCurrentTimestamp(),
						UpdatedAt:           utils.GetCurrentTimestamp(),
						Type:                constants.ENTITY_TYPE_GROUP_SUB_INTEGRATION,
					})
				}
			}

			//COMPILING INFORMATION FOR INSERTION
			for _, subIntegration := range subIntegrations {
				subIntegrationRequest = append(subIntegrationRequest, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
					Item: map[string]*dynamodb.AttributeValue{
						"PK": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.PK),
						},
						"SK": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.SK),
						},
						"GroupID": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.GroupID),
						},
						"IntegrationID": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.IntegrationID),
						},
						"IntegrationName": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.IntegrationName),
						},
						"ParentIntegrationID": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.ParentIntegrationID),
						},
						"DisplayPhoto": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.DisplayPhoto),
						},
						"CreatedAt": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.CreatedAt),
						},
						"UpdatedAt": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.UpdatedAt),
						},
						"Type": &dynamodb.AttributeValue{
							S: aws.String(subIntegration.Type),
						},
					},
				}})
			}

			//SPECIFYING TABLE NAME
			subIntegrationInput = &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{
					app.TABLE_NAME: subIntegrationRequest,
				},
			}

			//INSERTING DATA
			_, err = app.SVC.BatchWriteItem(subIntegrationInput)

			//ERROR AT INSERTING
			if err != nil {
				e := errors.New(constants.HTTP_STATUS_500)
				return false, e
			}
		}
		integWithConnectedItems, err := GetGroupConnectedItemsIntegrations(oldGroupID)
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_500)
			return false, e
		}
		for _, integConnected := range integWithConnectedItems {
			item := &models.Integration{
				PK:                  utils.AppendPrefix(constants.PREFIX_GROUP, newGroupID),
				SK:                  utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, integConnected.IntegrationID),
				GroupID:             newGroupID,
				IntegrationID:       integConnected.IntegrationID,
				ParentIntegrationID: integConnected.ParentIntegrationID,
				IntegrationSlug:     integConnected.IntegrationSlug,
				ConnectedItems:      integConnected.ConnectedItems,
				CompanyID:           companyID,
			}

			av, err := dynamodbattribute.MarshalMap(item)
			if err != nil {
				e := errors.New(constants.HTTP_STATUS_400)
				return false, e
			}

			input := &dynamodb.PutItemInput{
				Item:      av,
				TableName: aws.String(app.TABLE_NAME),
			}

			_, err = app.SVC.PutItem(input)
			if err != nil {
				e := errors.New(constants.HTTP_STATUS_400)
				return false, e
			}
		}
	}

	return true, nil
}

func GetGroupConnectedItemsIntegrations(groupID string) ([]models.Integration, error) {
	integrations := []models.Integration{}

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
						S: aws.String(constants.PREFIX_SUB_INTEGRATION),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return integrations, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &integrations)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return integrations, e
	}

	return integrations, nil
}

/*
****************
MergeGroup
- Combine two groups
- Requirements:
  - The groups must be on the same department
  - The groups must not have a parent group to avoid conflict

Body:
company_id - for logs
member_ids - group member id's to retained
retain_group_id - group to be retained
remove_group_id - group to be removed
integration_ids - integrations to retained
****************
*/

// @Summary Merge Group
// @Description This endpoint merges a specified group with another group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.MergeGroupRequest true "Merge group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/merge/ [post]
func (c GroupController) MergeGroup() revel.Result {
	// todo refactor

	result := make(map[string]interface{})

	// get request body
	memberIDs := c.Params.Form.Get("member_ids")
	companyID := c.Params.Form.Get("company_id")
	integrationIDs := c.Params.Form.Get("integration_ids")
	retainGroupID := c.Params.Form.Get("retain_group_id")
	removeGroupID := c.Params.Form.Get("remove_group_id")

	// unmarshal members to be inserted
	var groupMembers []models.GroupMember
	json.Unmarshal([]byte(memberIDs), &groupMembers)

	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.MERGE_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	// if len(groupMembers) == 0 {
	// 	result["message"] = "Invalid group members format." // change to constants later
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
	// 	return c.RenderJSON(result)
	// }

	//NEW CODE

	// check if retainGroupID and removeGroupID exists
	retainGroup, err := GetGroupByID(retainGroupID)
	if err != nil {
		result["message"] = "The group to be retain is invalid." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	removeGrp, err := GetGroupByID(removeGroupID)
	if err != nil {
		result["message"] = "The group to be remove is invalid." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	differentDept := retainGroup.DepartmentID != removeGrp.DepartmentID
	//
	var departmentMembersToAdd []models.DepartmentMember
	var departmentMembersToRemove []models.DepartmentMember
	// get group members
	retainGrpUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), retainGroupID, constants.MEMBER_TYPE_USER, c.Controller)
	if err != nil {
		result["message"] = "Error while getting group to be retained members." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	retainGrpMemberGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), retainGroupID, constants.MEMBER_TYPE_GROUP, c.Controller)
	if err != nil {
		result["message"] = "Error while getting group to be retained members." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	removeGrpUsers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), removeGroupID, constants.MEMBER_TYPE_USER, c.Controller)
	if err != nil {
		result["message"] = "Error while getting group to be removed members." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	if len(removeGrpUsers) != 0 && differentDept {
		for _, user := range removeGrpUsers {
			// remove department user from removeGrp.DepartmentID
			userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), user.MemberID)
			deptCountOccurrence := 0 // member.DepartmentID matched to other userGroup.DepartmentID
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				result["message"] = "Something went wrong fetching group member's groups"
				return c.RenderJSON(result)
			}
			for _, userGroup := range userGroupsList {
				if userGroup.DepartmentID == removeGrp.DepartmentID && userGroup.GroupID != removeGroupID {
					deptCountOccurrence++
				}
			}
			if deptCountOccurrence == 0 {
				// User doesn't exist to other Groups of the same Department
				departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
					PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, removeGrp.DepartmentID),
					SK:           utils.AppendPrefix(constants.PREFIX_USER, user.MemberID),
					DepartmentID: removeGrp.DepartmentID,
					UserID:       user.MemberID,
				})
			}

			// add department user to the retainGroup.DepartmentID
			departmentMembersToAdd = append(departmentMembersToAdd, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, retainGroup.DepartmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, user.MemberID),
				DepartmentID: retainGroup.DepartmentID,
				UserID:       user.MemberID,
			})
		}
	}

	removeGrpMemberGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), removeGroupID, constants.MEMBER_TYPE_GROUP, c.Controller)
	if err != nil {
		result["message"] = "Error while getting group to be remoeed members." // change to constants later
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}
	if len(removeGrpMemberGroups) != 0 && differentDept {
		for _, subgroup := range removeGrpMemberGroups {
			// get subgroup members
			groupMemberList, err := ops.GetGroupMembersOld(subgroup.GroupID, c.Controller)
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error())
				result["message"] = "Something went wrong fetching group members"
				return c.RenderJSON(result)
			}
			if len(groupMemberList) != 0 {
				for _, grpMember := range groupMemberList {
					if grpMember.MemberType == constants.MEMBER_TYPE_USER {
						// add department user to the retainGroup.DepartmentID
						departmentMembersToAdd = append(departmentMembersToAdd, models.DepartmentMember{
							PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, retainGroup.DepartmentID),
							SK:           utils.AppendPrefix(constants.PREFIX_USER, grpMember.MemberID),
							DepartmentID: retainGroup.DepartmentID,
							UserID:       grpMember.MemberID,
						})

						// prepare department user remove
						// check if member belongs to other Group of the same department
						userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), grpMember.MemberID)
						deptCountOccurrence := 0 // member.DepartmentID matched to other userGroup.DepartmentID
						if err != nil {
							result["status"] = utils.GetHTTPStatus(err.Error())
							result["message"] = "Something went wrong fetching group member's groups"
							return c.RenderJSON(result)
						}
						for _, userGroup := range userGroupsList {
							if userGroup.DepartmentID == removeGrp.DepartmentID && userGroup.GroupID != subgroup.GroupID {
								deptCountOccurrence++
							}
						}
						if deptCountOccurrence == 0 {
							// User doesn't exist to other Groups of the same Department
							departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
								PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, removeGrp.DepartmentID),
								SK:           utils.AppendPrefix(constants.PREFIX_USER, grpMember.MemberID),
								DepartmentID: removeGrp.DepartmentID,
								UserID:       grpMember.MemberID,
							})
						}
					}
				}
			}
		}
	}

	//CHECKING IF GROUPTOREMOVE IS MEMBER OF A GROUP
	//GetAllGroups to check if GroupToRemove is Member, if GroupToRemove is member, remove from group
	var groupMembersAppend []models.GroupMember
	var pageLimit int64
	pageLimit = constants.DEFAULT_PAGE_LIMIT
	paramLastEvaluatedKey := ""
	searchKey := ""

	//1. Get all groups to iterate
	groups, _, err := GetAllGroups(constants.ENTITY_TYPE_GROUP, pageLimit, paramLastEvaluatedKey, searchKey)
	if len(groups) != 0 {
		for _, group := range groups {
			//2. Fetch GroupMembers to check if groupToremove is member
			groupMembersAsGroups, err := GetGroupMembers(c.ViewArgs["companyID"].(string), group.GroupID, constants.MEMBER_TYPE_GROUP, c.Controller)
			if err != nil {
				result["message"] = "Error while getting group to be retained members." // change to constants later
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
				return c.RenderJSON(result)
			}

			if len(groupMembersAsGroups) != 0 {
				for _, groupMemberAsGroupToRemove := range groupMembersAsGroups {
					//3. Check if groupToRemove is member
					if groupMemberAsGroupToRemove.SK == constants.PREFIX_GROUP+removeGrp.GroupID && removeGrp.GroupName == groupMemberAsGroupToRemove.GroupName {
						//4. Append to remove
						groupMembersAppend = append(groupMembersAppend, groupMemberAsGroupToRemove)
					}
				}
			}
		}
	}

	//5. RemoveGroupMembers
	if len(groupMembersAppend) != 0 {
		err = RemoveGroupMembers("", groupMembersAppend, models.Group{})
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
	}

	// setRemoveGroupToInactive := true
	setRemoveGroupToInactive := true

	// check if group members id exists
	var validateMembers []models.GroupMember
	for _, m := range groupMembers {
		pk := utils.AppendPrefix(constants.PREFIX_GROUP, m.GroupID)
		prfx := constants.PREFIX_USER
		if m.MemberType == constants.MEMBER_TYPE_GROUP {
			prfx = constants.PREFIX_GROUP
		}
		sk := utils.AppendPrefix(prfx, m.MemberID)
		member, _ := GetGroupMember(pk, sk)
		if err == nil {
			validateMembers = append(validateMembers, member)
		}
		if m.MemberID == removeGroupID {
			setRemoveGroupToInactive = false
		}
	}

	filteredMembers := FilteredDuplicateMembers(validateMembers)

	// combine members
	var deleteMembers []models.GroupMember
	// var retainMembers []models.GroupMember
	deleteMembers = append(deleteMembers, retainGrpUsers...)
	deleteMembers = append(deleteMembers, retainGrpMemberGroups...)
	deleteMembers = append(deleteMembers, removeGrpUsers...)
	deleteMembers = append(deleteMembers, removeGrpMemberGroups...)

	// retainMembers = append(retainMembers, retainGrpUsers...)
	// retainMembers = append(retainMembers, retainGrpMemberGroups...)

	// filtered duplicates
	filteredDuplicates := FilteredDuplicateMembers(deleteMembers)

	// remove group members of both group
	batchLimit := 25
	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, member := range filteredDuplicates {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(member.PK),
				},
				"SK": {
					S: aws.String(member.SK),
				},
			},
		}})
		if i%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError := ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		result["message"] = "Error while deleting members." // change to constants later
	}

	// insert filteredMembers to be added in retainGroupID
	batches = nil
	currentBatch = nil
	for i, member := range filteredMembers {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, retainGroupID)),
				},
				"SK": {
					S: aws.String(member.SK),
				},
				"GroupID": {
					S: aws.String(retainGroupID),
				},
				"MemberID": {
					S: aws.String(member.MemberID),
				},
				"CompanyID": {
					S: aws.String(companyID),
				},
				"Status": {
					S: aws.String(constants.ITEM_STATUS_ACTIVE),
				},
				"Type": {
					S: aws.String(constants.ENTITY_TYPE_GROUP_MEMBER),
				},
				"MemberType": {
					S: aws.String(member.MemberType),
				},
				"CreatedAt": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
		}})
		if i%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError = ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		result["message"] = "Error while inserting members." // change to constants later
	}

	// Handle department users add and remove
	if differentDept {
		// INSERT DEPARTMENT MEMBERS START
		filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembersToAdd)
		deptUserAddErr := AddDepartmentUsers(filteredDepartmentMembers, retainGroup.DepartmentID)
		if deptUserAddErr != nil {
			result["message"] = "Error while adding users to a department."
		}
		// INSERT DEPARTMENT MEMBERS END

		// REMOVE DEPARTMENT MEMBERS START
		if len(departmentMembersToRemove) != 0 {
			filteredDepartmentMembersToRemove := FilterDuplicatedDepartmentMembers(departmentMembersToRemove)
			deptUserRemoveErr := RemoveDepartmentUsers(filteredDepartmentMembersToRemove)
			if deptUserRemoveErr != nil {
				result["message"] = "Error while removing users from a department."
			}
		}
		// REMOVE DEPARTMENT MEMBERS END
	}

	// get group integrations
	var retainGrpIntegrations []models.GroupIntegration
	var removeGrpIntegrations []models.GroupIntegration
	if integrationIDs != "" {
		integs, err := GetGroupIntegrations(retainGroupID)
		if err == nil {
			retainGrpIntegrations = integs
		}
		integs, err = GetGroupIntegrations(removeGroupID)
		if err == nil {
			removeGrpIntegrations = integs
		}
	}

	//get integration connected items
	integWithConnectedItems, err := GetGroupConnectedItemsIntegrations(removeGroupID)
	if err != nil {
		result["message"] = "error while GetGroupConnectedItemsIntegrations logs"
	}
	for _, integConnected := range integWithConnectedItems {
		item := &models.Integration{
			PK:                  utils.AppendPrefix(constants.PREFIX_GROUP, retainGroupID),
			SK:                  utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, integConnected.IntegrationID),
			GroupID:             retainGroupID,
			IntegrationID:       integConnected.IntegrationID,
			ParentIntegrationID: integConnected.ParentIntegrationID,
			IntegrationSlug:     integConnected.IntegrationSlug,
			ConnectedItems:      integConnected.ConnectedItems,
			CompanyID:           companyID,
		}

		av, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err = app.SVC.PutItem(input)
		if err != nil {
			result["err"] = err
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	// combine integrations
	var deleteIntegrations []models.GroupIntegration
	deleteIntegrations = append(deleteIntegrations, retainGrpIntegrations...)
	deleteIntegrations = append(deleteIntegrations, removeGrpIntegrations...)

	// remove all integrations for both group
	batches = nil
	currentBatch = nil

	for i, integration := range deleteIntegrations {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(integration.PK),
				},
				"SK": {
					S: aws.String(integration.SK),
				},
			},
		}})
		if i%batchLimit == 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
		}
	}

	if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
		batches = append(batches, currentBatch)
	}

	_, batchError = ops.BatchWriteItemHandler(batches)
	if batchError != nil {
		result["message"] = "Error while deleting both group integrations" // change to constants later
	}

	// unmarshal group ids
	var integrations []models.GroupIntegration
	// integrations = retainGrpIntegrations
	json.Unmarshal([]byte(integrationIDs), &integrations)

	// filter integrations
	filteredIntegrations := FilterDuplicateIntegrations(integrations)

	if len(filteredIntegrations) != 0 {
		// insert the integration to be retained
		batches = nil
		currentBatch = nil
		for i, integration := range filteredIntegrations {

			currentBatch = append(currentBatch, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, retainGroupID)),
					},
					"SK": {
						S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integration.IntegrationID)),
					},
					"GroupID": {
						S: aws.String(retainGroupID),
					},
					"IntegrationID": {
						S: aws.String(integration.IntegrationID),
					},
					"Type": {
						S: aws.String(constants.ENTITY_TYPE_GROUP_INTEGRATION),
					},
					"CreatedAt": {
						S: aws.String(utils.GetCurrentTimestamp()),
					},
					"CompanyID": {
						S: aws.String(companyID),
					},
				},
			}})
			if i%batchLimit == 0 {
				batches = append(batches, currentBatch)
				currentBatch = nil
			}
		}

		if len(currentBatch) > 0 && len(currentBatch) != batchLimit {
			batches = append(batches, currentBatch)
		}

		_, batchError = ops.BatchWriteItemHandler(batches)
		if batchError != nil {
			result["message"] = "Error while inserting integrations." // change to constants later
		}
	}

	// update removeGroupID status to DELETED if removeGroupID not exists in memberIDs

	if setRemoveGroupToInactive {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":ds": {
					S: aws.String(constants.ITEM_STATUS_DELETED),
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
					S: aws.String(removeGrp.PK),
				},
				"SK": {
					S: aws.String(removeGrp.SK),
				},
			},
			UpdateExpression: aws.String("SET #s = :ds, UpdatedAt = :ua"),
		}

		_, err = app.SVC.UpdateItem(input)
		if err != nil {
			result["error"] = err.Error()
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	// generate log
	var logs []models.Logs
	// message: Group Y merged into Group X. See the new members of Group X
	var logInforMembers []models.LogModuleParams
	for _, m := range filteredMembers {
		logInforMembers = append(logInforMembers, models.LogModuleParams{
			ID:   m.MemberID,
			Type: m.MemberType,
		})
	}

	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_MERGE_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			RetainedGroup: &models.LogModuleParams{
				ID: retainGroupID,
			},
			RemovedGroup: &models.LogModuleParams{
				ID: removeGroupID,
			},
			Members: logInforMembers,
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["message"] = "error while creating logs"
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
AddIndividualsAsMembers
- Add group member users to a given group
- Used in People module add to group modal
- Same functionality with AddToGroup
Body:
group_id <string> - (required)
users <[]GroupMember> - array of user id and role, eg: [{"ItemID": "<id>"", "GroupRole": "<role>"}, {"ItemID": "<id>"", "GroupRole": "<role>"}] (required)
****************
*/

// @Summary Add Individuals As Members
// @Description This endpoint adds individuals as members to a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddIndividualsAsMembersRequest true "Add individuals as members body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/add_members/individual [post]
func (c GroupController) AddIndividualsAsMembers() revel.Result {
	result := make(map[string]interface{})

	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	users := c.Params.Form.Get("users")
	userID := c.ViewArgs["userID"].(string)
	departmentID := c.Params.Form.Get("department_id")

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.ADD_GROUP_MEMBER, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	group, errGetGroup := GetGroupByID(groupID)
	if errGetGroup != nil {
		result["status"] = constants.HTTP_STATUS[errGetGroup.Error()]
		return c.RenderJSON(result)
	}

	var memberUsers []models.GroupMember
	json.Unmarshal([]byte(users), &memberUsers)

	if len(memberUsers) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	var departmentMembers []models.DepartmentMember
	// check if users exists and roles are valid
	for i, user := range memberUsers {
		userData, err := ops.GetUserData(user.MemberID, "")
		if userData.UserID == "" {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400) // change status code
			return c.RenderJSON(result)
		}
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error()) // change status code
			return c.RenderJSON(result)
		}
		memberUsers[i].GroupRole = strings.ToUpper(user.GroupRole)
		departmentMembers = append(departmentMembers, models.DepartmentMember{
			PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
			SK:           utils.AppendPrefix(constants.PREFIX_USER, userData.UserID),
			DepartmentID: departmentID,
			UserID:       userData.UserID,
		})
	}

	// INSERT DEPARTMENT MEMBERS START
	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptAddUsersErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptAddUsersErr != nil {
		result["departmentError"] = "Error adding users to department"
	}
	// INSERT DEPARTMENT MEMBERS END

	err := c.AddGroupMembers(companyID, constants.MEMBER_TYPE_USER, group, memberUsers, true)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(result)
}

/*
****************
AddGroupsAsMembers
- Add group member to a given group (nested)
Body:
group_id <string> - (required)
groups <[]string> -  array of group id's, eg: [{"ItemID": <id>}]
****************
*/

// @Summary Add Groups As Members
// @Description This endpoint adds individuals as members to a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddGroupsAsMembersRequest true "Add groups as members body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/add_members/group [post]
func (c GroupController) AddGroupsAsMembers() revel.Result {
	result := make(map[string]interface{})

	groupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id")
	groups := c.Params.Form.Get("groups")
	departmentID := c.Params.Form.Get("department_id")

	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.ADD_GROUP_MEMBER, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	group, errGetGroup := GetGroupByID(groupID)
	if errGetGroup != nil {
		result["status"] = constants.HTTP_STATUS[errGetGroup.Error()]
		return c.RenderJSON(result)
	}

	var memberGroups []models.GroupMember
	json.Unmarshal([]byte(groups), &memberGroups)

	if len(memberGroups) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	var departmentMembers []models.DepartmentMember
	for _, group := range memberGroups {
		for _, user := range group.GroupMembers {
			userData, err := ops.GetUserData(user.MemberID, "")

			if userData.UserID == "" {
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400) // change status code
				return c.RenderJSON(result)
			}
			if err != nil {
				result["status"] = utils.GetHTTPStatus(err.Error()) // change status code
				return c.RenderJSON(result)
			}

			departmentMembers = append(departmentMembers, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, userData.UserID),
				DepartmentID: departmentID,
				UserID:       userData.UserID,
			})
		}
	}

	// INSERT DEPARTMENT MEMBERS START

	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembers)
	deptAddUsersErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptAddUsersErr != nil {
		result["departmentError"] = "Error adding users to department"
	}
	// INSERT DEPARTMENT MEMBERS END

	err := c.AddGroupMembers(companyID, constants.MEMBER_TYPE_GROUP, group, memberGroups, true)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// generate log
	var logInfoGroups []models.LogModuleParams
	for _, member := range memberGroups {
		logInfoGroups = append(logInfoGroups, models.LogModuleParams{
			ID:   member.GroupID,
			Name: member.Name,
		})
	}

	companyLog := models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Groups: logInfoGroups,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Groups: logInfoGroups,
			Group: &models.LogModuleParams{
				ID: groupID,
			},
		},
	}

	var errorMessages []string

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating company logs")
	}
	_ = companyLogID

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating group logs")
	}
	_ = groupLogID

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["groupID"] = groupID
	result["memberGroups"] = memberGroups

	return c.RenderJSON(result)
}

/*
****************
BranchGroup
- create a new branch of a group, members can be copy or moved
Params:
groupID - PK
Body:
group_name - required
member_users, member_groups - {"copy":[{"MemberID": <ID>}], "move":[{"PK": <PK>, "SK": <SK>}]}
****************
*/

// @Summary Branch Group
// @Description This endpoint creates a branch of a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/branch/:groupID [post]
func (c GroupController) BranchGroup(groupID string) revel.Result {
	result := make(map[string]interface{})

	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.BRANCH_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	groupInfo, err := GetGroupByID(groupID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// get form params
	companyID := c.Params.Form.Get("company_id")
	departmentID := c.Params.Form.Get("department_id")
	groupName := c.Params.Form.Get("group_name")
	groupDescription := c.Params.Form.Get("group_description")
	memberUsers := c.Params.Form.Get("member_users")
	memberGroups := c.Params.Form.Get("member_groups")
	groupEmail := c.Params.Form.Get("group_email_with_domain")

	groupUUID := utils.GenerateTimestampWithUID()

	// departmentID := tmp.DepartmentID

	group := models.Group{
		PK:               utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
		SK:               utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
		GroupID:          groupUUID,
		GSI_SK:           utils.AppendPrefix(constants.PREFIX_GROUP, strings.ToLower(groupName)),
		CompanyID:        companyID,
		DepartmentID:     departmentID,
		GroupName:        groupName,
		GroupDescription: groupDescription,
		GroupColor:       utils.GetRandomColor(),
		Status:           constants.ITEM_STATUS_ACTIVE,
		Type:             constants.ENTITY_TYPE_GROUP,
		NewGroup:         constants.BOOL_TRUE,
		CreatedAt:        utils.GetCurrentTimestamp(),
		SearchKey:        strings.ToLower(groupName),
	}
	if groupEmail != "" {
		group.AssociatedAccounts = map[string][]string{
			"google": []string{groupEmail},
		}
	}
	// CHECK IF GROUPNAME IS UNIQUE
	uniqueGroup := IsGroupNameUnique(group.CompanyID, group.SearchKey)
	if uniqueGroup {
		//result["status"] = result
		result["entered group name"] = group.GroupName
		result["message"] = "Group name has already been taken. Please pick another group name."
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// validate form
	group.Validate(c.Validation, constants.USER_PERMISSION_ADD_GROUP)
	if c.Validation.HasErrors() {
		result["errors"] = c.Validation.Errors
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	var users, groups BranchMembers
	json.Unmarshal([]byte(memberUsers), &users)
	json.Unmarshal([]byte(memberGroups), &groups)

	// check members to be moved if exists
	var moveMemberUsers []models.GroupMember
	var moveMemberGroups []models.GroupMember
	var departmentMembersToAdd []models.DepartmentMember
	var departmentMembersToRemove []models.DepartmentMember

	if len(users.Move) != 0 {
		for _, user := range users.Move {
			groupMember, err := GetGroupMember(user.PK, user.SK)
			if err != nil {
				result["message"] = err
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
				return c.RenderJSON(result)
			}
			moveMemberUsers = append(moveMemberUsers, groupMember)
			departmentMembersToAdd = append(departmentMembersToAdd, models.DepartmentMember{
				PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
				SK:           utils.AppendPrefix(constants.PREFIX_USER, groupMember.MemberID),
				DepartmentID: departmentID,
				UserID:       groupMember.MemberID,
			})

			if groupInfo.DepartmentID != departmentID {
				// check if member belongs to other Group of the same department
				userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), groupMember.MemberID)
				deptCountOccurrence := 0 // GroupMember.DepartmentID matched to other Group.DepartmentID
				if err != nil {
					result["status"] = utils.GetHTTPStatus(err.Error())
					result["message"] = "Something went wrong fetching group member's groups"
					return c.RenderJSON(result)
				}
				for _, userGroup := range userGroupsList {
					if userGroup.DepartmentID == groupInfo.DepartmentID && userGroup.GroupID != groupID {
						deptCountOccurrence++
					}
				}
				if deptCountOccurrence == 0 {
					// User doesn't exist to other Groups of the same Department
					departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
						PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, groupInfo.DepartmentID),
						SK:           utils.AppendPrefix(constants.PREFIX_USER, groupMember.MemberID),
						DepartmentID: groupInfo.DepartmentID,
						UserID:       groupMember.MemberID,
					})
				}
			}
		}
	}

	if len(groups.Move) != 0 {
		for _, group := range groups.Move {
			subGroup, err := GetGroupMember(group.PK, group.SK)
			if err != nil {
				result["message"] = err
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
				return c.RenderJSON(result)
			}
			moveMemberGroups = append(moveMemberGroups, subGroup)
		}
	}

	if len(users.Copy) != 0 {
		for _, user := range users.Copy {
			groupMember, err := GetGroupMember(user.PK, user.SK)
			if err != nil {
				result["message"] = err
				result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
				return c.RenderJSON(result)
			}
			if departmentID != groupMember.DepartmentID {
				departmentMembersToAdd = append(departmentMembersToAdd, models.DepartmentMember{
					PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID),
					SK:           utils.AppendPrefix(constants.PREFIX_USER, groupMember.MemberID),
					DepartmentID: departmentID,
					UserID:       groupMember.MemberID,
				})
			}
		}
	}

	// insert branched group
	group, cID, err := c.AddGroup(companyID, group, groups.Copy, users.Copy)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}
	log.Println("cID: ", cID)

	// update group id of users to be moved
	if len(moveMemberUsers) != 0 {
		err = c.UpdateGroupMembersGroupID(group, moveMemberUsers, companyID, constants.MEMBER_TYPE_USER)
		if err != nil {
			// result["message"] = "Error while moving member users."
			result["message"] = err.Error()
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	// update group id of groups to be moved
	if len(moveMemberGroups) != 0 {
		err = c.UpdateGroupMembersGroupID(group, moveMemberGroups, companyID, constants.MEMBER_TYPE_GROUP)
		if err != nil {
			result["message"] = "Error while moving member groups."
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	_, err = CloneIntegrations(groupID, groupUUID, companyID, []string{}, true)
	if err != nil {
		result["message"] = "ERROR WHILE CLONING INTEGRATIONS"
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// INSERT DEPARTMENT MEMBERS START
	filteredDepartmentMembers := FilterDuplicatedDepartmentMembers(departmentMembersToAdd)

	deptUserAddErr := AddDepartmentUsers(filteredDepartmentMembers, departmentID)
	if deptUserAddErr != nil {
		result["message"] = "Error while adding users to a department."
	}
	// INSERT DEPARTMENT MEMBERS END

	// REMOVE DEPARTMENT MEMBERS START
	if len(departmentMembersToRemove) != 0 {
		filteredDepartmentMembersToRemove := FilterDuplicatedDepartmentMembers(departmentMembersToRemove)
		deptUserRemoveErr := RemoveDepartmentUsers(filteredDepartmentMembersToRemove)
		if deptUserRemoveErr != nil {
			result["message"] = "Error while removing users from a department."
		}
	}

	// REMOVE DEPARTMENT MEMBERS END

	// generate log
	var logs []models.Logs
	// message: GroupX has been branched to GroupY

	var newMembers []models.LogModuleParams
	for _, m := range moveMemberUsers {
		newMembers = append(newMembers, models.LogModuleParams{
			ID:   m.MemberID,
			Type: m.MemberType,
		})
	}

	logs = append(logs, models.Logs{
		CompanyID: companyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_BRANCH_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Origin: &models.LogModuleParams{
				ID: groupID,
			},
			Group: &models.LogModuleParams{
				ID:        groupUUID,
				GroupName: groupName,
			},
			Members: newMembers,
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["message"] = "error while creating logs"
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["group"] = group
	// result["logID"] = logID

	return c.RenderJSON(result)
}

/*
****************
AddBookmarkGroup
URL: PUT /v1/groups/:groupID
Params:
- groupID <string>
Body:
departmentID <string> - if the department id change, move the group
groupName <string>
groupDescription, groupStatus <string> - optional
****************
*/

// @Summary Create Group Bookmark
// @Description This endpoint allows a user to bookmark a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.AddBookmarkGroupRequest true "Add bookmark group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/bookmark/add [post]
func (c GroupController) AddBookmarkGroup() revel.Result {
	result := make(map[string]interface{})

	// params
	groupID := c.Params.Form.Get("group_id")
	departmentID := c.Params.Form.Get("department_id")
	userID := c.ViewArgs["userID"].(string)

	//group model
	group := models.Group{}

	// check if group exist
	fetch, err := ops.GetGroupByID(groupID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(fetch)
	}

	//if no data in fetch
	if len(fetch.Items) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	//binding data to &group
	err = dynamodbattribute.UnmarshalMap(fetch.Items[0], &group)
	if err != nil {
		result["message"] = "Something went wrong with unmarshalmap"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	//SET NEW = FALSE
	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(group.PK),
			},
			"SK": {
				S: aws.String(group.SK),
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":bf": {
				S: aws.String(constants.BOOL_FALSE),
			},
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET NewGroup = :bf, UpdatedAt = :ua"),
	}

	//inserting query
	_, err = app.SVC.UpdateItem(input)
	//return 500 if invalid
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// check if userID exists
	u, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		result["message"] = "Error at Getting users"
		result["status"] = utils.GetHTTPStatus(opsError.Status.Code)
		return c.RenderJSON(result)
	}

	gID := &dynamodb.AttributeValue{
		S: aws.String(groupID),
	}

	var listOfGroups []*dynamodb.AttributeValue
	listOfGroups = append(listOfGroups, gID)
	if len(u.BookmarkGroups) != 0 {
		for _, bg := range u.BookmarkGroups {
			listOfGroups = append(listOfGroups, &dynamodb.AttributeValue{
				S: aws.String(bg),
			})
		}
	}

	// Prepare update query
	updateInput := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(u.PK)},
			"SK": {S: aws.String(u.SK)},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":bm": {L: listOfGroups},
			":ua": {S: aws.String(utils.GetCurrentTimestamp())},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET BookmarkGroups = :bm, UpdatedAt = :ua"),
	}

	//inserting query
	_, err1 := app.SVC.UpdateItem(updateInput)
	//return 500 if invalid
	if err1 != nil {
		result["message"] = "Error at inputting: " + err1.Error()
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	//preparing updatedat groups
	gInput := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID)),
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":ua": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET UpdatedAt = :ua"),
	}

	//inserting query
	_, err = app.SVC.UpdateItem(gInput)
	//return 500 if invalid
	if err != nil {
		result["message"] = "Error at inputting"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// // generate log
	// var logs []models.Logs
	// // message: GroupX has been bookmarked by UserX
	// logs = append(logs, models.Logs{
	// 	CompanyID: u.ActiveCompany,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_ADD_BOOKMARK_GROUP,
	// 	LogType:   constants.ENTITY_TYPE_GROUP,
	// 	LogInfo: &models.LogInformation{
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["message"] = "error while creating logs"
	// }

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
DeleteBookmarkGroup
****************
*/

// @Summary Delete Group Bookmark
// @Description This endpoint allows a user to remove a bookmark from a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.DeleteBookmarkGroupRequest true "Delete bookmark group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/bookmark/delete [post]
func (c GroupController) DeleteBookmarkGroup() revel.Result {
	result := make(map[string]interface{})

	// params
	groupID := c.Params.Form.Get("group_id")
	userID := c.ViewArgs["userID"].(string)

	// check if userID exists
	u, opsError := ops.GetUserByID(userID)
	if opsError != nil {
		result["message"] = "Error at Getting users"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_461)
		return c.RenderJSON(result)
	}

	//finding index of groupID
	var index string

	for idx, item := range u.BookmarkGroups {
		if item == groupID {
			//converting type int to string
			index = strconv.Itoa(idx)
		}
	}

	if index == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_474)
		return c.RenderJSON(result)
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
		UpdateExpression: aws.String("REMOVE BookmarkGroups[" + index + "]"),
	}

	//executing query
	_, err := app.SVC.UpdateItem(input)

	//return 500 if errors
	if err != nil {
		result["message"] = "Error at inputting"
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// // generate log
	// var logs []models.Logs
	// // message: GroupX has been un-bookmarked by UserX
	// logs = append(logs, models.Logs{
	// 	CompanyID: u.ActiveCompany,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_DELETE_BOOKMARK_GROUP,
	// 	LogType:   constants.ENTITY_TYPE_GROUP,
	// 	LogInfo: &models.LogInformation{
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 		User: &models.LogModuleParams{
	// 			ID: userID,
	// 		},
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["message"] = "error while creating logs"
	// }

	// result["exp"] = expressionString
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
UpdateGroup
URL: PUT /v1/groups/:groupID
Params:
- groupID <string>
Body:
departmentID <string> - if the department id change, move the group
groupName <string>
groupDescription, groupStatus <string> - optional
****************
*/

// @Summary Update Group
// @Description This endpoint updates a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param body body models.UpdateGroupRequest true "Update group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID [put]
func (c GroupController) UpdateGroup(groupID string) revel.Result {
	result := make(map[string]interface{})

	// params
	departmentID := c.Params.Form.Get("department_id")
	groupName := utils.TrimSpaces(c.Params.Form.Get("group_name"))
	groupColor := c.Params.Form.Get("group_color")
	groupDescription := utils.TrimSpaces(c.Params.Form.Get("group_description")) // optional
	groupStatus := c.Params.Form.Get("group_status")                             // optional
	userID := c.ViewArgs["userID"].(string)
	groupNameChangeStr := c.Params.Form.Get("group_name_change")

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.EDIT_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}
	// items, err := GetGroupsByGroupName(groupName, c.ViewArgs["companyID"].(string))
	// if err != nil {
	// 	result["message"] = "Something went wrong with getting groups"
	// 	result["status"] = utils.GetHTTPStatus(err.Error())
	// 	return c.RenderJSON(result)
	// }

	// err = dynamodbattribute.UnmarshalListOfMaps(items, &groups)
	// if err != nil {
	// 	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
	// 	return c.RenderJSON(result)
	// }

	// var groupsNameList []string
	// for _, group := range groups {
	// 	groupsNameList = append(groupsNameList, group.GroupName)
	// }

	groupsNameList, err := ops.GetGroupsByCompany(c.ViewArgs["companyID"].(string))
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// Separated method for generating a unique group name to avoid conflict
	newUniqueGroupName := groupName

	groupNameChange, err := strconv.ParseBool(groupNameChangeStr)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	if groupNameChange {
		newUniqueGroupName = utils.MakeGroupNameUnique(groupName, groupsNameList)
	}

	// check if groupID exists
	g, err := GetGroupByID(groupID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}
	// var newUniqueGroupName string
	// if groupName == g.GroupName {
	// 	newUniqueGroupName = groupName
	// } else {
	// 	newUniqueGroupName = utils.MakeUsernameUnique(groupName, groupsNameList)

	// }
	// validate form
	group := models.Group{
		GroupName:        newUniqueGroupName,
		GroupColor:       groupColor,
		GroupDescription: groupDescription,
		Status:           groupStatus,
		UpdatedAt:        utils.GetCurrentTimestamp(),
		SearchKey:        strings.ToLower(groupName),
		GSI_SK:           utils.AppendPrefix(constants.PREFIX_GROUP, strings.ToLower(groupName)),
	}

	group.Validate(c.Validation, "UPDATE")
	if c.Validation.HasErrors() {
		result["error"] = c.Validation.Errors
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	// set default optional values, (can be done on frontend, just to handle the empty values on postman)
	if groupStatus == "" {
		group.Status = g.Status
	}

	// check if group name is unique
	// if g.GroupName != groupName {
	// 	isUnique, err := IsGroupNameUnique(g.DepartmentID, groupName)
	// 	if !isUnique || err != nil {
	// 		result["status"] = utils.GetHTTPStatus(err.Error())
	// 		return c.RenderJSON(result)
	// 	}
	// }

	// check if departmentID change and if it has value -> if yes move group to that department by deleting and inserting
	if g.DepartmentID == departmentID || departmentID == "" {
		// update
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":gn": {
					S: aws.String(group.GroupName),
				},
				":gc": {
					S: aws.String(group.GroupColor),
				},
				":gd": {
					S: aws.String(group.GroupDescription),
				},
				":gs": {
					S: aws.String(group.Status),
				},
				":sk": {
					S: aws.String(group.SearchKey),
				},
				":gsisk": {
					S: aws.String(group.GSI_SK),
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
					S: aws.String(g.PK),
				},
				"SK": {
					S: aws.String(g.SK),
				},
			},
			UpdateExpression: aws.String("SET GroupName = :gn, GroupColor = :gc, GroupDescription = :gd, #s = :gs, SearchKey = :sk, GSI_SK = :gsisk, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		// todo create log

	} else {
		// check if department exists
		deptExists := CheckIfDepartmentExists(departmentID, c.Validation)
		if !deptExists {
			result["status"] = constants.HTTP_STATUS[constants.HTTP_STATUS_470]
			return c.RenderJSON(result)
		}

		// check if group name already exists
		// isUnique, err := IsGroupNameUnique(departmentID, groupName)
		// if !isUnique || err != nil {
		// 	result["status"] = utils.GetHTTPStatus(err.Error())
		// 	return c.RenderJSON(result)
		// }
		// delete
		_, err = ops.DeleteByPartitionKey(g.PK, g.SK)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		// insert
		group.DepartmentID = departmentID
		group.CreatedAt = utils.GetCurrentTimestamp()
		group.UpdatedAt = ""

		av, err := dynamodbattribute.MarshalMap(group)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			result["error"] = err.Error()
			return c.RenderJSON(result)
		}

		putInput := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err = app.SVC.PutItem(putInput)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			result["error"] = err.Error()
			return c.RenderJSON(result)
		}

		// todo create log
	}

	// generate log
	var logs []models.Logs
	// message: GroupX has been renamed to GroupY
	logs = append(logs, models.Logs{
		CompanyID: g.CompanyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_UPDATE_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
				// GroupName: groupName,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
			// RenamedGroup: &models.LogModuleParams {
			// 	ID: groupID,
			// 	GroupName: group.GroupName,
			// },
		},
	})
	_, err = CreateBatchLog(logs)
	if err != nil {
		result["message"] = "error while creating logs"
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
UpdateGroupMember
PATCH    /v1/groups/:groupID/people/:userID
Body:
action - values: UPDATE and DELETE, CLONE (required)
role - values: ADMIN, MEMBER, VIEWER (required if action = UPDATE or clone)
****************
*/

// @Summary Update Group Member
// @Description This endpoint updates a member within a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param itemID path string true "Item ID"
// @Param body body models.UpdateGroupMemberRequest true "Update group member body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/member/:itemID [put]
func (c GroupController) UpdateGroupMember(groupID, itemID string) revel.Result {
	result := make(map[string]interface{})

	memberType := c.Params.Form.Get("member_type")
	role := c.Params.Form.Get("role")
	requestGroupID := c.Params.Form.Get("group_id")
	companyID := c.Params.Form.Get("company_id") // new
	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.EDIT_GROUP, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	// check if member type valid, if user check if role is not empty
	memberType = strings.ToUpper(memberType)
	if !utils.StringInSlice(memberType, constraints.MEMBER_TYPES) {
		result["message"] = "Invalid member type." // change to constant
		return c.RenderJSON(result)
	}

	if memberType == constants.MEMBER_TYPE_GROUP && role != "" {
		result["message"] = "Group member do not accept role parameter."
		return c.RenderJSON(result)
	}

	if memberType == constants.MEMBER_TYPE_USER {
		if !utils.StringInSlice(strings.ToUpper(role), constraints.GROUP_ROLES) {
			result["status"] = constants.HTTP_STATUS[constants.HTTP_STATUS_465]
			return c.RenderJSON(result)
		}
	}

	// check if user exists on the group
	member, err := GetGroupMember(utils.AppendPrefix(constants.PREFIX_GROUP, groupID), (memberType + "#" + itemID)) // refactor
	if err != nil {
		result["message"] = err.Error()
		return c.RenderJSON(result)
	}

	// if requestGroupID has a value and not equal to groupID, -> update memebr group id
	var action string
	if requestGroupID == "" || groupID == requestGroupID {
		action = constants.ACTION_UPDATE
	} else {
		action = constants.ACTION_MOVE
	}

	switch action {
	case constants.ACTION_UPDATE:
		if memberType == constants.MEMBER_TYPE_GROUP {
			break
		}

		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":gr": {
					S: aws.String(strings.ToUpper(role)),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(member.PK),
				},
				"SK": {
					S: aws.String(member.SK),
				},
			},
			UpdateExpression: aws.String("SET GroupRole = :gr, UpdatedAt = :ua"),
		}

		_, err := app.SVC.UpdateItem(input)
		if err != nil {
			result["error"] = err.Error()
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

	case constants.ACTION_MOVE:
		// check if request group id exists
		group, err := GetGroupByID(requestGroupID)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}

		// check if the member already exists
		groupMember, err := GetGroupMember(group.PK, member.SK)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		if groupMember.GroupID != "" {
			result["message"] = "Member already exists on the group."
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		requestGroupMembers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), requestGroupID, constants.MEMBER_TYPE_GROUP, c.Controller)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}

		isGroupParentAMember := false
		for _, m := range requestGroupMembers {
			if m.MemberID == groupID {
				isGroupParentAMember = true
			}
		}

		if isGroupParentAMember {
			result["message"] = "Parent group member already exists on the group."
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		itemGroupMembers, err := GetGroupMembers(c.ViewArgs["companyID"].(string), itemID, constants.MEMBER_TYPE_GROUP, c.Controller)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}

		isRequestedGroupIDValid := false
		for _, m := range itemGroupMembers {
			if m.MemberID == requestGroupID {
				isRequestedGroupIDValid = true
			}
		}

		if isRequestedGroupIDValid {
			result["message"] = "Can't move a group member to a group that it is currently under of it."
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		// set updated role of the user member
		if memberType == constants.MEMBER_TYPE_USER {
			member.GroupRole = strings.ToUpper(role)
		}

		var members []models.GroupMember
		members = append(members, member)
		err = c.UpdateGroupMembersGroupID(group, members, companyID, memberType)
		if err != nil {
			result["message"] = err
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
DeleteGroupMembers
- Remove group members by their primary keys
Params:
groupID <string> - Group to be removed it's members
Body:
member <[]GroupMember> - array of group members, eg: [{"PK": <pk>, "SK": <sk>}]
****************
*/

// @Summary Delete Group Members
// @Description This endpoint deletes members from a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param body body models.DeleteGroupMembersRequest true "Delete group members body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/group_members [delete]
func (c GroupController) DeleteGroupMembers(groupID string) revel.Result {
	result := make(map[string]interface{})

	members := c.Params.Form.Get("members")

	var groupMembers []models.GroupMember
	json.Unmarshal([]byte(members), &groupMembers)

	userID := c.ViewArgs["userID"].(string)

	//CHECKED PERMISSION FOR ADDING GROUP
	checked := ops.CheckPermissions(constants.REMOVE_GROUP_MEMBER, userID, c.ViewArgs["companyID"].(string))
	//if !checked RETURNED TRUE - ERROR APPLIES
	if !checked {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
		return c.RenderJSON(result)
	}

	if len(groupMembers) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	// check if group id exists
	group, err := GetGroupByID(groupID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}
	_ = group

	var departmentMembersToRemove []models.DepartmentMember
	// check if group has a members of
	groupMembersExists := true
	for _, member := range groupMembers {
		m, err := GetGroupMember(member.PK, member.SK)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(result)
		}
		if m.PK == "" {
			groupMembersExists = false
		}

		// prepare department users to remove
		// member = subgroup
		groupMemberList, err := ops.GetGroupMembersOld(member.GroupID, c.Controller)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(err.Error())
			result["message"] = "Something went wrong fetching group members"
			return c.RenderJSON(result)
		}
		if len(groupMemberList) != 0 {
			for _, grpMember := range groupMemberList {
				if grpMember.MemberType == constants.MEMBER_TYPE_USER {
					// check if member belongs to other Group of the same department
					userGroupsList, err := GetUserCompanyGroups(c.ViewArgs["companyID"].(string), grpMember.MemberID)
					deptCountOccurrence := 0 // member.DepartmentID matched to other userGroup.DepartmentID
					if err != nil {
						result["status"] = utils.GetHTTPStatus(err.Error())
						result["message"] = "Something went wrong fetching group member's groups"
						return c.RenderJSON(result)
					}
					for _, userGroup := range userGroupsList {
						if userGroup.DepartmentID == group.DepartmentID && userGroup.GroupID != groupID {
							deptCountOccurrence++
						}
					}
					if deptCountOccurrence == 0 {
						// User doesn't exist to other Groups of the same Department
						departmentMembersToRemove = append(departmentMembersToRemove, models.DepartmentMember{
							PK:           utils.AppendPrefix(constants.PREFIX_DEPARTMENT, group.DepartmentID),
							SK:           utils.AppendPrefix(constants.PREFIX_USER, grpMember.MemberID),
							DepartmentID: group.DepartmentID,
							UserID:       grpMember.MemberID,
						})
					}
				}
			}
		}
	}

	if !groupMembersExists {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}

	err = RemoveGroupMembers("", groupMembers, models.Group{})
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// REMOVE DEPARTMENT MEMBERS START
	if len(departmentMembersToRemove) != 0 {
		var filteredMembersToRemove []models.DepartmentMember
		for _, deptMember := range departmentMembersToRemove {
			skip := false
			for _, u := range filteredMembersToRemove {
				if deptMember.PK == u.PK && deptMember.SK == u.SK {
					skip = true
					break
				}
			}
			if !skip {
				filteredMembersToRemove = append(filteredMembersToRemove, deptMember)
			}
		}

		deptUserRemoveErr := RemoveDepartmentUsers(filteredMembersToRemove)
		if deptUserRemoveErr != nil {
			result["message"] = "Error while removing users from a department."
		}
	}
	// REMOVE DEPARTMENT MEMBERS END

	// generate log
	// var logs []models.Logs
	// message: SubGroupX has been deleted by UserX
	var deletedMembers []models.LogModuleParams
	for _, m := range groupMembers {
		deletedMembers = append(deletedMembers, models.LogModuleParams{
			ID:   m.MemberID,
			Type: m.MemberType,
		})
	}
	// logs = append(logs, models.Logs{
	// 	CompanyID: group.CompanyID,
	// 	UserID:    c.ViewArgs["userID"].(string),
	// 	LogAction: constants.LOG_ACTION_REMOVE_GROUP_MEMBERS,
	// 	LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
	// 	LogInfo: &models.LogInformation{
	// 		Group: &models.LogModuleParams{
	// 			ID: groupID,
	// 		},
	// 		Members: deletedMembers,
	// 		// User: &models.LogModuleParams {
	// 		// 	ID: c.ViewArgs["userID"].(string),
	// 		// },
	// 	},
	// })
	// _, err = CreateBatchLog(logs)
	// if err != nil {
	// 	result["message"] = "error while creating logs"
	// }

	companyLog := models.Logs{
		CompanyID: c.ViewArgs["companyID"].(string),
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			Members: deletedMembers,
		},
	}

	groupLog := models.Logs{
		GroupID:   groupID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_REMOVE_GROUP_MEMBERS,
		LogType:   constants.ENTITY_TYPE_GROUP_MEMBER,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: groupID,
			},
			Members: deletedMembers,
		},
	}

	var errorMessages []string

	companyLogID, err := ops.InsertLog(companyLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating company logs")
	}
	_ = companyLogID

	groupLogID, err := ops.InsertLog(groupLog)
	if err != nil {
		errorMessages = append(errorMessages, "Error while creating group logs")
	}
	_ = groupLogID

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
GetGroupMember
- Returns item on GROUP_MEMBER entity
- Used for checking if a member exists on a group
Params:
pk, sk <string> - primary keys
****************
*/
func GetGroupMember(pk, sk string) (models.GroupMember, error) {
	result, err := app.SVC.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pk),
			},
			"SK": {
				S: aws.String(sk),
			},
		},
	})
	if err != nil {
		e := errors.New(err.Error())
		return models.GroupMember{}, e
	}

	if result.Item == nil {
		return models.GroupMember{}, nil
	}

	item := models.GroupMember{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		e := errors.New(err.Error())
		return models.GroupMember{}, e
	}

	return item, nil
}

/*
****************
UpdateGroupMembersGroupID
- To update a members group id, item(s) should be deleted first then insert a new ones with the new group id attached
- Used for merging and branching of groups
Params:
groupID, groupName string
members - array of struct { PK, SK string }, eg: [{"PK": "GROUP#12345", "SK": "USER#12345"}]
****************
*/
func (c GroupController) UpdateGroupMembersGroupID(group models.Group, members []models.GroupMember, companyID, memberType string) error {
	err := RemoveGroupMembers(memberType, members, models.Group{})
	if err != nil {
		return errors.New("error while deleting group members")
	}

	filteredMemberUsers := FilteredDuplicateMembers(members)
	err = c.AddGroupMembers(companyID, memberType, group, filteredMemberUsers, false)
	if err != nil {
		return errors.New("error while adding group members")
	}

	return nil
}

/*
****************
FilteredDuplicateMembers
- Removes duplicate group members entry before inserting
- If the duplicate entry has a group role, the higher role will be selected
Params:
members []GroupMember
****************
*/
func FilteredDuplicateMembers(members []models.GroupMember) []models.GroupMember {
	var unique []models.GroupMember
	for _, item := range members {
		skip := false
		for j, u := range unique {
			if item.SK == u.SK {
				if item.GroupRole != u.GroupRole {
					unique[j].GroupRole = GetHigherGroupRole(item.GroupRole, u.GroupRole)
				}
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, item)
		}
	}
	return unique
}

func FilteredDuplicateGroupMembers(members []models.GroupMember) []models.GroupMember {
	var unique []models.GroupMember
	for _, item := range members {
		skip := false
		for _, u := range unique {
			if item.MemberID == u.MemberID {
				// if item.GroupRole != u.GroupRole {
				// 	unique[j].GroupRole = GetHigherGroupRole(item.GroupRole, u.GroupRole)
				// }
				skip = true
				break
			}
		}
		if !skip {

			unique = append(unique, item)
		}
	}
	return unique
}

/*
****************
FilterDuplicateIntegrations
- Removes duplicate integrations
Params:
integrations []GroupIntegration
****************
*/
func FilterDuplicateIntegrations(integrations []models.GroupIntegration) []models.GroupIntegration {
	var unique []models.GroupIntegration
	for _, item := range integrations {
		skip := false
		for _, u := range unique {
			if item.IntegrationID == u.IntegrationID {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, item)
		}
	}
	return unique
}

/*
****************
GetHigherGroupRole
- Return the higher role of two different group role
Params:
role1 string, role2 string - roles to be compared
TODO: refactor
****************
*/
func GetHigherGroupRole(role1, role2 string) string {
	roles := make(map[string]int)
	roles["ADMIN"] = 3
	roles["MEMBER"] = 2
	roles["VIEWER"] = 1

	if roles[role1] < roles[role2] {
		return role2
	} else {
		return role1
	}
}

// Refactored version of GetGroupByID()
func GetCompanyGroupNew(companyID, groupID string) (models.Group, *models.ErrorResponse) {
	group := models.Group{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
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
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_GROUP),
					},
				},
			},
		},
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return group, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        fmt.Sprintf("Error getting group in a company: %s", err.Error()),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}

	if len(result.Items) == 0 {
		return group, &models.ErrorResponse{
			HTTPStatusCode: 404,
			Message:        fmt.Sprintf("Group not found in the company."),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_404),
		}
	}

	err = dynamodbattribute.UnmarshalMap(result.Items[0], &group)
	if err != nil {
		return group, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        fmt.Sprintf("Error unmarshalling group: %s", err.Error()),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}

	return group, nil
}
func GetCompanyGroupWithFilter(companyID, groupID, departmentId, searchKey string) (models.Group, *models.ErrorResponse) {
	group := models.Group{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupID)),
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
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_GROUP),
					},
				},
			},
		},
	}

	if searchKey != "" {
		params.QueryFilter = map[string]*dynamodb.Condition{
			"SearchKey": {
				ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(searchKey),
					},
				}},
		}
	}
	if departmentId != "" {
		params.QueryFilter = map[string]*dynamodb.Condition{
			"DepartmentID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(departmentId),
					},
				}},
		}
	}
	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return group, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        fmt.Sprintf("Error getting group in a company: %s", err.Error()),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}

	if len(result.Items) == 0 {
		return group, nil
	}

	err = dynamodbattribute.UnmarshalMap(result.Items[0], &group)
	if err != nil {
		return group, &models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        fmt.Sprintf("Error unmarshalling group: %s", err.Error()),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		}
	}

	return group, nil
}

/*
****************
GetGroupByID
- Get group details
- Can be used for checking if group exists on a department
Params:
groupID <string>
****************
*/
func GetGroupByID(groupID string) (models.Group, error) {
	group := models.Group{}

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
						S: aws.String(constants.PREFIX_DEPARTMENT),
					},
				},
			},
		},
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return group, e
	}

	if len(result.Items) == 0 {
		e := errors.New(constants.HTTP_STATUS_404)
		return group, e
	}

	err = dynamodbattribute.UnmarshalMap(result.Items[0], &group)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return group, e
	}

	return group, nil
}

/*
****************
GetParentGroup
- Get the parent group of specific group
- Used for checking before merging two groups
Params:
groupID string
****************
*/
func GetParentGroup(groupID string) (models.Group, error) {
	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_GROUP + groupID),
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
		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return models.Group{}, e
	}

	if len(result.Items) == 0 {
		return models.Group{}, nil
	}

	var group models.Group
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &group)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return models.Group{}, e
	}

	return group, nil
}

/*
****************
IsGroupNameUnique
- Returns true when group name already taken on the department
- Used to avoid group duplicated on a department
Params:
departmentID, groupName <string>
****************
*/
func IsGroupNameUnique(companyID, groupName string) bool {
	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_GROUP),
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
		QueryFilter: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"SearchKey": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(groupName),
					},
				},
			},
		},

		IndexName: aws.String(constants.INDEX_NAME_GET_GROUPS),
		TableName: aws.String(app.TABLE_NAME),
	}

	res, err := app.SVC.Query(params)
	if err != nil {
		return false
	}

	if len(res.Items) != 0 {
		return true
	} else {
		return false
	}
}

/*
****************
Get Members
Params:
departmentID
Can be used for checking if Group exists
****************
*/
func GetGroups(departmentID string, pageLimit int64, exclusiveStartKey, searchKey string) ([]models.Group, models.Group, error) {

	group := []models.Group{}
	lastEvaluatedKey := models.Group{}

	var params *dynamodb.QueryInput

	if searchKey == "" {
		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
			KeyConditions: map[string]*dynamodb.Condition{
				"SK": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID)),
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
			Limit: aws.Int64(pageLimit),
		}
	} else {
		//Expression declaration
		filt := expression.Name("SearchKey").Contains(searchKey)
		proj := expression.NamesList(
			expression.Name("GroupID"),
			expression.Name("GroupDescription"),
			expression.Name("DepartmentID"),
			expression.Name("GroupName"),
			expression.Name("Status"),
			expression.Name("CreatedAt"),
			expression.Name("UpdatedAt"),
		)

		//Expression Builder
		expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_400)
			return group, lastEvaluatedKey, e
		}

		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
			KeyConditions: map[string]*dynamodb.Condition{
				"SK": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(utils.AppendPrefix(constants.PREFIX_DEPARTMENT, departmentID)),
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
			Limit:                     aws.Int64(pageLimit),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			ExclusiveStartKey:         utils.MarshalLastEvaluatedKey(models.Group{}, exclusiveStartKey),
		}
	}

	result, err := app.SVC.Query(params)

	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return group, lastEvaluatedKey, e
	}

	if len(result.Items) == 0 {
		e := errors.New(constants.HTTP_STATUS_404)
		return group, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &group)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return group, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return group, lastEvaluatedKey, e
	}

	// group = append(group, lastEvaluatedKey)

	return group, lastEvaluatedKey, nil
}
func GetGroupsByGroupName(groupName, companyID string) ([]map[string]*dynamodb.AttributeValue, error) {
	var items []map[string]*dynamodb.AttributeValue
	result, err := GetGroupsQueryHandler(groupName, companyID, nil)
	if err != nil {
		e := errors.New(err.Error())
		return nil, e
	}
	items = append(items, result.Items...)
	key := result.LastEvaluatedKey

	for len(key) != 0 {
		result, err := GetGroupsQueryHandler(groupName, companyID, key)
		if err != nil {
			e := errors.New(err.Error())
			return nil, e
		}
		items = append(items, result.Items...)
		key = result.LastEvaluatedKey
	}

	return items, nil
}

/*
****************
GetGroupsQueryHandler
- get all group by group name
Params:
groupName, companyID <string>
****************
*/
func GetGroupsQueryHandler(groupName, companyID string, exclusiveStartKey map[string]*dynamodb.AttributeValue) (*dynamodb.QueryOutput, error) {
	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_GROUP),
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
		QueryFilter: map[string]*dynamodb.Condition{
			"CompanyID": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(companyID),
					},
				},
			},
			"GroupName": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(groupName),
					},
				},
			},
		},

		IndexName:         aws.String(constants.INDEX_NAME_GET_GROUPS),
		TableName:         aws.String(app.TABLE_NAME),
		ExclusiveStartKey: exclusiveStartKey,
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
GetGroupsQueryHandler
- get all group by group Id
Params:
group ID, companyID <string>
****************
*/
func GetGroupsByGroupID(groupId, companyId, searchKey, departmentId string) (models.Group, error) {
	groups := models.Group{}
	queryFilter := map[string]*dynamodb.Condition{
		"CompanyID": {
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(companyId),
				},
			},
		},
		"GroupID": {
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(groupId),
				},
			},
		},
	}
	if departmentId != "" {
		queryFilter["DepartmentID"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(departmentId),
				},
			},
		}
	}
	if searchKey != "" {
		queryFilter["SearchKey"] = &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(searchKey),
				},
			},
		}
	}
	params := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ENTITY_TYPE_GROUP),
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
		Limit:       aws.Int64(1),
		QueryFilter: queryFilter,
		IndexName:   aws.String(constants.INDEX_NAME_GET_GROUPS),
		TableName:   aws.String(app.TABLE_NAME),
	}

	result, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return groups, e
	}
	if len(result.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(result.Items[0], &groups)
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_500)
			return groups, e
		}
	}
	return groups, nil
}

/*
****************
Get All Members
Params:
groupID
Can be used when getting every single group
****************
*/
func GetAllGroups(entityType string, pageLimit int64, exclusiveStartKey, searchKey string) ([]models.Group, models.Group, error) {

	//Model
	group := []models.Group{}
	lastEvaluatedKey := models.Group{}

	//Input
	var params *dynamodb.QueryInput

	if searchKey == "" {
		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_GET_GROUPS),
			KeyConditions: map[string]*dynamodb.Condition{
				"Type": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(entityType),
						},
					},
				},
			},
			Limit: aws.Int64(pageLimit),
		}
	} else {
		//expression declaration
		filt := expression.Name("SearchKey").Contains(searchKey)
		proj := expression.NamesList(
			expression.Name("GroupID"),
			expression.Name("GroupDescription"),
			expression.Name("DepartmentID"),
			expression.Name("GroupName"),
			expression.Name("Status"),
			expression.Name("CreatedAt"),
			expression.Name("UpdatedAt"),
		)

		//expression builder
		expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_400)
			return nil, lastEvaluatedKey, e
		}

		params = &dynamodb.QueryInput{
			TableName: aws.String(app.TABLE_NAME),
			IndexName: aws.String(constants.INDEX_NAME_GET_GROUPS),
			KeyConditions: map[string]*dynamodb.Condition{
				"Type": {
					ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(entityType),
						},
					},
				},
			},
			Limit:                     aws.Int64(pageLimit),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			ExclusiveStartKey:         utils.MarshalLastEvaluatedKey(models.Group{}, exclusiveStartKey),
		}
	}

	result, err := app.SVC.Query(params)

	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return group, lastEvaluatedKey, e
	}

	if len(result.Items) == 0 {
		e := errors.New(constants.HTTP_STATUS_404)
		return group, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &group)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return group, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return group, lastEvaluatedKey, e
	}

	// group = append(group, lastEvaluatedKey)

	return group, lastEvaluatedKey, nil
}

/*
****************
CreateGroupLog
- Helper function for creating logs under group
****************
*/
func (c GroupController) CreateGroupLog(companyID string, details *models.LogModuleParams, logAction string) (string, error) {
	logCtrl := LogController{}
	authorID := c.ViewArgs["userID"].(string)
	logInfo := &models.LogInformation{
		// Action:      logAction,
		Group:       details,
		PerformedBy: authorID,
	}
	logID, err := logCtrl.CreateLog(authorID, companyID, logAction, constants.ENTITY_TYPE_GROUP, logInfo)
	if err != nil {
		e := errors.New(err.Error())
		return "", e
	}
	return logID, nil
}

/*
****************
SaveConnectedItems()
- save connected items in sub integration entity
****************
*/

// @Summary Save Connected Items
// @Description This endpoint saves connected items to a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param integration_id path string true "Integration ID"
// @Param group_id path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/sub_integrations [put]
func (c GroupController) SaveConnectedItems() revel.Result {
	result := make(map[string]interface{})

	integrationId := c.Params.Query.Get("integration_id") // sub integration id
	groupId := c.Params.Query.Get("group_id")

	companyID, ok := c.ViewArgs["companyID"].(string)
	if !ok {
		companyID = ""
	}

	// todo pass parent integration id?

	// get sub integration
	var subIntegration models.Integration
	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, integrationId)),
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

	if len(res.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(res.Items[0], &subIntegration)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}
	} else {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
		return c.RenderJSON(result)
	}

	// insert group sub integration w/ connected items
	items := c.Params.Form.Get("connected_items")
	var unmarshalItems []string
	json.Unmarshal([]byte(items), &unmarshalItems)

	// connected drive items sensitive data
	connectedDriveItems := c.Params.Form.Get("connected_drive_items")
	unmarshalDriveItems := []models.DriveItem{}
	json.Unmarshal([]byte(connectedDriveItems), &unmarshalDriveItems)

	item := &models.Integration{
		PK:                  utils.AppendPrefix(constants.PREFIX_GROUP, groupId),
		SK:                  utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, integrationId),
		GroupID:             groupId,
		IntegrationID:       integrationId,
		ParentIntegrationID: subIntegration.ParentIntegrationID,
		IntegrationSlug:     subIntegration.IntegrationSlug, // hard coded, update later
		ConnectedItems:      unmarshalItems,
		CompanyID:           companyID, // added for listing all connected integration in the company
		ConnectedDriveItems: unmarshalDriveItems,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		result["err"] = err
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	// check if group and integration connection exists
	groupIntegration, err := app.SVC.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(app.TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupId)),
			},
			"SK": {
				S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, subIntegration.ParentIntegrationID)),
			},
		},
	})

	var errorMessages []string

	if groupIntegration.Item == nil && err == nil {
		// insert group integration connection
		item = &models.Integration{
			PK:            utils.AppendPrefix(constants.PREFIX_GROUP, groupId),
			SK:            utils.AppendPrefix(constants.PREFIX_INTEGRATION, subIntegration.ParentIntegrationID),
			GroupID:       groupId,
			IntegrationID: subIntegration.ParentIntegrationID,
			CompanyID:     companyID,
		}

		av, err = dynamodbattribute.MarshalMap(item)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}

		input = &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err = app.SVC.PutItem(input)
		if err != nil {
			result["err"] = err
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		// create group activity & company log
		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_INTEGRATION,
			LogType:   constants.ENTITY_TYPE_GROUP,
			LogInfo: &models.LogInformation{
				Group: &models.LogModuleParams{
					ID: groupId,
				},
				Integration: &models.LogModuleParams{
					ID: subIntegration.ParentIntegrationID,
				},
			},
		}

		groupLog := models.Logs{
			GroupID:   groupId,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_INTEGRATION,
			LogType:   constants.ENTITY_TYPE_GROUP,
			LogInfo: &models.LogInformation{
				Group: &models.LogModuleParams{
					ID: groupId,
				},
				Integration: &models.LogModuleParams{
					ID: subIntegration.ParentIntegrationID,
				},
			},
		}

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating company logs for adding group and integration connection")
		}
		_ = companyLogID

		groupLogID, err := ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating group logs for adding group and integration connection")
		}
		_ = groupLogID
	}

	//check if the group sub applications if no connected items
	isRemoveGroupIntegration, err := CheckSubIntegrationsConnectedItemIsEmpty(groupId, subIntegration.ParentIntegrationID)
	if err != nil {
		result["from"] = "CheckSubIntegrationsConnectedItemIsEmpty"
		result["err"] = err
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	if isRemoveGroupIntegration {
		userID := c.ViewArgs["userID"].(string)
		companyID := c.ViewArgs["companyID"].(string)

		checked := ops.CheckPermissions(constants.REMOVE_GROUP_INTEGRATION, userID, companyID)
		//if !checked RETURNED TRUE - ERROR APPLIES
		if !checked {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
			return c.RenderJSON(result)
		}

		result["TEST : "] = map[string]interface{}{
			"userID : ":    userID,
			"companyID : ": companyID,
		}

		getSubIntegs, err := GetGroupSubIntegrationsByParentID(groupId, subIntegration.ParentIntegrationID)
		if err != nil {
			result["message"] = "Error getting sub integrations"
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		if len(getSubIntegs) != 0 {
			for _, items := range getSubIntegs {
				subIntegrations := &dynamodb.DeleteItemInput{
					Key: map[string]*dynamodb.AttributeValue{
						"PK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, items.GroupID)),
						},
						"SK": {
							S: aws.String(utils.AppendPrefix(constants.PREFIX_SUB_INTEGRATION, items.IntegrationID)),
						},
					},
					TableName: aws.String(app.TABLE_NAME),
				}

				_, err := app.SVC.DeleteItem(subIntegrations)
				if err != nil {
					result["message"] = "Error deleting sub integrations"
					result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
					return c.RenderJSON(result)
				}
			}
		}

		integrations := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, groupId)),
				},
				"SK": {
					S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, subIntegration.ParentIntegrationID)),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
		}

		_, err = app.SVC.DeleteItem(integrations)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}

		// // generate log
		// var logs []models.Logs
		// // message: IntegrationX has been disconnected to GroupX
		// logs = append(logs, models.Logs{
		// 	CompanyID: companyID,
		// 	UserID:    c.ViewArgs["userID"].(string),
		// 	LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
		// 	LogType:   constants.ENTITY_TYPE_GROUP,
		// 	LogInfo: &models.LogInformation{
		// 		Group: &models.LogModuleParams{
		// 			ID: groupId,
		// 		},
		// 		// User: &models.LogModuleParams {
		// 		// 	ID: c.ViewArgs["userID"].(string),
		// 		// },
		// 		Integration: &models.LogModuleParams{
		// 			ID: subIntegration.ParentIntegrationID,
		// 		},
		// 	},
		// })
		// _, err = CreateBatchLog(logs)
		// if err != nil {
		// 	result["logs"] = "error while creating logs"
		// }

		// create group activity & company log
		companyLog := models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
			LogType:   constants.ENTITY_TYPE_GROUP,
			LogInfo: &models.LogInformation{
				Group: &models.LogModuleParams{
					ID: groupId,
				},
				Integration: &models.LogModuleParams{
					ID: subIntegration.ParentIntegrationID,
				},
			},
		}

		groupLog := models.Logs{
			GroupID:   groupId,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_REMOVE_GROUP_INTEGRATION,
			LogType:   constants.ENTITY_TYPE_GROUP,
			LogInfo: &models.LogInformation{
				Group: &models.LogModuleParams{
					ID: groupId,
				},
				Integration: &models.LogModuleParams{
					ID: subIntegration.ParentIntegrationID,
				},
			},
		}

		companyLogID, err := ops.InsertLog(companyLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating company logs for removing group and integration connection")
		}
		_ = companyLogID

		groupLogID, err := ops.InsertLog(groupLog)
		if err != nil {
			errorMessages = append(errorMessages, "Error while creating group logs for removing group and integration connection")
		}
		_ = groupLogID

		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
		return c.RenderJSON(result)
	}

	if len(errorMessages) != 0 {
		result["errors"] = errorMessages
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

func CheckSubIntegrationsConnectedItemIsEmpty(groupID, parentIntegrationID string) (bool, error) {
	//get group integrations
	integrationsResult, err := GetGroupIntegrations(groupID)
	if err != nil {
		return false, err
	}
	//
	connectedItemAllEmpty := true
	//find parent integration
	for _, groupIntegrations := range integrationsResult {
		//if found
		if groupIntegrations.IntegrationID == parentIntegrationID {
			//check connected items if not empty
			//if not empty return false
			//if all is empty return true
			subIntegrationsResult, err := GetGroupSubIntegration(groupID)
			if err != nil {
				return false, err
			}
			//filter subintagration related to parent integration
			subInteg := []models.GroupSubIntegration{}
			for _, sub := range subIntegrationsResult {
				if sub.ParentIntegrationID == parentIntegrationID {
					subInteg = append(subInteg, sub)
				}
			}

			for _, checkItems := range subInteg {
				if len(checkItems.ConnectedItems) > 0 {
					connectedItemAllEmpty = false
					break
				}
			}

			break
		}
	}

	return connectedItemAllEmpty, nil
}

/*
****************
Check if there are duplicated groups due to getting the bookmark groups first
FilterBookmarkGroups()
****************
*/
func FilterDuplicatedGroups(groups []models.Group) []models.Group {

	group := models.Group{}

	var filteredGroups []models.Group
	for _, group = range groups {
		skip := false
		for _, u := range filteredGroups {
			if group.GroupID == u.GroupID {
				skip = true
				break
			}
		}
		if !skip {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

/*
****************
Check if there are bookmark groups on next group request
FilterBookmarkGroups()
****************
*/
func FilterBookmarkGroups(groups []models.Group, bookmarkGroups []string) []models.Group {

	group := models.Group{}

	var filteredGroups []models.Group
	for _, group = range groups {
		skip := false
		for _, groupID := range bookmarkGroups {
			if group.GroupID == groupID {
				skip = true
				break
			}
		}
		if !skip {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

/*
****************
RemoveGroupSubIntegrations()
****************
*/

// @Summary Remove Group Sub Integrations
// @Description This endpoint removes sub-integrations from a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/sub_integrations/:groupID [delete]
func (c GroupController) RemoveGroupSubIntegrations(groupID string) revel.Result {
	result := make(map[string]interface{})

	var subIntegrations []models.Integration

	// todo validate required params
	if groupID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_422)
		return c.RenderJSON(result)
	}
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
						S: aws.String(constants.PREFIX_SUB_INTEGRATION),
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

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &subIntegrations)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	var batches [][]*dynamodb.WriteRequest
	var currentBatch []*dynamodb.WriteRequest

	for i, sub := range subIntegrations {
		currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
			Key: map[string]*dynamodb.AttributeValue{
				"PK": &dynamodb.AttributeValue{
					S: aws.String(sub.PK),
				},
				"SK": &dynamodb.AttributeValue{
					S: aws.String(sub.SK),
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
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
ConnectIntegration()
-connect integration to group
****************
*/

// @Summary Connect Integration
// @Description This endpoint connects an integration to a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param body body models.ConnectIntegrationRequest true "Connect integration body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/connect_integration [put]
func (c GroupController) ConnectIntegration(groupID string) revel.Result {
	result := make(map[string]interface{})

	integrationID := c.Params.Form.Get("integration_id")

	item := &models.Integration{
		PK:            utils.AppendPrefix(constants.PREFIX_GROUP, groupID),
		SK:            utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationID),
		GroupID:       groupID,
		IntegrationID: integrationID,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(input)
	if err != nil {
		result["err"] = err
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
ConnectIntegrationToGroups
*****************
*/

// @Summary Connect Integration To Groups
// @Description This endpoint connects an integration to multiple specified groups, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.ConnectIntegrationToGroupsRequest true "Connect integration to groups body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/integrations/connect [post]
func (c GroupController) ConnectIntegrationToGroups() revel.Result {
	result := make(map[string]interface{})

	integrationID := c.Params.Form.Get("integration_id")
	groupCollections := c.Params.Form.Get("groups")
	companyID := c.ViewArgs["companyID"].(string)

	var groups []models.Group
	json.Unmarshal([]byte(groupCollections), &groups)

	if len(groups) == 0 {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	var putRequest []*dynamodb.WriteRequest

	for _, group := range groups {
		putRequest = append(putRequest, &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					"PK": &dynamodb.AttributeValue{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, group.GroupID)),
					},
					"SK": &dynamodb.AttributeValue{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationID)),
					},
					"IntegrationID": &dynamodb.AttributeValue{
						S: aws.String(integrationID),
					},
					"GroupID": &dynamodb.AttributeValue{
						S: aws.String(group.GroupID),
					},
					"CompanyID": &dynamodb.AttributeValue{
						S: aws.String(companyID),
					},
				},
			},
		})
	}

	itemsInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			app.TABLE_NAME: putRequest,
		},
	}

	_, err := app.SVC.BatchWriteItem(itemsInput)
	if err != nil {
		result["error_message"] = err.Error()
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["message"] = "Groups successfully added"
	return c.RenderJSON(result)
}

// GetGroupInformation
func GetGroupInformation(groupID string, includes ...string) (models.Group, error) {
	var group models.Group

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_GROUP + groupID),
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

	result, err := app.SVC.Query(params)
	if err != nil {
		return group, err
	}

	if len(result.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(result.Items[0], &group)
		if err != nil {
			return group, err
		}

		if len(includes) != 0 {
			if utils.StringInSlice("department", includes) {
				department, err := GetDepartmentInformation(group.DepartmentID)
				if err == nil {
					group.Department = department
				}
			}
		}
	}

	return group, nil
}

// @Summary Get Group Department
// @Description This endpoint retrieves the department associated with a specific group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/:groupID/department [get]
func (c GroupController) GetGroupDepartment(groupID string) revel.Result {
	group, err := ops.GetGroupDepartmentByID(groupID)
	if err != nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "Something went wrong with group",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	if group.DepartmentID == "" {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "Error on getting group department ID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}
	qd := ops.GetDepartmentByIDPayload{
		DepartmentID: group.DepartmentID,
		CheckStatus:  true,
	}
	department, opsError := ops.GetDepartmentByID(qd, c.Validation)
	if opsError != nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "Error on getting department ID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	return c.RenderJSON(department)
}

type GetGroupMembersCountOutput struct {
	IndividualMembers int `json:"individualMembers"`
	SubGroupMembers   int `json:"subGroupMembers"`
	TotalGroupMembers int `json:"totalGroupMembers"`
}

// GetGroupMembersInGroupCount

// @Summary Get Group Members In Group Count
// @Description This endpoint retrieves the count of members in a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/profile/count [get]
func (c GroupController) GetGroupMembersInGroupCount(groupId string) revel.Result {
	//
	//
	//
	// )

	subGroupMembers, err := GetGroupMembersInGroup(c.ViewArgs["companyID"].(string), groupId, constants.MEMBER_TYPE_GROUP, groupId)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}

	//
	//
	//
	//
	//

	individualMembers, err := GetGroupMembersInGroup(c.ViewArgs["companyID"].(string), groupId, constants.MEMBER_TYPE_USER, groupId)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}

	//
	//
	//
	//
	//

	output := GetGroupMembersCountOutput{
		IndividualMembers: len(individualMembers),
		SubGroupMembers:   len(subGroupMembers),
		TotalGroupMembers: len(individualMembers) + len(subGroupMembers),
	}

	c.Response.Status = 200
	return c.RenderJSON(output)
}

// GetGroupMembersInGroup() using handle query limit
func GetGroupMembersInGroup(companyID, groupID, memberType string, mainGroupID ...string) ([]models.GroupMember, error) {
	members := []models.GroupMember{}

	var sk string
	switch memberType {
	case constants.MEMBER_TYPE_USER:
		sk = constants.PREFIX_USER
	case constants.MEMBER_TYPE_GROUP:
		sk = constants.PREFIX_GROUP
	}

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
						S: aws.String(sk),
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

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return members, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return members, e
	}

	return members, nil
}

type GetAllGroupsOfCompanyCountOutput struct {
	AllGroupsCount int `json:"allGroupsCount"`
}

// GetAllGroupsOfCompanyCount

// @Summary Get Company Groups Count
// @Description This endpoint retrieves the count of all groups associated with a company, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/company/count [get]
func (c GroupController) GetAllGroupsOfCompanyCount() revel.Result {
	// companyID := c.ViewArgs["companyID"].(string)
	// searchKey := c.Params.Query.Get("key")
	// // status := c.Params.Query.Get("status")
	// paramLastEvaluatedKey := c.Params.Query.Get("last_evaluated_key")

	// groups, lastEvaluatedKey, err := GetAllGroupsOfCompany(c.ViewArgs["companyID"].(string), searchKey, constants.ITEM_STATUS_ACTIVE, paramLastEvaluatedKey)
	// if err != nil {
	// 	c.Response.Status = 400
	// 	return c.RenderJSON(err.Error())
	// }
	totalGroups, err := GetGroupsTotal(c.ViewArgs["companyID"].(string))
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}
	output := GetAllGroupsOfCompanyCountOutput{
		AllGroupsCount: totalGroups,
	}

	c.Response.Status = 200
	return c.RenderJSON(output)
}

func GetAllGroupsOfCompany(companyID string, searchKey string, status string, paramLastEvaluatedKey string) ([]models.Group, models.Group, error) {
	groups := []models.Group{}
	lastEvaluatedKey := models.Group{}

	pageLimit := utils.ConvertStringPageLimitToInt64("10")

	if searchKey != "" {
		searchKey = strings.ToLower(searchKey)
	}
	if status == "" {
		status = constants.ITEM_STATUS_ACTIVE
	}
	if status != "" {
		status = strings.ToUpper(status)
	}

	var queryFilter map[string]*dynamodb.Condition
	queryFilter = map[string]*dynamodb.Condition{
		"SearchKey": &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_CONTAINS),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(searchKey),
				},
			},
		},
		"Status": &dynamodb.Condition{
			ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
			AttributeValueList: []*dynamodb.AttributeValue{
				{
					S: aws.String(status),
				},
			},
		},
	}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
		},
		QueryFilter:       queryFilter, //kahit wala nasince count only?
		Limit:             aws.Int64(pageLimit),
		IndexName:         aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
		ScanIndexForward:  aws.Bool(false),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(models.Group{}, paramLastEvaluatedKey),
	}

	res, err := ops.HandleQuery(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return groups, lastEvaluatedKey, e
	}

	key := res.LastEvaluatedKey
	for len(key) != 0 {
		if int64(len(res.Items)) >= pageLimit {
			break
		}
		params.ExclusiveStartKey = key
		r, err := ops.HandleQuery(params)
		if err != nil {
			break
		}
		key = r.LastEvaluatedKey
		res.Items = append(res.Items, r.Items...)
	}

	_ = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &groups)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return groups, lastEvaluatedKey, e
	}

	return groups, lastEvaluatedKey, nil
}

// @Summary Get Group Count By Company
// @Description This endpoint retrieves the count of groups associated with a specific company, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param companyID path string true "Company ID"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/companies/:companyID/count [get]
func (c GroupController) GetGroupCountByCompany(companyID string) revel.Result {
	groupList, err := GetGroupList(companyID)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err.Error())
	}
	output := len(groupList)
	return c.RenderJSON(output)
}

func GetGroupList(companyID string) ([]models.Group, error) {
	var groups []models.Group
	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
						S: aws.String(constants.PREFIX_GROUP),
					},
				},
			},
		},
		QueryFilter: map[string]*dynamodb.Condition{
			"Type": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String("GROUP"),
					},
				},
			},
			"Status": {
				ComparisonOperator: aws.String(constants.CONDITION_NOT_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.ITEM_STATUS_INACTIVE),
					},
				},
			},
		},
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS),
	}
	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return groups, err
	}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &groups)
	if err != nil {
		return groups, err
	}
	return groups, nil
}

type CopyGroupIntegrationsPayload struct {
	SourceGroupID         string   `json:"sourceGroupId"`
	IntegrationListToCopy []string `json:"integrationListToCopy"`
	IsCopyConnectedItems  bool     `json:"isCopyConnectedItems"`
}

// @Summary Copy Group Integrations
// @Description This endpoint copies integrations from one group to another, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param groupID path string true "Group ID"
// @Param body body models.CopyGroupIntegrationsRequest true "Copy group integrations body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/copy/:groupID [post]
func (c GroupController) CopyGroupIntegrations(groupID string) revel.Result {

	companyID := c.ViewArgs["companyID"].(string)

	var data CopyGroupIntegrationsPayload

	c.Params.BindJSON(&data)

	isClonedIntegrationsSuccess, err := CloneIntegrations(data.SourceGroupID, groupID, companyID, data.IntegrationListToCopy, data.IsCopyConnectedItems)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "CopyGroupIntegrations - Error on CloneIntegrations",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		})
	}
	if !isClonedIntegrationsSuccess {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "CopyGroupIntegrations - Error while cloneing integrations",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
		})
	}

	for _, integration := range data.IntegrationListToCopy {
		logs := []models.Logs{}

		logs = append(logs, models.Logs{
			CompanyID: companyID,
			UserID:    c.ViewArgs["userID"].(string),
			LogAction: constants.LOG_ACTION_ADD_GROUP_INTEGRATION,
			LogType:   constants.ENTITY_TYPE_GROUP,
			LogInfo: &models.LogInformation{
				Group: &models.LogModuleParams{
					ID: groupID,
				},
				Origin: &models.LogModuleParams{
					ID: data.SourceGroupID,
				},
				Integration: &models.LogModuleParams{
					ID: integration,
				},
			},
		})

		_, err = CreateBatchLog(logs)
		if err != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        "CopyGroupIntegrations - Error creating logs",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_400),
			})
		}
	}

	c.Response.Status = 200
	return nil
}

type GetCompanyMemberParams struct {
	UserID    string
	CompanyID string
}

type DeleteIntegrationsRequestInput struct {
	UserID        string `json:"user_id"`
	GroupID       string `json:"group_id"`
	IntegrationID string `json:"integration_id,omitempty"`
}

// @Summary Delete Integrations Request
// @Description This endpoint processes a request to delete integrations from a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.DeleteIntegrationsRequestRequest true "Delete integrations request body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/integrations/delete-request [post]
func (c GroupController) DeleteIntegrationsRequest() revel.Result {
	result := make(map[string]interface{})

	companyID := c.ViewArgs["companyID"].(string)

	var input DeleteIntegrationsRequestInput
	c.Params.BindJSON(&input)

	user, err := ops.GetCompanyMember(
		ops.GetCompanyMemberParams{
			UserID:    input.UserID,
			CompanyID: companyID,
		},
		c.Controller,
	)
	if err != nil {
		c.Response.Status = 400
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		result["message"] = "Unable to retrieve User"
		return c.RenderJSON(result)
	}

	group, err := GetCompanyGroupNew(companyID, input.GroupID)
	if err != nil {
		c.Response.Status = 400
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		result["message"] = "Unable to retrieve Group"
		return c.RenderJSON(result)
	}

	integration, error := GetIntegrationByID(input.IntegrationID)
	if error != nil {
		c.Response.Status = 400
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		result["message"] = "Unable to retrieve Integration"
		return c.RenderJSON(result)
	}

	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	if err != nil {
		c.Response.Status = 400
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		result["message"] = "Unable to retrieve Company Admins"
		return c.RenderJSON(result)
	}

	notificationContent := models.NotificationContentType{
		RequesterUserID: input.UserID,
		ActiveCompany:   companyID,
		GroupID:         input.GroupID,
		Integration: models.NotificationIntegration{
			IntegrationID:   input.IntegrationID,
			IntegrationSlug: integration.IntegrationSlug,
		},
		Message: user.FirstName + " " + user.LastName + " has requested to remove " + integration.IntegrationName + " from " + group.GroupName + ".",
	}

	var success []models.Notification
	var failed []models.Notification
	var duplicate []models.Notification
	for _, admin := range companyAdmins {
		notifications, _ := ops.GetUserNotifications(admin.UserID, companyID)
		duplicateFound := false
		for _, notif := range notifications {
			if notif.NotificationType == constants.REQUEST_REMOVE_INTEGRATION && notif.UserID == admin.UserID && notif.NotificationContent.Integration.IntegrationID == input.IntegrationID && notif.NotificationContent.GroupID == input.GroupID && notif.NotificationContent.ActiveCompany == companyID {
				duplicateFound = true
				duplicate = append(duplicate, notif)
				break
			}
		}

		if duplicateFound {
			continue
		} else {
			resp, err := ops.CreateNotification(ops.CreateNotificationInput{
				UserID:              admin.UserID,
				NotificationType:    constants.REQUEST_REMOVE_INTEGRATION,
				NotificationContent: notificationContent,
				Global:              false,
			}, c.Controller)

			if err != nil {
				failed = append(failed, resp)
			} else {
				success = append(success, resp)
			}
		}
	}

	result["success"] = success
	result["failed"] = failed
	result["duplicate"] = duplicate
	return c.RenderJSON(result)
}

type RequestToJoinGroupParams struct {
	GroupID string `json:"group_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

func requestToJoinGroupSendNotification(c GroupController, adminID string, requesterUserInfo models.CompanyUser, input RequestToJoinGroupParams) revel.Result {
	companyID := c.ViewArgs["companyID"].(string)

	//get groupinfo
	group, err := GetGroupByID(input.GroupID)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to get group info",
		})
	}

	notificationContent := models.NotificationContentType{
		RequesterUserID: requesterUserInfo.UserID,
		ActiveCompany:   companyID,
		Message:         requesterUserInfo.FirstName + " " + requesterUserInfo.LastName + " has requested to join in " + group.GroupName + ".",
		GroupID:         input.GroupID,
		IsAccepted:      "UNDER_REVIEW",
	}

	createdNotification, nErr := ops.CreateNotification(ops.CreateNotificationInput{
		UserID:              adminID,
		NotificationType:    constants.REQUEST_TO_JOIN_GROUP,
		NotificationContent: notificationContent,
		Global:              false,
	}, c.Controller)

	if nErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to create notification for " + requesterUserInfo.UserID,
		})
	}

	return c.RenderJSON(createdNotification)
}

// @Summary Request To Join Group
// @Description This endpoint allows a user to request to join a specified group, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.RequestToJoinGroupRequest true "Request to join group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/members/request [post]
func (c GroupController) RequestToJoinGroup() revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	var input RequestToJoinGroupParams
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.UserID, input.GroupID}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "RequestToJoinGroup Error: Missing required parameter - userId/groupId",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	//get company admins
	var getAdminsIDs []string
	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company Admins",
		})
	}
	for _, cAdmin := range companyAdmins {
		getAdminsIDs = append(getAdminsIDs, cAdmin.UserID)
	}

	//get department admins
	deptAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_DEPARTMENT_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Department Admins",
		})
	}
	for _, dAdmin := range deptAdmins {
		getAdminsIDs = append(getAdminsIDs, dAdmin.UserID)
	}

	//get group admins
	groupAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_GROUP_ADMIN)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Group Admins",
		})
	}
	for _, gAdmin := range groupAdmins {
		getAdminsIDs = append(getAdminsIDs, gAdmin.UserID)
	}

	//remove duplicate ids (there are instance that a user has multiple roles...)
	adminIDs := utils.RemoveDuplicateStrings(getAdminsIDs)

	//get user info
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

	_ = requesterInfo

	if len(adminIDs) != 0 {
		for _, adminID := range adminIDs {
			userNotifications, _ := ops.GetUserNotifications(adminID, companyID)

			duplicateFound := false
			for _, notif := range userNotifications {
				if notif.NotificationType == constants.REQUEST_TO_JOIN_GROUP && notif.UserID == adminID && notif.NotificationContent.RequesterUserID == input.UserID && notif.NotificationContent.ActiveCompany == companyID {
					if input.GroupID != "" && notif.NotificationContent.GroupID == input.GroupID && notif.NotificationContent.IsAccepted == "UNDER_REVIEW" {
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
			} else {
				requestToJoinGroupSendNotification(c, adminID, requesterInfo, input)
			}
		}
	}

	return nil
}

func GetGroupsTotal(companyID string) (totalGroups int, err error) {

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
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
		IndexName: aws.String(constants.INDEX_NAME_GET_COMPANY_ITEMS_NEW),
	}

	result, err := ops.HandleQueryLimit(params)
	if err != nil {
		return totalGroups, err
	}

	return len(result.Items), nil
}

// type RemoveMemberToSubIntegration interface {
// 	Remove(groupKey string) error
// }

// type AddMemberToSubIntegration interface {
// 	Add(connectedItems []string) error
// }

// type GoogleConfigToken struct {
// 	Token  *oauth2.Token
// 	Config *oauth2.Config
// }

// type GoogleAdminRemoveUsers struct {
// 	GoogleConfigToken
// 	Users []models.MemberCronJobData
// }

// func (subInteg *GoogleAdminRemoveUsers) Remove(groupKey string) error {
// 	ctx := context.Background()
// 	token := subInteg.Token
// 	config := subInteg.Config

// 	client := config.Client(ctx, token)
// 	srv, err := admin.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return fmt.Errorf("unable to retrieve directory Client")
// 	}

// 	for _, member := range subInteg.Users {
// 		//* Get user data
// 		user, err := ops.GetUserData(member.MemberID, "")
// 		if err != nil {
// 			return err
// 		}

// 		err = srv.Members.Delete(groupKey, user.Email).Do()
// 		if err != nil {
// 			gErr := err.(*googleapi.Error)
// 			if gErr.Code != 404 {
// 				return fmt.Errorf("error code: %d , error message: %q", gErr.Code, gErr.Message)
// 			}
// 		}
// 	}

// 	return nil
// }

// type GoogleAdminRemoveGroups struct {
// 	GoogleConfigToken
// 	Groups []models.MemberCronJobData
// }

// func (subInteg *GoogleAdminRemoveGroups) Remove(groupKey string) error {
// 	ctx := context.Background()
// 	token := subInteg.Token
// 	config := subInteg.Config

// 	client := config.Client(ctx, token)
// 	srv, err := admin.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return errors.New("unable to retrieve directory client")
// 	}

// 	for _, subGroup := range subInteg.Groups {
// 		//* Get group data
// 		group, err := GetGroupByID(subGroup.MemberID)
// 		if err != nil {
// 			return err
// 		}
// 		//* Next loop if no associated google account
// 		if group.AssociatedAccounts["google"] == nil {
// 			continue
// 		}

// 		email := group.AssociatedAccounts["google"][0]

// 		err = srv.Members.Delete(groupKey, email).Do()
// 		if err != nil {
// 			gErr := err.(*googleapi.Error)
// 			if gErr.Code != 404 {
// 				return fmt.Errorf("error code: %d , error message: %q", gErr.Code, gErr.Message)
// 			}
// 		}
// 	}

// 	return nil
// }

// func RemoveMembersToGoogleAdmin(googleAdmin RemoveMemberToSubIntegration, groupKey string) error {
// 	err := googleAdmin.Remove(groupKey)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// type AddGoogleAdminUsers struct {
// 	GoogleConfigToken
// 	Member    models.MemberCronJobData
// 	CompanyID string
// }

// func (gointeg *AddGoogleAdminUsers) Add(connectedItems []string) error {
// 	token := gointeg.Token
// 	config := gointeg.Config
// 	ctx := context.Background()
// 	var c *revel.Controller
// 	client := config.Client(ctx, token)
// 	srv, err := admin.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return fmt.Errorf("unable to retrieve directory client")
// 	}

// 	//* Get user data
// 	user, errUser := ops.GetUserByID(gointeg.Member.MemberID)
// 	if err != nil {
// 		return fmt.Errorf("error code: %d, error message: %s", errUser.Code, errUser.Message)
// 	}

// 	var member admin.Member
// 	member.Email = user.Email
// 	member.Role = "MEMBER"

// 	companyUser, opsError := ops.GetCompanyMember(ops.GetCompanyMemberParams{
// 		UserID: user.UserID,
// 		// CompanyID: u.ActiveCompany,
// 		CompanyID: gointeg.CompanyID,
// 	}, c)

// 	if opsError != nil {
// 		return fmt.Errorf("error code: %d, error message: %s", opsError.HTTPStatusCode, opsError.Message)
// 	}

// 	if len(companyUser.AssociatedAccounts[constants.INTEG_SLUG_GOOGLE_CLOUD]) != 0 {
// 		associatedGoogleAccount := companyUser.AssociatedAccounts[constants.INTEG_SLUG_GOOGLE_CLOUD][0]
// 		member.Email = associatedGoogleAccount
// 		// what if no associated account? like external user
// 		// continue to add them instead?
// 	}

// 	for _, groupKey := range connectedItems {
// 		_, err = srv.Members.Insert(groupKey, &member).Do()
// 		if err != nil {
// 			skipError := false
// 			gErr := err.(*googleapi.Error)
// 			if len(gErr.Errors) != 0 {
// 				if gErr.Errors[0].Reason == "duplicate" {
// 					skipError = true
// 				}
// 			}
// 			if !skipError {
// 				return fmt.Errorf("error code: %d, error message: %s", gErr.Code, gErr.Message)
// 			}
// 		}
// 	}

// 	return nil
// }

// type AddGoogleAdminGroups struct {
// 	GoogleConfigToken
// 	Member models.MemberCronJobData
// }

// func (gointeg *AddGoogleAdminGroups) Add(connectedItems []string) error {
// 	token := gointeg.Token
// 	config := gointeg.Config
// 	ctx := context.Background()
// 	client := config.Client(ctx, token)
// 	srv, err := admin.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return fmt.Errorf("unable to retrieve directory client")
// 	}

// 	var member admin.Member
// 	member.Role = "MEMBER"

// 	memberGroup, err := groupfunctions.GetGroupByID(gointeg.Member.MemberID)
// 	if err != nil {
// 		return err
// 	}

// 	associatedAccounts := memberGroup.AssociatedAccounts
// 	associatedAccountsGoogle := associatedAccounts["google"]

// 	if len(associatedAccounts) != 0 {
// 		member.Email = associatedAccountsGoogle[0]
// 	} else {
// 		member.Email = memberGroup.GroupEmail
// 	}

// 	if member.Email != "" {
// 		for _, groupKey := range connectedItems {
// 			_, err := srv.Members.Insert(groupKey, &member).Do()
// 			if err != nil {
// 				skipError := false
// 				gErr := err.(*googleapi.Error)
// 				if len(gErr.Errors) != 0 {
// 					if gErr.Errors[0].Reason == "duplicate" {
// 						skipError = true
// 					}
// 				}
// 				if !skipError {
// 					return fmt.Errorf("error code: %d, error message: %q", gErr.Code, gErr.Message)
// 				}
// 			}
// 		}
// 	}

// 	return nil
// }

// func AddMembersToGoogleAdmin(googleAdmin AddMemberToSubIntegration, connectedItems []string) error {
// 	err := googleAdmin.Add(connectedItems)
// 	if err != nil {
// 		return err
// 	}

//		return nil
//	}
type ImportExternalGoogleGroupInput struct {
	GroupName        string
	GroupEmail       string
	GroupDescription string
	CompanyID        string
	DepartmentID     string
}

// @Summary Import External Google Group
// @Description This endpoint imports an external Google group into the system, returning a 200 OK upon success.
// @Tags groups
// @Produce json
// @Param body body models.ImportExternalGoogleGroupRequest true "Import external google group body"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /groups/import/external/google [post]
func (c GroupController) ImportExternalGoogleGroup() revel.Result {
	var input ImportExternalGoogleGroupInput
	c.Params.BindJSON(&input)

	groupUUID := utils.GenerateTimestampWithUID()

	// TODO validation

	group := models.Group{
		PK:               utils.AppendPrefix(constants.PREFIX_GROUP, groupUUID),
		SK:               utils.AppendPrefix(constants.PREFIX_DEPARTMENT, input.DepartmentID),
		GSI_SK:           utils.AppendPrefix(constants.PREFIX_GROUP, strings.ToLower(input.GroupName)),
		GroupID:          groupUUID,
		DepartmentID:     input.DepartmentID,
		CompanyID:        input.CompanyID,
		GroupName:        input.GroupName,
		GroupDescription: input.GroupDescription,
		GroupColor:       utils.GetRandomColor(),
		Status:           constants.ITEM_STATUS_ACTIVE,
		Type:             constants.ENTITY_TYPE_GROUP,
		NewGroup:         constants.BOOL_TRUE,
		CreatedAt:        utils.GetCurrentTimestamp(),
		SearchKey:        strings.ToLower(input.GroupName),
	}

	group.AssociatedAccounts = map[string][]string{
		"google": []string{input.GroupEmail},
	}

	data, err := dynamodbattribute.MarshalMap(group)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        err.Error(),
		})
	}

	groupInput := &dynamodb.PutItemInput{
		Item:      data,
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.PutItem(groupInput)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        err.Error(),
		})
	}

	var logs []models.Logs
	logs = append(logs, models.Logs{
		CompanyID: input.CompanyID,
		UserID:    c.ViewArgs["userID"].(string),
		LogAction: constants.LOG_ACTION_ADD_GROUP,
		LogType:   constants.ENTITY_TYPE_GROUP,
		LogInfo: &models.LogInformation{
			Group: &models.LogModuleParams{
				ID: group.GroupID,
			},
			User: &models.LogModuleParams{
				ID: c.ViewArgs["userID"].(string),
			},
			Department: &models.LogModuleParams{
				ID: input.DepartmentID,
			},
		},
	})
	log, err := CreateBatchLog(logs)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        err.Error(),
		})
	}
	_ = log

	return c.RenderJSON(group)
}
