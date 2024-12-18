package controllers

import (
	"encoding/json"
	"grooper/app/constants"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"runtime"
	"sort"
	"sync"

	bitbucketModel "grooper/app/models/bitbucket"

	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
)

type BitbucketGroupsController struct {
	*revel.Controller
}

type SendBitbucketInvitationsInput struct {
	Emails  []string `json:"unregisteredUsers"`
	MyGroup string   `json:"myGroup"`
}

func (c BitbucketGroupsController) SendBitbucketInvitations() revel.Result {
	bitbucketCreds := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input SendBitbucketInvitationsInput
	c.Params.BindJSON(&input)

	for _, email := range input.Emails {
		postBody, _ := json.Marshal(map[string]interface{}{
			"email":      email,
			"group_slug": input.MyGroup,
		})
		err := bitbucketOperations.InviteUsersWithResponse(bitbucketCreds.Workspace, postBody, bitbucketCreds.Token)
		if err != nil {
			return c.RenderJSON(err)
		}

	}

	return c.RenderJSON("WALA")
}

func (c BitbucketGroupsController) ListBitbucketGroups() revel.Result {
	bitbucketCreds := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	bitbucketGroups, err := bitbucketOperations.ListGroups(
		bitbucketCreds.Workspace,
		bitbucketCreds.Token)
	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(err)
	}

	formatGroups := []bitbucketModel.GroupsFormatFE{}

	for _, group := range *bitbucketGroups {
		formatGroups = append(formatGroups, bitbucketModel.GroupsFormatFE{
			ID:         group.Slug,
			Label:      group.Name,
			Value:      group.Slug,
			Slug:       group.Slug,
			Permission: group.Permission,
		})
	}

	c.Response.Status = 200
	return c.RenderJSON(formatGroups)
}
func (c BitbucketGroupsController) ListConnectedBitbucketGroups(groupID string) revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token
	companyId := c.ViewArgs["companyID"].(string)
	integrationID := c.Params.Query.Get("integration_id")
	userIntegrations := c.Params.Query.Get("uids")
	saasMembers := []string{}
	if userIntegrations != "" {
		errMarshal := json.Unmarshal([]byte(userIntegrations), &saasMembers)
		if errMarshal != nil {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "BitbucketGroupMembers Error: Unmarshal Members",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}
	}
	connectedItems, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketGroups Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	pendingInvitations, error := bitbucketOperations.GetPendingInvitations(bitbucketToken, bitbucketWorkspace)
	if error != nil {
		return c.RenderJSON(error)
	}

	emails := []string{}
	for _, invitation := range pendingInvitations {
		emails = append(emails, invitation.Email)
	}

	if len(connectedItems) == 0 {
		c.Response.Status = 200
		return c.RenderJSON(connectedItems)
	}

	formatGroups, Perr := BitbucketGroupsWithWorkerPool(BitbucketGroupsWithWorkerPoolStruct{
		Token:              bitbucketToken,
		Workspace:          bitbucketWorkspace,
		IntegrationID:      integrationID,
		ConnectedItem:      connectedItems[0],
		CompanyID:          companyId,
		SaaSMembers:        saasMembers,
		PendingInvitations: emails,
		C:                  c.Controller,
	})
	if Perr != nil {
		c.Response.Status = 400
		c.RenderJSON(Perr)
	}
	// bitbucketGroups, bErr := bitbucketOperations.ListGroups(bitbucketWorkspace, bitbucketToken)
	// if bErr != nil {
	// 	c.Response.Status = 401
	// 	return c.RenderJSON(err)
	// }
	// formatGroups := []bitbucketModel.GroupsFormatFE{}

	// for _, group := range *bitbucketGroups {
	// 	for _, connectedItem := range connectedItems {
	// 		if group.Slug == connectedItem {
	// 			formatGroupMembers, groupMembers, err := BitbucketGroupMembersHelper(BitbucketGroupMembersHelperParams{
	// 				Workspace:     bitbucketWorkspace,
	// 				GroupSlug:     group.Slug,
	// 				Token:         bitbucketToken,
	// 				SaasMembers:   saasMembers,
	// 				IntegrationId: integrationID,
	// 				CompanyId:     companyId,
	// 				C:             c.Controller,
	// 			})
	// 			if err != nil {
	// 				return c.RenderJSON(err)
	// 			}

	// 			_ = groupMembers

	// 			formatGroups = append(formatGroups, bitbucketModel.GroupsFormatFE{
	// 				ID:         group.Slug,
	// 				Label:      group.Name,
	// 				Value:      group.Slug,
	// 				Slug:       group.Slug,
	// 				Permission: group.Permission,
	// 				Members:    *formatGroupMembers,
	// 			})
	// 		}
	// 	}
	// }

	c.Response.Status = 200
	return c.RenderJSON(formatGroups)
}

type BitbucketGroupsWithWorkerPoolStruct struct {
	Token              string
	Workspace          string
	IntegrationID      string
	ConnectedItem      string
	CompanyID          string
	SaaSMembers        []string
	PendingInvitations []string
	C                  *revel.Controller
}

