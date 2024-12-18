package controllers

import (
	"encoding/json"
	"fmt"
	"grooper/app/constants"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	"grooper/app/models"
	bitbucketModel "grooper/app/models/bitbucket"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/revel/revel"
	"github.com/revel/revel/cache"
)

type BitbucketProjectsController struct {
	*revel.Controller
}

// * List Projects in a Workspace
func (c BitbucketProjectsController) ListBitbucketProjects() revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	projects, bErr := bitbucketOperations.ListBitbucketProjectsHelper(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	formattedProjects := []bitbucketModel.ProjectsFormatFE{}
	for _, project := range *projects {
		formattedProjects = append(formattedProjects, bitbucketModel.ProjectsFormatFE{
			ID:          project.UUID,
			Label:       project.Name,
			Value:       project.Key,
			Key:         project.Key,
			Name:        project.Name,
			Description: project.Description,
			IsPrivate:   project.IsPrivate,
			CreatedOn:   project.CreatedOn,
			UpdatedOn:   project.UpdatedOn,
		})
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"projects": formattedProjects,
	})
}

// * List Connected Projects in a Workspace
func (c BitbucketProjectsController) ListConnectedBitbucketProjects(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	accounts := c.Params.Query.Get("uids")
	companyID := c.ViewArgs["companyID"].(string)
	saasMembers := []string{}
	if accounts != "" {
		errMarshal := json.Unmarshal([]byte(accounts), &saasMembers)
		if errMarshal != nil {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "ListConnectedBitbucketProjects Error: Unmarshal Members",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}
	}
	if utils.FindEmptyStringElement([]string{groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "ListBitbucketConnectedProjects Error: Missing required parameter - GroupID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	//* get connected projects
	connectedItems, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_PROJECTS)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketProjects Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}
	//* get connected groups
	connectedGroups, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketGroups Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	//*get group members new (with sub group)
	memberUsers, err := ops.GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_USER)
	if err != nil {
		// TODO: Handle error
	}

	for idx, member := range memberUsers {
		user, err := ops.GetUserByID(member.MemberID)
		if err != nil {

		}
		companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    member.MemberID,
			CompanyID: companyID,
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

	memberGroups, err := ops.GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_GROUP)
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

	//* get projects
	projectsList, bErr := bitbucketOperations.ListBitbucketProjectsHelper(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	//* get connected projects
	formattedProjects := []bitbucketModel.ConnectedProjectsFormatFE{}
	var connectedProjects []bitbucketModel.Project
	for _, project := range *projectsList {
		if utils.StringInSlice(project.Key, connectedItems) {
			formattedProjects = append(formattedProjects, bitbucketModel.ConnectedProjectsFormatFE{})
			connectedProjects = append(connectedProjects, project)
		}
	}

	if len(connectedProjects) != 0 {

		formattedProjectChannel := make(chan bitbucketModel.ConnectedProjectsFormatFE)

		pendingInvitations, err := bitbucketOperations.GetPendingInvitations(bitbucketCredentials.Token, bitbucketCredentials.Workspace)
		if err != nil {
			return c.RenderJSON(err)
		}

		emails := []string{}
		for _, invitation := range pendingInvitations {
			emails = append(emails, invitation.Email)
		}

		for _, cProject := range connectedProjects {

			listProjectPermissions := []bitbucketModel.ProjectPermissionsFE{}
			page := 1

			for {
				projectUserPermissions, err := bitbucketOperations.ListProjectUserPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cProject.Key)
				if err != nil {
					c.Response.Status = 401
					return c.RenderJSON(err)
				}
				// listProjectUserPermissions = append(listProjectUserPermissions, bitbucketModel.ProjectUserPermissions{}.Values...)
				for _, userPermission := range projectUserPermissions.Values {
					listProjectPermissions = append(listProjectPermissions, bitbucketModel.ProjectPermissionsFE{
						Type:       userPermission.User.Type,
						Name:       userPermission.User.Name,
						Slug:       userPermission.User.AccountID,
						Permission: userPermission.Permission,
						Status:     "NOT_SYNCED",
						Email:      "",
						SaaSID:     "",
					})
				}
				if len(projectUserPermissions.Values) <= 0 {
					break
				}
				page++
			}

			page = 1

			for {
				projectGroupPermissions, err := bitbucketOperations.ListProjectGroupPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cProject.Key)
				if err != nil {
					c.Response.Status = 401
					return c.RenderJSON(err)
				}
				for _, groupPermission := range projectGroupPermissions.Values {
					listProjectPermissions = append(listProjectPermissions, bitbucketModel.ProjectPermissionsFE{
						Type:       groupPermission.Group.Type,
						Name:       groupPermission.Group.Name,
						Slug:       groupPermission.Group.Slug,
						Permission: groupPermission.Permission,
						Status:     "NOT_SYNCED",
						SaaSID:     "",
					})
				}
				if len(projectGroupPermissions.Values) <= 0 {
					break
				}
				page++
			}

			connectedProjectsParams := ListConnectedProjectsWithRoutinesParams{
				BitbucketCredentials: bitbucketCredentials,
				ConnectedGroups:      connectedGroups,
				Accounts:             saasMembers,
				Project:              cProject,
				Users:                memberUsers,
				Groups:               memberGroups,
				Permissions:          listProjectPermissions,
				PendingInvitations:   emails,
			}
			go ListConnectedProjectsWithRoutines(connectedProjectsParams, formattedProjectChannel)

			// formattedProjectChannel.ProjectPermissions = append(listProjectGroupPermissions, listProjectUserPermissions)
		}
		for i := 0; i < len(formattedProjects); i++ {
			formattedProjects[i] = <-formattedProjectChannel
		}

	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"projects": formattedProjects,
	})
}
func (c BitbucketProjectsController) GetConnectedBitbucketProjectMember(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	accounts := c.Params.Query.Get("uids")
	companyID := c.ViewArgs["companyID"].(string)
	// userID := c.ViewArgs["userID"].(string)
	groupID := c.Params.Query.Get("groupID")
	saasMembers := []string{}
	if accounts != "" {
		errMarshal := json.Unmarshal([]byte(accounts), &saasMembers)
		if errMarshal != nil {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "GetConnectedBitbucketProjectMember Error: Unmarshal Members",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}
	}
	getProject, opsErr := bitbucketOperations.GetProject(bitbucketCredentials.Token, bitbucketCredentials.Workspace, projectKey)
	if opsErr != nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "GetConnectedBitbucketProjectMember Error: ProjectKey not found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}
	// formattedProject := bitbucketModel.ConnectedProjectsFormatFE{}
	formattedProjectChannel := make(chan bitbucketModel.ConnectedProjectsFormatFE)
	formattedProjects := []bitbucketModel.ConnectedProjectsFormatFE{}
	var connectedProjects []bitbucketModel.Project

	formattedProjects = append(formattedProjects, bitbucketModel.ConnectedProjectsFormatFE{})
	connectedProjects = append(connectedProjects, *getProject)

	//*get group members new (with sub group)
	memberUsers, err := ops.GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_USER)
	if err != nil {
		// TODO: Handle error
	}

	for idx, member := range memberUsers {
		user, err := ops.GetUserByID(member.MemberID)
		if err != nil {

		}
		companyUser, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    member.MemberID,
			CompanyID: companyID,
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

	memberGroups, err := ops.GetGroupMembersNew(companyID, groupID, constants.MEMBER_TYPE_GROUP)
	if err != nil {
		// TODO: Handle error
	}

	// gcm := ops.GetCompanyMembersParams{UserID: userID, CompanyID: companyID}
	// companyUsers, opsError := ops.GetCompanyMembers(gcm, c.Controller)
	// if opsError != nil {
	// }

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

	pendingInvitations, error := bitbucketOperations.GetPendingInvitations(bitbucketCredentials.Token, bitbucketCredentials.Workspace)
	if error != nil {
		return c.RenderJSON(error)
	}

	emails := []string{}
	for _, invitation := range pendingInvitations {
		emails = append(emails, invitation.Email)
	}

	for _, cProject := range connectedProjects {

		listProjectPermissions := []bitbucketModel.ProjectPermissionsFE{}
		page := 1

		for {
			projectUserPermissions, err := bitbucketOperations.ListProjectUserPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cProject.Key)
			if err != nil {
				c.Response.Status = 401
				return c.RenderJSON(err)
			}
			// listProjectUserPermissions = append(listProjectUserPermissions, bitbucketModel.ProjectUserPermissions{}.Values...)
			for _, userPermission := range projectUserPermissions.Values {
				listProjectPermissions = append(listProjectPermissions, bitbucketModel.ProjectPermissionsFE{
					Type:       userPermission.User.Type,
					Name:       userPermission.User.Name,
					Slug:       userPermission.User.AccountID,
					Permission: userPermission.Permission,
					Status:     "NOT_SYNCED",
					Email:      "",
					SaaSID:     "",
				})
			}
			if len(projectUserPermissions.Values) <= 0 {
				break
			}
			page++
		}

		page = 1

		for {
			projectGroupPermissions, err := bitbucketOperations.ListProjectGroupPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cProject.Key)
			if err != nil {
				c.Response.Status = 401
				return c.RenderJSON(err)
			}
			for _, groupPermission := range projectGroupPermissions.Values {
				listProjectPermissions = append(listProjectPermissions, bitbucketModel.ProjectPermissionsFE{
					Type:       groupPermission.Group.Type,
					Name:       groupPermission.Group.Name,
					Slug:       groupPermission.Group.Slug,
					Permission: groupPermission.Permission,
					Status:     "NOT_SYNCED",
					SaaSID:     "",
				})
			}
			if len(projectGroupPermissions.Values) <= 0 {
				break
			}
			page++
		}

		connectedProjectsParams := ListConnectedProjectsWithRoutinesParams{
			BitbucketCredentials: bitbucketCredentials,
			ConnectedGroups:      []string{},
			Accounts:             saasMembers,
			Project:              cProject,
			Users:                memberUsers,
			Groups:               memberGroups,
			Permissions:          listProjectPermissions,
			PendingInvitations:   emails,
		}
		go ListConnectedProjectsWithRoutines(connectedProjectsParams, formattedProjectChannel)
		// formattedProjectChannel.ProjectPermissions = append(listProjectGroupPermissions, listProjectUserPermissions)
	}
	for i := 0; i < len(formattedProjects); i++ {
		formattedProjects[i] = <-formattedProjectChannel
	}

	c.Response.Status = 200
	return c.RenderJSON(formattedProjects[0])
}