type ByBitbucketGroupSlug []bitbucketModel.GroupsResponse

func (a ByBitbucketGroupSlug) Len() int           { return len(a) }
func (a ByBitbucketGroupSlug) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByBitbucketGroupSlug) Less(i, j int) bool { return a[i].Slug < a[j].Slug }
func BitbucketGroupsWithWorkerPool(params BitbucketGroupsWithWorkerPoolStruct) ([]bitbucketModel.GroupsFormatFE, *models.ErrorResponse) {
	bitbucketGroups, bErr := bitbucketOperations.ListGroups(params.Workspace, params.Token)
	if bErr != nil {
		return nil, bErr
	}
	sort.Sort(ByBitbucketGroupSlug(*bitbucketGroups))

	bitbucketGroupResponse := bitbucketModel.GroupsResponse{}

	index := sort.Search(len(*bitbucketGroups), func(i int) bool {
		return (*bitbucketGroups)[i].Slug >= params.ConnectedItem
	})
	if index < len((*bitbucketGroups)) && (*bitbucketGroups)[index].Slug == params.ConnectedItem {
		bitbucketGroupResponse = (*bitbucketGroups)[index]
	}

	formatGroupMembers, groupMembers, err := BitbucketGroupMembersHelper(BitbucketGroupMembersHelperParams{
		Workspace:          params.Workspace,
		GroupSlug:          bitbucketGroupResponse.Slug,
		Token:              params.Token,
		SaasMembers:        params.SaaSMembers,
		IntegrationId:      params.IntegrationID,
		CompanyId:          params.CompanyID,
		PendingInvitations: params.PendingInvitations,
		C:                  params.C,
	})
	if err != nil {
		return []bitbucketModel.GroupsFormatFE{}, err
	}

	_ = groupMembers
	var formatGroups []bitbucketModel.GroupsFormatFE
	formatGroups = append(formatGroups, bitbucketModel.GroupsFormatFE{
		ID:          bitbucketGroupResponse.Slug,
		Label:       bitbucketGroupResponse.Name,
		Value:       bitbucketGroupResponse.Slug,
		Slug:        bitbucketGroupResponse.Slug,
		Permission:  bitbucketGroupResponse.Permission,
		SaaSMembers: formatGroupMembers,
		Members:     []bitbucketModel.MemberResponse{},
	})

	return formatGroups, nil
}
func (c BitbucketGroupsController) CreateBitbucketGroup(groupID string) revel.Result {
	var data bitbucketModel.CreateBitbucketGroupPayload
	c.Params.BindJSON(&data)

	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	if utils.FindEmptyStringElement([]string{data.Name}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "CreateBitbucketGroup Error: Missing required parameter - GroupName",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	groupData := bitbucketModel.BitbucketGroupStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		PostBody:  []byte("name=" + data.Name),
	}

	createdGroup, bErr := bitbucketOperations.CreateGroup(groupData)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	postBody, _ := json.Marshal(map[string]interface{}{})

	memberData := bitbucketModel.BitbucketGroupMemberStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		Slug:      createdGroup.Slug,
		PostBody:  postBody,
	}
	var postBodyGroup []byte
	if data.Permission != "" {
		postBodyGroup, _ = json.Marshal(map[string]interface{}{
			"name": data.Name,
			// "permission": data.Permission,
		})
	} else {
		postBodyGroup, _ = json.Marshal(map[string]interface{}{
			"name": data.Name,
			// "permission": nil,
		})
	}
	groupData.PostBody = postBodyGroup
	groupData.Slug = createdGroup.Slug
	updatePermissionGroup, bErr := bitbucketOperations.UpdateGroup(groupData)
	if bErr != nil {
		//TODO logs
	}
	_ = updatePermissionGroup

	if data.ConnectedGroup {
		if len(data.Members) > 0 {
			for _, uuid := range data.Members {
				_, err := bitbucketOperations.AddMember(memberData, uuid)
				if err != nil {
					//TODO: Log Error?
				}
			}
		}
	}

	jobs.Now(ManageApplications{
		GroupSlug: createdGroup.Slug,
		GroupID:   groupID,
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		Type:      "ADD",
	})

	c.Response.Status = 201
	return c.RenderJSON(bitbucketModel.GroupsFormatFE{
		ID:    createdGroup.Slug,
		Value: createdGroup.Slug,
		Label: createdGroup.Name,
		Slug:  createdGroup.Slug,
	})
}
func (c BitbucketGroupsController) UpdateBitbucketGroup(groupSlug string) revel.Result {
	var data bitbucketModel.UpdateBitbucketGroupPayload
	c.Params.BindJSON(&data)

	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	if utils.FindEmptyStringElement([]string{data.Name, groupSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketGroup Error: Missing required parameter - GroupSlug/GroupName",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	var postBody []byte
	if data.Permission != "" {
		postBody, _ = json.Marshal(map[string]interface{}{
			"name": data.Name,
			// "permission": data.Permission,
		})
	} else {
		postBody, _ = json.Marshal(map[string]interface{}{
			"name": data.Name,
			// "permission": nil,
		})
	}
	groupData := bitbucketModel.BitbucketGroupStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		PostBody:  postBody,
		Slug:      groupSlug,
	}

	updatedGroup, err := bitbucketOperations.UpdateGroup(groupData)

	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(err)
	}

	c.Response.Status = 201
	return c.RenderJSON(bitbucketModel.GroupsFormatFE{
		ID:         updatedGroup.Slug,
		Value:      updatedGroup.Slug,
		Label:      updatedGroup.Name,
		Slug:       updatedGroup.Slug,
		Permission: updatedGroup.Permission,
	})
}
func (c BitbucketGroupsController) BitbucketGroupMembers(groupSlug string) revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token
	userIntegrations := c.Params.Query.Get("uids")

	members := []string{}
	errMarshal := json.Unmarshal([]byte(userIntegrations), &members)
	integrationID := c.Params.Query.Get("integration_id")
	companyId := c.ViewArgs["companyID"].(string)
	if errMarshal != nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "BitbucketGroupMembers Error: Missing required parameter - GroupSlug",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	if utils.FindEmptyStringElement([]string{groupSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "BitbucketGroupMembers Error: Missing required parameter - GroupSlug",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	formatGroupMembers, groupMembers, err := BitbucketGroupMembersHelper(BitbucketGroupMembersHelperParams{
		Workspace:     bitbucketWorkspace,
		GroupSlug:     groupSlug,
		Token:         bitbucketToken,
		SaasMembers:   members,
		IntegrationId: integrationID,
		CompanyId:     companyId,
		C:             c.Controller,
	})
	if err != nil {
		c.Response.Status = err.HTTPStatusCode
		return c.RenderJSON(err)
	}

	_ = groupMembers

	c.Response.Status = 200
	return c.RenderJSON(formatGroupMembers)
}
func (c BitbucketGroupsController) AddBitbucketGroupMembers(groupID string) revel.Result {
	companyId := c.ViewArgs["companyID"].(string)
	var data bitbucketModel.AddBitbucketGroupMemberPayload
	c.Params.BindJSON(&data)

	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	postBody, _ := json.Marshal(map[string]interface{}{})
	memberData := bitbucketModel.BitbucketGroupMemberStruct{
		Workspace: bitbucketWorkspace,
		Slug:      data.Slug,
		Token:     bitbucketToken,
		PostBody:  postBody,
	}
	var groupMembers []bitbucketModel.MemberResponse
	// var failedRequest []bitbucketModel.Error
	// connectedRepository, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_REPOSITORIES)
	// if err != nil {
	// 	c.Response.Status = 400
	// 	return c.RenderJSON(err)
	// }

	// connectedBitbucketProjects, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_PROJECTS)
	// if err != nil {
	// 	c.Response.Status = 400
	// 	return c.RenderJSON(err)
	// }

	invitedMembers := []bitbucketModel.InviteBitbucketMember{}
	// savedUUID := []string{}

	if len(data.InviteMembers) != 0 {
		// if len(connectedGroup) != 0 {
		for _, item := range data.InviteMembers {
			// postBody, _ := json.Marshal(map[string]interface{}{
			// 	"accountname": item.Name,
			// 	"permission":  "read",
			// 	"email":       item.Email,
			// })
			postBody, _ := json.Marshal(map[string]interface{}{
				"group_slug": data.Slug,
				"email":      item.Email,
			})
			// var userData bitbucketModel.InviteUserPayload

			// userData = bitbucketModel.InviteUserToGroupPayload{
			// 	Workspace: bitbucketWorkspace,
			// 	RepoSlug:  connectedGroup[0],
			// 	Token:     bitbucketToken,
			// 	PostBody:  postBody,
			// 	AccountName: item.Name,
			// }
			err := bitbucketOperations.InviteUsersToGroup(bitbucketModel.InviteUserToGroupPayload{
				Workspace:   bitbucketWorkspace,
				RepoSlug:    data.Slug,
				Token:       bitbucketToken,
				PostBody:    postBody,
				AccountName: item.Name,
			})
			if err != nil {
				// c.Response.Status = 401
				// return c.RenderJSON(err)
				continue
			}
			invitedMembers = append(invitedMembers, item)

			saveAccountErr := ops.SaveUserIntegrationAccounts(ops.SaveUserIntegrationAccountsInput{
				CompanyID:   companyId,
				UserID:      item.ID,
				Integration: constants.INTEG_SLUG_BITBUCKET,
				Account:     item.Email,
			}, c.Controller)

			if saveAccountErr != nil {
			}

			// }
		}

	}
	for _, item := range data.UUIDs {
		if item == "" {
			continue
		}

		resp, bErr := bitbucketOperations.AddMember(memberData, item)
		if bErr != nil {
			//TODO: log Errors
			//
		}
		if resp == nil {
			break
		}
		groupMembers = append(groupMembers, *resp)
		// if len(connectedBitbucketProjects) != 0 {
		// 	for _, projectItem := range connectedBitbucketProjects {

		// 		postBodyRGP, marshalError := json.Marshal(map[string]interface{}{
		// 			"permission": "read", // read is the default permission
		// 		})
		// 		if marshalError != nil {
		// 		}
		// 		createProject, err := bitbucketOperations.UpdateProjectUserPermission(bitbucketToken, bitbucketWorkspace, projectItem, item, postBodyRGP)
		// 		if err != nil {
		// 			failedRequest = append(failedRequest, bitbucketModel.Error{
		// 				Message: "Failed adding " + item + " on bitbucket project",
		// 			})
		// 		}
		// 		_ = createProject
		// 	}
		// }
		// if len(connectedRepository) != 0 {
		// 	for _, repoItem := range connectedRepository {

		// 		postBodyRGP, marshalError := json.Marshal(map[string]interface{}{
		// 			"permission": "read", // read is the default permission
		// 		})
		// 		if marshalError != nil {
		// 		}
		// 		createRepo, err := bitbucketOperations.UpdateRepositoryUserPermission(bitbucketToken, bitbucketWorkspace, repoItem, item, postBodyRGP)
		// 		if err != nil {
		// 			failedRequest = append(failedRequest, bitbucketModel.Error{
		// 				Message: "Failed adding " + item + " on bitbucket repository",
		// 			})
		// 		}
		// 		_ = createRepo
		// 	}

		// }
	}
	// if groupID != "" {
	// 	jobs.Now(ManageApplications{
	// 		GroupSlug: data.Slug,
	// 		GroupID:   groupID,
	// 		Workspace: bitbucketWorkspace,
	// 		Token:     bitbucketToken,
	// 		Type:      "ADD",
	// 	})
	// }

	c.Response.Status = 201
	return c.RenderJSON(groupMembers)
}
func (c BitbucketGroupsController) SyncBitbucketGroupMembers() revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	var data []bitbucketModel.AddBitbucketGroupMemberPayload
	c.Params.BindJSON(&data)

	postBody, _ := json.Marshal(map[string]interface{}{})

	memberData := bitbucketModel.BitbucketGroupMemberStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		PostBody:  postBody,
	}

	pendingUsers := []string{}
	pendingInvitations, err := bitbucketOperations.GetPendingInvitations(bitbucketToken, bitbucketWorkspace)
	if err != nil {
		return c.RenderJSON(err)
	}
	for _, pendingInvitation := range pendingInvitations {
		pendingUsers = append(pendingUsers, pendingInvitation.Email)
	}

	for _, group := range data {
		memberData.Slug = group.Slug
		for _, uuid := range group.UUIDs {
			foundInPendingUsers := false
			for _, email := range pendingUsers {
				if email == uuid {
					inviteBody, _ := json.Marshal(map[string]interface{}{
						"email":      email,
						"group_slug": group.Slug,
					})
					err := bitbucketOperations.InviteUsersWithResponse(bitbucketWorkspace, inviteBody, bitbucketToken)
					if err != nil {
						return c.RenderJSON(err)
					}
					break
				}
			}

			if !foundInPendingUsers {
				_, err := bitbucketOperations.AddMember(memberData, uuid)
				if err != nil {
					return c.RenderJSON(err)
				}
			}
		}
	}
	c.Response.Status = 204
	return nil
}

func (c BitbucketGroupsController) UpdateBitbucketGroupMembersMapping() revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	var data bitbucketModel.UpdateBitbucketGroupMembersPayload
	c.Params.BindJSON(&data)

	postBody, _ := json.Marshal(map[string]interface{}{})

	if len(data.ToAdd) > 0 {

		var addData bitbucketModel.BitbucketGroupMemberStruct
		addData = bitbucketModel.BitbucketGroupMemberStruct{
			Workspace: bitbucketWorkspace,
			Token:     bitbucketToken,
			PostBody:  postBody,
		}
		for _, connectedItem := range data.ConnectedItems {
			addData.Slug = connectedItem
			for _, uuid := range data.ToAdd {
				_, err := bitbucketOperations.AddMember(addData, uuid)
				if err != nil {
					//
				}
			}
		}
	}

	if len(data.ToRemove) > 0 {
		var removeData bitbucketModel.BitbucketGroupMemberStruct
		removeData = bitbucketModel.BitbucketGroupMemberStruct{
			Workspace: bitbucketWorkspace,
			Token:     bitbucketToken,
		}
		for _, connectedItem := range data.ConnectedItems {
			removeData.Slug = connectedItem
			for _, uuid := range data.ToRemove {
				err := bitbucketOperations.DeleteMember(removeData, uuid)
				if err != nil {
					//Should handle error ?
					//
				}
			}
		}
	}

	c.Response.Status = 204
	return nil
}
func (c BitbucketGroupsController) DeleteBitbucketGroupMembers(groupSlug, uuid string) revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	removeData := bitbucketModel.BitbucketGroupMemberStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		Slug:      groupSlug,
	}

	err := bitbucketOperations.DeleteMember(removeData, uuid)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err)
	}

	c.Response.Status = 204
	return nil
}
func (c BitbucketGroupsController) RemoveBitbucketGroupConnection(groupID, groupSlug string) revel.Result {
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	integrationID := c.Params.Query.Get("integration_id")
	userIntegrations := c.Params.Query.Get("uids")
	saasMembers := []string{}
	errMarshal := json.Unmarshal([]byte(userIntegrations), &saasMembers)
	_ = errMarshal
	uids, err := ops.GetIntegrationUIDs(ops.GetIntegrationUIDsParams{
		UID:           "",
		IntegrationID: integrationID,
	}, c.Controller)
	if err != nil {
		//TODO Handle Error
	}
	_ = uids

	removeData := bitbucketModel.BitbucketGroupMemberStruct{
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		Slug:      groupSlug,
	}
	for _, saasMember := range saasMembers {
		bErr := bitbucketOperations.DeleteMember(removeData, saasMember)
		if bErr != nil {
			c.Response.Status = 401
			return c.RenderJSON(err)
		}
	}

	jobs.Now(ManageApplications{
		GroupSlug: groupSlug,
		GroupID:   groupID,
		Workspace: bitbucketWorkspace,
		Token:     bitbucketToken,
		Type:      "DELETE",
	})
	return c.RenderJSON("")
}