type ListConnectedProjectsWithRoutinesParams struct {
	BitbucketCredentials bitbucketModel.BitbucketCredentials
	ConnectedGroups      []string
	Accounts             []string
	Project              bitbucketModel.Project
	Users                []models.GroupMember
	Groups               []models.GroupMember
	Permissions          []bitbucketModel.ProjectPermissionsFE ///User and Group memebrs of Project
	PendingInvitations   []string
}

func ListConnectedProjectsWithRoutines(params ListConnectedProjectsWithRoutinesParams, projectDataChannel chan<- bitbucketModel.ConnectedProjectsFormatFE) {

	projectPermissionsList, nonExistingPermissionsCount, bErr := GetBitbucketProjectPermissionHelper(GetBBPUPInput{
		Token:              params.BitbucketCredentials.Token,
		Workspace:          params.BitbucketCredentials.Workspace,
		ProjectKey:         params.Project.Key,
		Accounts:           params.Accounts,
		Users:              params.Users,
		Groups:             params.Groups,
		PermissionList:     params.Permissions,
		PendingInvitations: params.PendingInvitations,
		// ConnectedGroups: connectedGroups,
	})
	if bErr != nil {
		// c.Response.Status = 401
		// return c.RenderJSON(bErr)
	}
	cProject := params.Project
	formattedProject := bitbucketModel.ConnectedProjectsFormatFE{
		ID:                          cProject.UUID,
		Label:                       cProject.Name,
		Value:                       cProject.Key,
		Key:                         cProject.Key,
		Name:                        cProject.Name,
		Description:                 cProject.Description,
		IsPrivate:                   cProject.IsPrivate,
		CreatedOn:                   cProject.CreatedOn,
		UpdatedOn:                   cProject.UpdatedOn,
		ProjectPermissions:          params.Permissions,
		SaaSMembersPermissions:      *projectPermissionsList,
		NonExistingPermissionsCount: nonExistingPermissionsCount,
	}
	projectDataChannel <- formattedProject
}