// **HELPER**//
type BitbucketGroupMembersHelperParams struct {
	Workspace          string
	GroupSlug          string
	Token              string
	IntegrationId      string
	SaasMembers        []string
	CompanyId          string
	PendingInvitations []string
	C                  *revel.Controller
}
type ByAccountID []bitbucketModel.MemberResponse

func (a ByAccountID) Len() int           { return len(a) }
func (a ByAccountID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAccountID) Less(i, j int) bool { return a[i].AccountID < a[j].AccountID }

func BitbucketGroupMembersHelper(data BitbucketGroupMembersHelperParams) ([]bitbucketModel.BitbucketGroupMemberFE, *[]bitbucketModel.MemberResponse, *models.ErrorResponse) {
	var groupMembersFE []bitbucketModel.BitbucketGroupMemberFE
	var groupMembers []bitbucketModel.MemberResponse

	numWorkers := 4 * runtime.NumCPU()

	members, err := bitbucketOperations.GroupMembers(data.Workspace, data.GroupSlug, data.Token)
	if err != nil {
		return nil, nil, err
	}

	if len(*members) == 0 {
		return groupMembersFE, &groupMembers, nil
	}

	sort.Sort(ByAccountID(*members))

	memberChan := make(chan string, len(data.SaasMembers))

	for _, member := range data.SaasMembers {
		memberChan <- member
	}
	close(memberChan)

	var wg sync.WaitGroup
	var mu sync.Mutex
	numGoroutines := numWorkers

	if numWorkers > len(data.SaasMembers) {
		numGoroutines = len(data.SaasMembers)
	}
	wg.Add(numGoroutines)

	for i := 0; i < len(*members); i++ {
		userInfo, err := bitbucketOperations.GetBitbucketUser(data.Token, (*members)[i].AccountID)
		if err == nil {
			(*members)[i].Status = userInfo.AccountStatus
		}
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for saasMember := range memberChan {
				bitbucketUserTemp := bitbucketModel.Users{}
				hasAccount := false
				isExisting := false
				var userEmail string

				companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
					UserID:    saasMember,
					CompanyID: data.CompanyId,
				}, data.C)
				if err != nil {
					// return nil, nil, err
				}
				user, err := ops.GetUserByID(saasMember)
				if err != nil {
					// return nil, nil, err
				}
				userEmail = user.Email
				associatedAccount, ok := companyUser.AssociatedAccounts[constants.INTEG_SLUG_BITBUCKET]

				if ok && len(associatedAccount) != 0 {
					accountStatus := "ACTIVE"
					if associatedAccount[0] != "" {
						hasAccount = true
						// if errCache := cache.Get("bitbucketUserInfo#"+associatedAccount[0], &bitbucketUserTemp); errCache != nil {
						userInfo, err := bitbucketOperations.GetBitbucketUser(data.Token, associatedAccount[0])
						if err == nil {

							membershipBitbucket, err := bitbucketOperations.GetBitbucketUserMembership(data.Token, data.Workspace, userInfo.AccountID)
							if err != nil {
								accountStatus = "INACTIVE"
							} else {

								var membership bitbucketModel.Membership

								membership = *membershipBitbucket

								if membership.User.AccountID != "" {
									if membership.Workspace.Slug != data.Workspace {
										accountStatus = "INACTIVE"
									}
								}

							}

							// go cache.Set(errCacheMembership, *membershipBitbucket, 30*time.Minute)
							// }

							bitbucketUserTemp = bitbucketModel.Users{
								DisplayName:   userInfo.DisplayName,
								UUID:          userInfo.UUID,
								Type:          "user",
								AccountID:     userInfo.AccountID,
								Nickname:      userInfo.Nickname,
								Links:         userInfo.Links,
								AccountStatus: accountStatus,
							}

							// go cache.Set("bitbucketUserInfo#"+associatedAccount[0], bitbucketUserTemp, 30*time.Minute)
						}
						// }
					}
				}
				mu.Lock()
				if hasAccount {

					index := sort.Search(len(*members), func(i int) bool {
						return (*members)[i].AccountID >= associatedAccount[0]
					})

					if index < len((*members)) && (*members)[index].AccountID == associatedAccount[0] {
						foundMember := (*members)[index]
						isExisting = true
						groupMembersFE = append(groupMembersFE, bitbucketModel.BitbucketGroupMemberFE{
							ID:        saasMember,
							Avatar:    foundMember.Avatar,
							AccountID: foundMember.AccountID,
							Name:      foundMember.DisplayName,
							Nickname:  foundMember.Nickname,
							Email:     userEmail,
							Active:    foundMember.IsActive,
							Self:      foundMember.ResourceURI,
							UUID:      saasMember,
							Existing:  isExisting,
							Status:    bitbucketUserTemp.AccountStatus,
						})
						(*members)[index].ConnectedToSaaS = true
						(*members)[index].Email = userEmail
						(*members)[index].Existing = true

					}
				}

				if !isExisting && hasAccount {
					groupMembersFE = append(groupMembersFE, bitbucketModel.BitbucketGroupMemberFE{
						ID:              saasMember,
						Avatar:          "",
						AccountID:       associatedAccount[0],
						Name:            bitbucketUserTemp.DisplayName,
						Nickname:        "",
						Email:           userEmail,
						Active:          false,
						Self:            "",
						UUID:            "",
						Existing:        isExisting,
						Status:          bitbucketUserTemp.AccountStatus,
						ConnectedToSaaS: true,
					})
				}
				if !isExisting && !hasAccount {

					invited := false

					for _, str := range data.PendingInvitations {
						if str == userEmail {
							invited = true
						}
					}

					status := ""

					if invited {
						status = "PENDING"
					}

					groupMembersFE = append(groupMembersFE, bitbucketModel.BitbucketGroupMemberFE{
						ID:              saasMember,
						Avatar:          "",
						AccountID:       "",
						Name:            "",
						Nickname:        "",
						Email:           userEmail,
						Active:          false,
						Self:            "",
						UUID:            "",
						Existing:        isExisting,
						Status:          status,
						ConnectedToSaaS: false,
					})
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	groupMembers = append(groupMembers, *members...)
	return groupMembersFE, &groupMembers, nil
}

// *Jobs **//
type ManageApplications struct {
	GroupSlug string
	GroupID   string
	Workspace string
	Token     string
	Type      string
}

func (c ManageApplications) Run() {
	subIntegrations, err := ops.GetGroupSubIntegration(c.GroupID)
	if err != nil {
		// todo handle error
	}
	postBody, marshalError := json.Marshal(map[string]interface{}{
		"permission": "read",
	})
	if marshalError != nil {
		// TODO handle error
	}
	for _, sub := range subIntegrations {
		// bitbucket projects
		if sub.IntegrationSlug == constants.INTEG_SLUG_BITBUCKET_PROJECTS {
			for _, projectKey := range sub.ConnectedItems {
				// todo handle errors and results
				if c.Type == "ADD" {
					result, err := bitbucketOperations.UpdateProjectGroupPermission(c.Token, c.Workspace, projectKey, c.GroupSlug, postBody)
					_ = result
					_ = err
				}
				if c.Type == "DELETE" {
					result, err := bitbucketOperations.RemoveProjectGroupPermission(c.Token, c.Workspace, projectKey, c.GroupSlug)
					_ = result
					_ = err
				}
			}

		}
		// bitbucket repositories
		if sub.IntegrationSlug == constants.INTEG_SLUG_BITBUCKET_REPOSITORIES {
			for _, repositorySlug := range sub.ConnectedItems {
				// todo handle errors and results
				if c.Type == "ADD" {
					result, err := bitbucketOperations.UpdateRepositoryGroupPermission(c.Token, c.Workspace, repositorySlug, c.GroupSlug, postBody)
					_ = result
					_ = err
				}
				if c.Type == "DELETE" {
					result, err := bitbucketOperations.RemoveRepositoryGroupPermission(c.Token, c.Workspace, repositorySlug, c.GroupSlug)
					_ = err
					_ = result
				}
			}
		}
	}
}

var groupCache = make(map[string]*bitbucketModel.GroupsResponse)

func GetBitbucketGroupHelper(workspace, token, groupSlug string) (*bitbucketModel.GroupsResponse, *models.ErrorResponse) {
	// Check if the group is already in the cache
	if group, ok := groupCache[groupSlug]; ok {
		return group, nil
	}

	groups, bErr := bitbucketOperations.ListGroups(workspace, token)
	if bErr != nil {
		return nil, bErr
	}

	groupSet := make(map[string]*bitbucketModel.GroupsResponse)
	for _, group := range *groups {
		groupCopy := group
		groupSet[group.Slug] = &groupCopy
	}

	group, exists := groupSet[groupSlug]
	if exists {
		// Add the group to the cache
		groupCache[groupSlug] = group
		return group, nil
	}

	return nil, nil
}

type ByGroupID []bitbucketModel.GroupsResponse

func (a ByGroupID) Len() int           { return len(a) }
func (a ByGroupID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByGroupID) Less(i, j int) bool { return a[i].Slug < a[j].Slug }

type ByProjectFE []bitbucketModel.ProjectPermissionsFE

func (a ByProjectFE) Len() int           { return len(a) }
func (a ByProjectFE) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProjectFE) Less(i, j int) bool { return a[i].Slug < a[j].Slug }