// * Get Project in a Workspace
func (c BitbucketProjectsController) GetBitbucketProject(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	projectKey := c.Params.Query.Get("projectKey")

	if utils.FindEmptyStringElement([]string{projectKey, groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "GetBitbucketProject Error: Missing required parameter - ProjectKey/GroupID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	// * get connected groups
	connectedGroups, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketGroups Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	// * test data
	// connectedGroups := []string {
	// 	"developers",
	// 	"sample",
	// }

	trimmedProjectKey := strings.TrimSpace(strings.ToUpper(projectKey))

	//* get project
	project, bErr := bitbucketOperations.GetProject(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		trimmedProjectKey,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	projectPermissionsList, nonExistingPermissionsCount, bErr := GetBitbucketProjectGroupPermissionHelper(GetBBPGPInput{
		Token:           bitbucketCredentials.Token,
		Workspace:       bitbucketCredentials.Workspace,
		ProjectKey:      project.Key,
		ConnectedGroups: connectedGroups,
	})
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	formattedProject := bitbucketModel.ConnectedProjectsFormatFE{
		ID:                          project.UUID,
		Label:                       project.Name,
		Value:                       project.Key,
		Key:                         project.Key,
		Description:                 project.Description,
		IsPrivate:                   project.IsPrivate,
		CreatedOn:                   project.CreatedOn,
		UpdatedOn:                   project.UpdatedOn,
		Name:                        project.Name,
		ProjectPermissions:          *projectPermissionsList,
		NonExistingPermissionsCount: nonExistingPermissionsCount,
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"project": formattedProject,
	})
}

// * Create Project in a Workspace
type CreateBitbucketProjectInput struct {
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	Description string   `json:"description"`
	IsPrivate   bool     `json:"is_private"`
	Members     []string `json:"uuids"`
	Users       []string `json:"group_users"`
	Groups      []string `json:"group_sub_groups"`
}

func (c BitbucketProjectsController) CreateBitbucketProject(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input CreateBitbucketProjectInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Name, input.Key, groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "CreateBitbucketProject Error: Missing required parameter - Name/Key/GroupID.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedProjectName := strings.TrimSpace(input.Name)
	trimmedProjectKey := strings.TrimSpace(strings.ToUpper(input.Key))
	trimmedProjectDescription := strings.TrimSpace(input.Description)

	//* Get Connected Bitbucket Groups
	// connectedGroups, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
	// if err != nil {
	// 	c.Response.Status = 500
	// 	return c.RenderJSON(models.ErrorResponse{
	// 		HTTPStatusCode: 500,
	// 		Message:        "GetGroups Error: GetConnectedItemBySlug : " + err.Error(),
	// 		Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
	// 	})
	// }

	postBody, marshalError := json.Marshal(map[string]interface{}{
		"name":        trimmedProjectName,
		"key":         trimmedProjectKey,
		"description": trimmedProjectDescription,
		"is_private":  input.IsPrivate,
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "CreateBitbucketProject Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	createdProject, bErr := bitbucketOperations.CreateProject(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		postBody,
	)
	if bErr != nil {
		c.Response.Status = bErr.HTTPStatusCode
		return c.RenderJSON(bErr)
	}

	//* add connected groups to project permissions
	//* set permission on the added groups.
	var addedUserPermissions []*bitbucketModel.ProjectUserPermission
	var addedUserPermissionsErrors []*bitbucketModel.ProjectUserPermission
	var addedGroupPermissions []*bitbucketModel.ProjectGroupPermission
	var addedGroupPermissionsError []*bitbucketModel.ProjectGroupPermission
	// if len(connectedGroups) != 0 {
	if len(input.Users) != 0 {
		postBodyPGP, marshalError := json.Marshal(map[string]interface{}{
			"permission": "read", // read is the default permission
		})
		if marshalError != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        "UpdateBitbucketProjectUserPermissions Error: Error on marshalling data.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}

		for _, account := range input.Users {
			updatePGP, bErr := bitbucketOperations.UpdateProjectUserPermission(
				bitbucketCredentials.Token,
				bitbucketCredentials.Workspace,
				createdProject.Key,
				account,
				postBodyPGP,
			)
			if bErr != nil {
				addedUserPermissionsErrors = append(addedUserPermissionsErrors, &bitbucketModel.ProjectUserPermission{
					User: bitbucketModel.ProjectUserPermissionInfo{
						AccountID: account,
					},
					Permission: "read",
				})

			}

			addedUserPermissions = append(addedUserPermissions, updatePGP)
		}
	}

	if len(input.Groups) != 0 {
		postBodyPGP, marshalError := json.Marshal(map[string]interface{}{
			"permission": "read", // read is the default permission
		})
		if marshalError != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        "UpdateBitbucketProjectUserPermissions Error: Error on marshalling data.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}

		for _, account := range input.Groups {
			updatePGP, bErr := bitbucketOperations.UpdateProjectGroupPermission(
				bitbucketCredentials.Token,
				bitbucketCredentials.Workspace,
				createdProject.Key,
				account,
				postBodyPGP,
			)
			if bErr != nil {
				addedGroupPermissionsError = append(addedGroupPermissionsError, &bitbucketModel.ProjectGroupPermission{
					Group: bitbucketModel.ProjectGroupPermissionInfo{
						Slug: account,
					},
					Permission: "read",
				})

			}

			addedGroupPermissions = append(addedGroupPermissions, updatePGP)
		}
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"project":                        createdProject,
		"user_permissions_added":         addedUserPermissions,
		"user_permissions_added_errors":  addedUserPermissionsErrors,
		"group_permissions_added":        addedGroupPermissions,
		"group_permissions_added_errors": addedGroupPermissionsError,
	})
}

// * Update Project in Workspace
type UpdateBitbucketProjectInput struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

func (c BitbucketProjectsController) UpdateBitbucketProject(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	var input UpdateBitbucketProjectInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Name, input.Key, projectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketProject Error: Missing required parameter - Name/Key/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedProjectName := strings.TrimSpace(input.Name)
	trimmedProjectKey := strings.TrimSpace(strings.ToUpper(input.Key))
	trimmedProjectDescription := strings.TrimSpace(input.Description)

	postBody, marshalError := json.Marshal(map[string]interface{}{
		"name":        trimmedProjectName,
		"description": trimmedProjectDescription,
		"key":         trimmedProjectKey,
		"is_private":  input.IsPrivate,
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "UpdateBitbucketProject Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	updatedProject, bErr := bitbucketOperations.UpdateProject(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		projectKey,
		postBody,
	)
	if bErr != nil {
		c.Response.Status = bErr.HTTPStatusCode
		return c.RenderJSON(bErr)
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"project": updatedProject,
	})
}

// * Update Project Group Permissions
type UpdateBitbucketProjectGroupPermissionsInput struct {
	GroupSlug  string `json:"group_slug"`
	Permission string `json:"permission"`
	ID         string `json:"id"` // NEW PARAM INSTEAD OF GROUP SLUG
}

func (c BitbucketProjectsController) UpdateBitbucketProjectGroupPermission(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input UpdateBitbucketProjectGroupPermissionsInput
	c.Params.BindJSON(&input)

	input.GroupSlug = input.ID // HANDLE BITBUCKET CHANGES

	if utils.FindEmptyStringElement([]string{input.GroupSlug, input.Permission, projectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketProjectGroupPermissions Error: Missing required parameter - GroupSlug/Permission/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedGroupSlug := strings.TrimSpace(input.GroupSlug)
	trimmedPermission := strings.TrimSpace(input.Permission)

	postBodyPGP, marshalError := json.Marshal(map[string]interface{}{
		"permission": trimmedPermission,
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "UpdateBitbucketProjectGroupPermissions Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	updatedPGP, bErr := bitbucketOperations.UpdateProjectGroupPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		projectKey,
		trimmedGroupSlug,
		postBodyPGP,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}
	listFormattedProjectPermissions := bitbucketModel.ProjectPermissionsFE{
		Type:       updatedPGP.Group.Type,
		Name:       updatedPGP.Group.Name,
		Slug:       updatedPGP.Group.Slug,
		Permission: updatedPGP.Permission,
		IsExisting: true,
		Status:     "ACTIVE",
	}
	go cache.Set("projectPermission#"+projectKey+"#"+input.GroupSlug, listFormattedProjectPermissions, 10*time.Minute)

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"project_gp": updatedPGP,
	})
}

// * Delete Project Group Permission
func (c BitbucketProjectsController) RemoveBitbucketProjectGroupPermission(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	groupSlug := c.Params.Query.Get("group_slug")

	if utils.FindEmptyStringElement([]string{groupSlug, projectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "RemoveBitbucketProjectGroupPermission Error: Missing required parameter - GroupSlug/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedGroupSlug := strings.TrimSpace(groupSlug)

	_, bErr := bitbucketOperations.RemoveProjectGroupPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		projectKey,
		trimmedGroupSlug,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}
	go cache.Delete("projectPermission#" + projectKey + "#" + groupSlug)
	c.Response.Status = 204
	return nil
}

func ListBitbucketProjectGroupPermissionsHelper(token, workspace, projectKey string) (*[]bitbucketModel.ProjectGroupPermission, *models.ErrorResponse) {
	listProjectGroupPermissions := []bitbucketModel.ProjectGroupPermission{}

	page := 1
	for {
		projectGroupPermissions, err := bitbucketOperations.ListProjectGroupPermissions(token, strconv.Itoa(page), workspace, projectKey)
		if err != nil {
			return nil, err
		}
		listProjectGroupPermissions = append(listProjectGroupPermissions, projectGroupPermissions.Values...)
		if len(projectGroupPermissions.Values) <= 0 {
			break
		}
		page++
	}
	return &listProjectGroupPermissions, nil
}

type GetBBPGPInput struct {
	Token           string   `json:"token"`
	Workspace       string   `json:"workspace"`
	ProjectKey      string   `json:"project_key"`
	ConnectedGroups []string `json:"connected_groups"`
}

func GetBitbucketProjectGroupPermissionHelper(input GetBBPGPInput) (*[]bitbucketModel.ProjectPermissionsFE, int, *models.ErrorResponse) {
	listFormattedProjectPermissions := []bitbucketModel.ProjectPermissionsFE{}
	nonExistingPermissionsCount := 0
	if len(input.ConnectedGroups) != 0 {

		// if errCache := cache.Get("projectPermission#"+input.ProjectKey+"#"+input.ConnectedGroups[0], listFormattedProjectPermissions); errCache != nil {

		listPGP, bErr := ListBitbucketProjectGroupPermissionsHelper(
			input.Token,
			input.Workspace,
			input.ProjectKey,
		)
		if bErr != nil {
			return nil, 0, bErr
		}

		for _, groupSlug := range input.ConnectedGroups {
			isExistOnProject := false
			for _, projectPermission := range *listPGP {
				if groupSlug == projectPermission.Group.Slug {
					listFormattedProjectPermissions = append(listFormattedProjectPermissions, bitbucketModel.ProjectPermissionsFE{
						Type:       projectPermission.Group.Type,
						Name:       projectPermission.Group.Name,
						Slug:       projectPermission.Group.Slug,
						Permission: projectPermission.Permission,
						IsExisting: true,
						Status:     "ACTIVE",
					})
					isExistOnProject = true
					break
				}
			}

			if !isExistOnProject {
				groupInfo, err := GetBitbucketGroupHelper(
					input.Workspace,
					input.Token,
					groupSlug,
				)
				if err != nil {
					return nil, 0, err
				}

				if groupInfo != nil {
					listFormattedProjectPermissions = append(listFormattedProjectPermissions, bitbucketModel.ProjectPermissionsFE{
						Name:       groupInfo.Name,
						Slug:       groupSlug,
						Type:       "group",
						IsExisting: false,
						Status:     "NOT_SYNCED",
					})
					nonExistingPermissionsCount++
				}
			}
			// 	go cache.Set("projectPermission#"+input.ProjectKey+"#"+input.ConnectedGroups[0], listFormattedProjectPermissions, 10*time.Minute)

			// }
		}
	}

	return &listFormattedProjectPermissions, nonExistingPermissionsCount, nil
}

type UpdateBitbucketProjectUserPermissionsInput struct {
	Account    string `json:"id"`
	Permission string `json:"permission"`
}

func (c BitbucketProjectsController) UpdateBitbucketProjectUserPermission(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input UpdateBitbucketProjectUserPermissionsInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Account, input.Permission, projectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketProjectUserPermission Error: Missing required parameter - Account/Permission/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedAccount := strings.TrimSpace(input.Account)
	trimmedPermission := strings.TrimSpace(input.Permission)

	postBodyPUP, marshalError := json.Marshal(map[string]interface{}{
		"permission": trimmedPermission,
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "UpdateBitbucketProjectGroupPermissions Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}
	var membership bitbucketModel.Membership
	isExistOnBitbucket := false
	// errCacheMembership := "bitbucketMembershipUserInfo#" + trimmedAccount
	// if errCache := cache.Get(errCacheMembership, &membership); errCache != nil {

	membershipBitbucket, err := bitbucketOperations.GetBitbucketUserMembership(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		trimmedAccount,
	)
	if err == nil {

		membership = *membershipBitbucket
		if membership.User.AccountID != "" {
			isExistOnBitbucket = true
		}
		// go cache.Set(errCacheMembership, *membershipBitbucket, 10*time.Minute)
	} else {
		// go cache.Set(errCacheMembership, membership, 10*time.Minute)
	}
	// }
	if membership.User.AccountID != "" {
		isExistOnBitbucket = true
	}
	var updatedPUP *bitbucketModel.ProjectUserPermission
	if isExistOnBitbucket {

		updatedPUP, bErr := bitbucketOperations.UpdateProjectUserPermission(
			bitbucketCredentials.Token,
			bitbucketCredentials.Workspace,
			projectKey,
			trimmedAccount,
			postBodyPUP,
		)
		if bErr != nil {
			c.Response.Status = 401
			return c.RenderJSON(bErr)
		}

		listFormattedProjectPermissions := bitbucketModel.ProjectPermissionsFE{
			Type:       updatedPUP.User.Type,
			Name:       updatedPUP.User.Name,
			Slug:       updatedPUP.User.AccountID,
			Permission: updatedPUP.Permission,
			IsExisting: true,
			Status:     "ACTIVE",
		}
		go cache.Set("projectPermission#"+projectKey+"#"+input.Account, listFormattedProjectPermissions, 10*time.Minute)
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"project_gp": updatedPUP,
	})
}

// * Delete Project User Permission
func (c BitbucketProjectsController) RemoveBitbucketProjectUserPermission(projectKey string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	account := c.Params.Query.Get("id")

	if utils.FindEmptyStringElement([]string{account, projectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "RemoveBitbucketProjectUserPermission Error: Missing required parameter - Account/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedAccount := strings.TrimSpace(account)

	_, bErr := bitbucketOperations.RemoveProjectUserPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		projectKey,
		trimmedAccount,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}
	go cache.Delete("projectPermission#" + projectKey + "#" + account)
	c.Response.Status = 204
	return nil
}

func ListBitbucketProjectUserPermissionsHelper(token, workspace, projectKey string) (*[]bitbucketModel.ProjectUserPermission, *models.ErrorResponse) {
	listProjectUserPermissions := []bitbucketModel.ProjectUserPermission{}

	page := 1
	for {
		projectUserPermissions, err := bitbucketOperations.ListProjectUserPermissions(token, strconv.Itoa(page), workspace, projectKey)
		if err != nil {
			return nil, err
		}
		listProjectUserPermissions = append(listProjectUserPermissions, projectUserPermissions.Values...)
		if len(projectUserPermissions.Values) <= 0 {
			break
		}
		page++
	}
	return &listProjectUserPermissions, nil
}

type GetBBPUPInput struct {
	Token              string   `json:"token"`
	Workspace          string   `json:"workspace"`
	ProjectKey         string   `json:"project_key"`
	Accounts           []string `json:"accounts"`
	Users              []models.GroupMember
	Groups             []models.GroupMember
	PermissionList     []bitbucketModel.ProjectPermissionsFE
	PendingInvitations []string
}

type ByProjectID []bitbucketModel.ProjectUserPermission

func (a ByProjectID) Len() int           { return len(a) }
func (a ByProjectID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProjectID) Less(i, j int) bool { return a[i].User.AccountID < a[j].User.AccountID }
func GetBitbucketProjectPermissionHelper(input GetBBPUPInput) (*[]bitbucketModel.ProjectPermissionsFE, int, *models.ErrorResponse) {
	var (
		listFormattedProjectPermissions []bitbucketModel.ProjectPermissionsFE
		nonExistingPermissionsCount     int
		wg                              sync.WaitGroup
		mu                              sync.Mutex
	)

	// Fetch user permissions concurrently
	if len(input.Users) != 0 {
		userPermissionsChan := make(chan bitbucketModel.ProjectPermissionsFE)
		errorChan := make(chan *models.ErrorResponse, len(input.Users))

		for _, user := range input.Users {
			wg.Add(1)
			go func(user models.GroupMember) {
				defer wg.Done()
				processUserPermissions(user, input, nonExistingPermissionsCount, userPermissionsChan, errorChan, input.PendingInvitations)
			}(user)
		}

		go func() {
			wg.Wait()
			close(userPermissionsChan)
			close(errorChan)
		}()

		for perm := range userPermissionsChan {
			mu.Lock()
			listFormattedProjectPermissions = append(listFormattedProjectPermissions, perm)
			mu.Unlock()
		}

		for err := range errorChan {
			if err != nil {
				fmt.Println("Error fetching user permissions:", err)
			}
		}
	}

	// Fetch group permissions concurrently
	if len(input.Groups) != 0 {
		groupList, bErr := bitbucketOperations.ListGroups(input.Workspace, input.Token)
		if bErr != nil {
			return nil, 0, bErr
		}

		groupPermissionsChan := make(chan bitbucketModel.ProjectPermissionsFE)
		for _, group := range input.Groups {
			wg.Add(1)
			go func(group models.GroupMember) {
				defer wg.Done()
				processGroupPermissions(group, input, groupList, nonExistingPermissionsCount, groupPermissionsChan)
			}(group)
		}

		go func() {
			wg.Wait()
			close(groupPermissionsChan)
		}()

		for perm := range groupPermissionsChan {
			mu.Lock()
			listFormattedProjectPermissions = append(listFormattedProjectPermissions, perm)
			mu.Unlock()
		}
	}

	wg.Wait() // Ensure all goroutines have finished before proceeding

	return &listFormattedProjectPermissions, nonExistingPermissionsCount, nil
}

func processUserPermissions(user models.GroupMember, input GetBBPUPInput, nonExistingPermissionsCount int, userPermissionsChan chan<- bitbucketModel.ProjectPermissionsFE, errorChan chan<- *models.ErrorResponse, pendingInvitations []string) {
	associatedAccount := user.MemberInformation.AssociatedAccounts[constants.INTEG_SLUG_BITBUCKET]
	accountStatus := "ACTIVE"
	if len(associatedAccount) == 0 {
		invited := false

		for _, str := range pendingInvitations {
			if str == user.MemberInformation.Email {
				invited = true
			}
		}

		accountStatus = "NOT_SYNCED"

		if invited {
			accountStatus = "PENDING"
		}

		userPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Name:       "",
			Slug:       "",
			Type:       "user",
			IsExisting: false,
			Status:     accountStatus,
			Email:      user.MemberInformation.Email,
			SaaSID:     user.MemberID,
		}
		nonExistingPermissionsCount++
		return
	}

	userInfo, err := bitbucketOperations.GetBitbucketUser(input.Token, associatedAccount[0])
	if err != nil {
		errorChan <- err
		return
	}

	membershipBitbucket, err := bitbucketOperations.GetBitbucketUserMembership(input.Token, input.Workspace, userInfo.AccountID)
	if err != nil {
		accountStatus = "INACTIVE"
	} else {

		var membership bitbucketModel.Membership

		membership = *membershipBitbucket

		if membership.User.AccountID != "" {
			if membership.Workspace.Slug != input.Workspace {
				accountStatus = "INACTIVE"
			}
		}

	}

	bitbucketUserTemp := bitbucketModel.Users{
		DisplayName:   userInfo.DisplayName,
		UUID:          userInfo.UUID,
		Type:          "user",
		AccountID:     userInfo.AccountID,
		Nickname:      userInfo.Nickname,
		Links:         userInfo.Links,
		AccountStatus: accountStatus,
	}
	go cache.Set("bitbucketUserInfo#"+associatedAccount[0], bitbucketUserTemp, 10*time.Minute)

	index := -1
	for i, permission := range input.PermissionList {
		fmt.Println(permission.Slug)
		if permission.Slug == associatedAccount[0] {
			index = i
			break
		}
	}

	if index == -1 {
		index = len(input.PermissionList) + 1
	}

	fmt.Println("User associatedACcount", associatedAccount)
	fmt.Println(index, "index")

	if index < len(input.PermissionList) && input.PermissionList[index].Slug == associatedAccount[0] {

		if input.PermissionList[index].Permission != "none" {
			input.PermissionList[index].ConnectedToSaaS = true
			input.PermissionList[index].Status = bitbucketUserTemp.AccountStatus
			input.PermissionList[index].Email = user.MemberInformation.Email
			input.PermissionList[index].SaaSID = user.MemberID
			input.PermissionList[index].IsExisting = true
			userPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
				Type:       input.PermissionList[index].Type,
				Name:       input.PermissionList[index].Name,
				Slug:       input.PermissionList[index].Slug,
				Permission: input.PermissionList[index].Permission,
				Email:      user.MemberInformation.Email,
				SaaSID:     user.MemberID,
				IsExisting: true,
				Status:     bitbucketUserTemp.AccountStatus,
			}

		} else {
			handleNonExistingUserPermissions(user, bitbucketUserTemp, associatedAccount[0], nonExistingPermissionsCount, userPermissionsChan)
		}
	} else {
		handleNonExistingUserPermissions(user, bitbucketUserTemp, associatedAccount[0], nonExistingPermissionsCount, userPermissionsChan)
	}
}

func processGroupPermissions(group models.GroupMember, input GetBBPUPInput, groupList *[]bitbucketModel.GroupsResponse, nonExistingPermissionsCount int, groupPermissionsChan chan<- bitbucketModel.ProjectPermissionsFE) {

	groupIntegrationIndex := -1
	for i, integration := range group.GroupIntegrations {
		if integration.IntegrationSlug == "bitbucket" {
			groupIntegrationIndex = i
			break
		}
	}

	if groupIntegrationIndex == -1 {
		groupPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Type:       "group",
			Name:       "",
			Slug:       "",
			SaaSID:     group.MemberID,
			SaaSName:   group.GroupName,
			IsExisting: false,
			Status:     "NOT_SYNCED",
		}
		nonExistingPermissionsCount++
		return
	}

	// Linear search for groupSubIntegrationIndex
	groupSubIntegrationIndex := -1
	for i, subIntegration := range group.GroupIntegrations[groupIntegrationIndex].GroupSubIntegrations {
		if subIntegration.IntegrationSlug == "bitbucket-groups" {
			groupSubIntegrationIndex = i
			break
		}
	}

	if groupSubIntegrationIndex == -1 {
		return
	}

	groupAssociatedAccount := group.GroupIntegrations[groupIntegrationIndex].GroupSubIntegrations[groupSubIntegrationIndex].ConnectedItems

	// Linear search for groupPermissionIndex
	groupPermissionIndex := -1
	for i, permission := range input.PermissionList {
		if permission.Slug == groupAssociatedAccount[0] {
			groupPermissionIndex = i
			break
		}
	}

	if groupPermissionIndex == -1 {
		groupPermissionIndex = len(input.PermissionList) + 1
	}

	if groupPermissionIndex < len(input.PermissionList) && input.PermissionList[groupPermissionIndex].Slug == groupAssociatedAccount[0] {
		if input.PermissionList[groupPermissionIndex].Permission != "none" {
			input.PermissionList[groupPermissionIndex].ConnectedToSaaS = true
			input.PermissionList[groupPermissionIndex].SaaSName = group.GroupName
			input.PermissionList[groupPermissionIndex].Status = "ACTIVE"
			input.PermissionList[groupPermissionIndex].SaaSID = group.MemberID
			input.PermissionList[groupPermissionIndex].IsExisting = true
			groupPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
				Type:       input.PermissionList[groupPermissionIndex].Type,
				Name:       input.PermissionList[groupPermissionIndex].Name,
				Slug:       input.PermissionList[groupPermissionIndex].Slug,
				Permission: input.PermissionList[groupPermissionIndex].Permission,
				SaaSID:     group.MemberID,
				SaaSName:   group.GroupName,
				IsExisting: true,
				Status:     "ACTIVE",
			}
		} else {
			handleNonExistingGroupPermissions(group, groupList, groupAssociatedAccount[0], nonExistingPermissionsCount, groupPermissionsChan)
		}
	} else {
		handleNonExistingGroupPermissions(group, groupList, groupAssociatedAccount[0], nonExistingPermissionsCount, groupPermissionsChan)
	}
}

func handleNonExistingUserPermissions(user models.GroupMember, bitbucketUserTemp bitbucketModel.Users, associatedAccount string, nonExistingPermissionsCount int, userPermissionsChan chan<- bitbucketModel.ProjectPermissionsFE) {
	if bitbucketUserTemp.AccountStatus == "ACTIVE" {
		userPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Name:       bitbucketUserTemp.DisplayName,
			Slug:       associatedAccount,
			Type:       "user",
			IsExisting: false,
			Status:     "NOT_SYNCED",
			Email:      user.MemberInformation.Email,
			SaaSID:     user.MemberID,
		}
	} else if bitbucketUserTemp.AccountStatus == "INACTIVE" {
		userPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Name:       bitbucketUserTemp.DisplayName,
			Slug:       associatedAccount,
			Type:       "user",
			IsExisting: false,
			Status:     "DEACTIVATED",
			Email:      user.MemberInformation.Email,
			SaaSID:     user.MemberID,
		}
	}
	nonExistingPermissionsCount++
}