func (c BitbucketGroupsController) ViewUserPermissionService(userId string) revel.Result {
	result := make(map[string]interface{})

	var input models.ViewUserPermissionServiceInput
	var viewUserPermissionResult []models.ViewUserPermissionServiceResult

	c.Params.BindJSON(&input)

	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	uuid := c.Params.Query.Get("uuid")
	// include := c.Params.Query.Get("include")
	companyId := c.ViewArgs["companyID"].(string)
	bitbucketGroup := bitbucketModel.GroupsResponse{}
	_ = bitbucketGroup
	bitbucketGroups, bitbucketErr := bitbucketOperations.ListGroups(bitbucketWorkspace, bitbucketToken)
	if bitbucketErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(bitbucketErr)
	}
	sort.Sort(ByGroupID(*bitbucketGroups))

	groups, opsErr := ops.GetUserCompanyGroupsNew(companyId, userId)
	if opsErr != nil {
		c.Response.Status = 400
		return c.RenderJSON(opsErr)
	}
	var wg sync.WaitGroup
	numWorkers := 2 * runtime.NumCPU()
	groupChannel := make(chan string, len(groups))
	for _, group := range groups {
		groupChannel <- group.GroupID
	}
	close(groupChannel)

	numGoroutines := numWorkers
	if numWorkers > len(groups) {
		numGoroutines = len(groups)
	}
	wg.Add(numGoroutines)
	uniqueGroupSlug := []string{}
	uniqueRepositories := []string{}
	uniqueProjects := []string{}
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			for groupID := range groupChannel {
				// if strings.Contains(include, constants.INTEG_SLUG_BITBUCKET_GROUPS) {
				connectedBitbucketGroups, opsErr := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
				if opsErr != nil {
					// c.Response.Status = 400
					// return c.RenderJSON(opsErr)
				}
				if len(connectedBitbucketGroups) != 0 {
					if !(utils.FindStringInSplice(uniqueGroupSlug, connectedBitbucketGroups[0])) {

						index := sort.Search(len(*bitbucketGroups), func(idx int) bool {
							return ((*bitbucketGroups)[idx].Slug) >= connectedBitbucketGroups[0]
						})
						if index < len((*bitbucketGroups)) && (*bitbucketGroups)[index].Slug == connectedBitbucketGroups[0] {

							// for _, group := range *bitbucketGroups {
							// 	if group.Slug == connectedBitbucketGroups[0] {
							group := (*bitbucketGroups)[index]
							viewUserPermissionResult = append(viewUserPermissionResult, models.ViewUserPermissionServiceResult{
								ServiceName:     group.Name,
								ServiceEmail:    "",
								ServiceID:       group.Slug,
								IntegrationSlug: constants.INTEG_SLUG_BITBUCKET_GROUPS,
								Role:            group.Permission,
							})
							bitbucketGroup = group
							// }
							uniqueGroupSlug = append(uniqueGroupSlug, group.Slug)
						}
					}
				}
				// }
				// if strings.Contains(include,  constants.INTEG_SLUG_BITBUCKET_REPOSITORIES) {

				connectedBitbucketRepositories, opsErr := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_REPOSITORIES)
				if opsErr != nil {
					// c.Response.Status = 400
					// return c.RenderJSON(opsErr)
				}
				if len(connectedBitbucketRepositories) != 0 {
					var wgRepo sync.WaitGroup
					repositoryChannel := make(chan string, len(connectedBitbucketRepositories))

					for _, repository := range connectedBitbucketRepositories {
						repositoryChannel <- repository
					}
					close(repositoryChannel)
					repoWorkers := 5
					if len(connectedBitbucketRepositories) < 5 {
						repoWorkers = len(connectedBitbucketRepositories)
					}
					wgRepo.Add(repoWorkers)
					for i := 0; i < repoWorkers; i++ {

						go func() {
							defer wgRepo.Done()
							for repository := range repositoryChannel {
								if !utils.FindStringInSplice(uniqueRepositories, repository) {
									var bitbucketRepository bitbucketModel.Repository
									// if errCache := cache.Get("bitbucketRepositoryInfo#"+repository, &bitbucketRepository); errCache != nil {
									bitbucketRepositoryInfo, bitbucketErr := bitbucketOperations.GetRepository(bitbucketToken, bitbucketWorkspace, repository)
									if bitbucketErr == nil {
										bitbucketRepository = *bitbucketRepositoryInfo
										// go cache.Set("bitbucketRepositoryInfo#"+repository, bitbucketRepository, 30*time.Minute)
									}

									// }
									bitbucketRepositoryUserPermission, bitbucketErr := ListBitbucketRepositoryUsersPermissionsHelper(bitbucketToken, bitbucketWorkspace, repository)
									if bitbucketErr != nil {
										// c.Response.Status = 400
										// return c.RenderJSON(bitbucketErr)
									}

									sort.Sort(ByRepoUserPermissionID(*bitbucketRepositoryUserPermission))

									repoIndex := sort.Search(len(*bitbucketRepositoryUserPermission), func(idx int) bool {
										return ((*bitbucketRepositoryUserPermission)[idx].User.AccountID) >= uuid
									})
									if repoIndex < len((*bitbucketRepositoryUserPermission)) && (*bitbucketRepositoryUserPermission)[repoIndex].User.AccountID == uuid {

										// for _, repoPermission := range *bitbucketRepositoryUserPermission {
										// 	if repoPermission.User.AccountID == uuid {
										repoPermission := (*bitbucketRepositoryUserPermission)[repoIndex]
										viewUserPermissionResult = append(viewUserPermissionResult, models.ViewUserPermissionServiceResult{
											ServiceName:     bitbucketRepository.Name,
											ServiceEmail:    "",
											ServiceID:       repository,
											IntegrationSlug: constants.INTEG_SLUG_BITBUCKET_REPOSITORIES,
											Role:            repoPermission.Permission,
										})
										uniqueRepositories = append(uniqueRepositories, repository)
									}
								}
							}
						}()
					}

					wgRepo.Wait()

				}
				// }
				// if strings.Contains(include,  constants.INTEG_SLUG_BITBUCKET_PROJECTS) {

				connectedBitbucketProjects, opsErr := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_PROJECTS)
				if opsErr != nil {
					// c.Response.Status = 400
					// return c.RenderJSON(opsErr)
				}
				if len(connectedBitbucketProjects) != 0 {
					var wgProj sync.WaitGroup
					projectChannel := make(chan string, len(connectedBitbucketProjects))

					for _, project := range connectedBitbucketProjects {
						projectChannel <- project
					}
					close(projectChannel)

					projWorkers := 5
					if len(connectedBitbucketRepositories) < 5 {
						projWorkers = len(connectedBitbucketRepositories)
					}
					wgProj.Add(projWorkers)
					for i := 0; i < projWorkers; i++ {
						go func() {
							defer wgProj.Done()
							for project := range projectChannel {

								if !utils.FindStringInSplice(uniqueProjects, project) {

									var bitbucketProject bitbucketModel.Project
									// if errCache := cache.Get("bitbucketProjectInfo#"+project, &bitbucketProject); errCache != nil {

									bitbucketProjectInfo, bitbucketErr := bitbucketOperations.GetProject(bitbucketToken, bitbucketWorkspace, project)
									if bitbucketErr == nil {
										bitbucketProject = *bitbucketProjectInfo
										// go cache.Set("bitbucketProjectInfo#"+project, bitbucketProject, 30*time.Minute)
									}

									// }
									projectPermissionsList, _, bitbucketErr := GetBitbucketProjectPermissionHelper(GetBBPUPInput{
										Token:      bitbucketToken,
										Workspace:  bitbucketWorkspace,
										ProjectKey: bitbucketProject.Key,
										Accounts:   []string{uuid},
									})
									sort.Sort(ByProjectFE(*projectPermissionsList))

									if bitbucketErr != nil {
										// c.Response.Status = 400
										// return c.RenderJSON(bitbucketErr)
									}
									// for _, projectPermission := range *projectPermissionsList {

									projIndex := sort.Search(len(*projectPermissionsList), func(idx int) bool {
										return ((*projectPermissionsList)[idx].Slug) >= uuid
									})
									if projIndex < len((*projectPermissionsList)) && (*projectPermissionsList)[projIndex].Slug == uuid {
										projectPermission := (*projectPermissionsList)[projIndex]
										// if projectPermission.Slug == uuid {
										viewUserPermissionResult = append(viewUserPermissionResult, models.ViewUserPermissionServiceResult{
											ServiceName:     bitbucketProject.Name,
											ServiceEmail:    "",
											GroupName:       projectPermission.Name,
											GroupSlug:       uuid,
											ServiceID:       project,
											IntegrationSlug: constants.INTEG_SLUG_BITBUCKET_PROJECTS,
											Role:            projectPermission.Permission,
										})
										uniqueRepositories = append(uniqueRepositories, project)

									}
								}
							}
						}()
					}

					wgProj.Wait()
				}
			}
		}(i)
	}
	wg.Wait()
	c.Response.Status = 200
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	result["data"] = viewUserPermissionResult
	return c.RenderJSON(result)
}