func handleNonExistingGroupPermissions(group models.GroupMember, groupList *[]bitbucketModel.GroupsResponse, groupAccount string, nonExistingPermissionsCount int, groupPermissionsChan chan<- bitbucketModel.ProjectPermissionsFE) {
	fmt.Println("Group ACcount", groupAccount)
	existingGroupIndex := -1
	for i, group := range *groupList {
		if group.Slug == groupAccount {
			existingGroupIndex = i
			break
		}
	}

	if existingGroupIndex == -1 {
		existingGroupIndex = len((*groupList)) + 1
	}

	fmt.Println("FIND", ((*groupList)[existingGroupIndex]))

	if existingGroupIndex < len(*groupList) && (*groupList)[existingGroupIndex].Slug == groupAccount {
		groupPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Name:            (*groupList)[existingGroupIndex].Name,
			Slug:            groupAccount,
			Type:            "group",
			IsExisting:      false,
			Status:          "NOT_SYNCED",
			ConnectedToSaaS: false,
			SaaSName:        group.GroupName,
			SaaSID:          group.MemberID,
		}
	} else {
		groupPermissionsChan <- bitbucketModel.ProjectPermissionsFE{
			Name:       "",
			Slug:       groupAccount,
			Type:       "group",
			IsExisting: false,
			Status:     "DEACTIVATED",
			SaaSName:   group.GroupName,
			SaaSID:     group.MemberID,
		}
	}
	nonExistingPermissionsCount++
}
