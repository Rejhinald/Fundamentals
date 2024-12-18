package controllers

import (
	"encoding/json"
	"grooper/app/constants"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	bitbucketModel "grooper/app/models/bitbucket"

	"github.com/revel/revel"
	"github.com/revel/revel/cache"
)

type BitbucketRepositoriesController struct {
	*revel.Controller
}

func (c BitbucketRepositoriesController) ListBitbucketRepositories() revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	repositories, bErr := bitbucketOperations.ListBitbucketRepositoriesHelper(
		bitbucketCredentials.Workspace,
		bitbucketCredentials.Token,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	formattedRepositories := []bitbucketModel.RepositoryFormatFE{}
	for _, repo := range *repositories {
		formattedRepositories = append(formattedRepositories, bitbucketModel.RepositoryFormatFE{
			ID:          repo.UUID,
			Label:       repo.Name,
			Value:       repo.Slug,
			Slug:        repo.Slug,
			Name:        repo.Name,
			Description: repo.Description,
			IsPrivate:   repo.IsPrivate,
			CreatedOn:   repo.CreatedOn,
			LastUpdated: repo.LastUpdated,
			Project: bitbucketModel.ProjectsFormatFE{
				ID:          repo.Project.UUID,
				Label:       repo.Project.Name,
				Value:       repo.Project.Key,
				Key:         repo.Project.Key,
				Name:        repo.Project.Name,
				Description: repo.Project.Description,
				IsPrivate:   repo.Project.IsPrivate,
				CreatedOn:   repo.Project.CreatedOn,
				UpdatedOn:   repo.Project.UpdatedOn,
			},
		})
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repositories": formattedRepositories,
	})
}

type ByRepoID []bitbucketModel.Repository

func (a ByRepoID) Len() int           { return len(a) }
func (a ByRepoID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRepoID) Less(i, j int) bool { return a[i].Slug < a[j].Slug }
func (c BitbucketRepositoriesController) ListConnectedBitbucketRepositories(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	accounts := c.Params.Query.Get("uids")
	companyID := c.ViewArgs["companyID"].(string)
	saasMembers := []string{}
	if accounts != "" {
		errMarshal := json.Unmarshal([]byte(accounts), &saasMembers)
		if errMarshal != nil {
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 422,
				Message:        "BitbucketGroupMembers Error: Unmarshal Members",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}
	}

	if utils.FindEmptyStringElement([]string{groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "ListConnetedBitbucketRepositories Error: Missing required parameter - GroupID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	//* get connected repos
	connectedItems, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_REPOSITORIES)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketRepositories Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	reposList, bErr := bitbucketOperations.ListBitbucketRepositoriesHelper(
		bitbucketCredentials.Workspace,
		bitbucketCredentials.Token,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}
	sort.Sort(ByRepoID(*reposList))

	formattedRepositories := []bitbucketModel.ConnectedRepositoriesFormatFE{}
	var connectedRepos []bitbucketModel.Repository
	for _, connectedItem := range connectedItems {
		index := sort.Search(len(*reposList), func(i int) bool {
			return (*reposList)[i].Slug >= connectedItem
		})
		if index < len((*reposList)) && (*reposList)[index].Slug == connectedItem {
			connectedRepos = append(connectedRepos, (*reposList)[index])
			formattedRepositories = append(formattedRepositories, bitbucketModel.ConnectedRepositoriesFormatFE{})
		}
	}

	//call group members
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

	if len(connectedRepos) != 0 {

		bitbucketRepositoryChannel := make(chan bitbucketModel.ConnectedRepositoriesFormatFE)

		for _, cRepo := range connectedRepos {
			listRepositoryPermissions := []bitbucketModel.RepositoryPermissionsFE{}
			page := 1

			for {
				repositoryUserPermissions, err := bitbucketOperations.ListRepositoryUserPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cRepo.Slug)
				if err != nil {
					c.Response.Status = 401
					return c.RenderJSON(err)
				}
				for _, userPermission := range repositoryUserPermissions.Values {
					listRepositoryPermissions = append(listRepositoryPermissions, bitbucketModel.RepositoryPermissionsFE{
						Type:       userPermission.User.Type,
						Slug:       userPermission.User.AccountID,
						Name:       userPermission.User.Name,
						Permission: userPermission.Permission,
						Status:     "NOT_SYNCED",
						Email:      "",
						SaaSID:     "",
					})
				}
				if len(repositoryUserPermissions.Values) <= 0 {
					page = 1
					break
				}
				page++
			}

			for {
				repositoryGroupPermissions, err := bitbucketOperations.ListRepositoryGroupPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, cRepo.Slug)
				if err != nil {
					c.Response.Status = 401
					return c.RenderJSON(err)
				}

				for _, groupPermission := range repositoryGroupPermissions.Values {
					listRepositoryPermissions = append(listRepositoryPermissions, bitbucketModel.RepositoryPermissionsFE{
						Type:       groupPermission.Group.Type,
						Slug:       groupPermission.Group.Slug,
						Name:       groupPermission.Group.Name,
						Permission: groupPermission.Permission,
						Status:     "NOT_SYNCED",
						SaaSID:     "",
						SaaSName:   "",
					})
				}

				if len(repositoryGroupPermissions.Values) <= 0 {
					break
				}
				page++
			}

			go ListConnectedRepositoryWithRoutines(bitbucketCredentials, saasMembers, cRepo, bitbucketRepositoryChannel, memberUsers, memberGroups, listRepositoryPermissions, emails)
		}
		for i := 0; i < len(connectedRepos); i++ {
			formattedRepositories[i] = <-bitbucketRepositoryChannel
		}

	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repositories": formattedRepositories,
	})
}
func (c BitbucketRepositoriesController) GetConnectedBitbucketRepositoryMembers(repoSlug string) revel.Result {
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
				Message:        "GetConnectedBitbucketRepositories Error: Unmarshal Members",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}
	}

	getRepository, opsErr := bitbucketOperations.GetRepository(bitbucketCredentials.Token, bitbucketCredentials.Workspace, repoSlug)
	if opsErr != nil {
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "GetConnectedBitbucketRepositories Error: Repository not found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	formattedRepositories := []bitbucketModel.ConnectedRepositoriesFormatFE{}
	formattedRepository := bitbucketModel.ConnectedRepositoriesFormatFE{}
	var connectedRepos []bitbucketModel.Repository
	formattedRepositories = append(formattedRepositories, bitbucketModel.ConnectedRepositoriesFormatFE{})
	connectedRepos = append(connectedRepos, *getRepository)

	bitbucketRepositoryChannel := make(chan bitbucketModel.ConnectedRepositoriesFormatFE)

	pendingInvitations, error := bitbucketOperations.GetPendingInvitations(bitbucketCredentials.Token, bitbucketCredentials.Workspace)
	if error != nil {
		return c.RenderJSON(error)
	}

	emails := []string{}
	for _, invitation := range pendingInvitations {
		emails = append(emails, invitation.Email)
	}

	//call group members
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

	listRepositoryPermissions := []bitbucketModel.RepositoryPermissionsFE{}
	page := 1

	for {
		repositoryUserPermissions, err := bitbucketOperations.ListRepositoryUserPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, repoSlug)
		if err != nil {
			c.Response.Status = 401
			return c.RenderJSON(err)
		}

		for _, userPermission := range repositoryUserPermissions.Values {
			listRepositoryPermissions = append(listRepositoryPermissions, bitbucketModel.RepositoryPermissionsFE{
				Type:       userPermission.User.Type,
				Slug:       userPermission.User.AccountID,
				Name:       userPermission.User.Name,
				Permission: userPermission.Permission,
				Status:     "NOT_SYNCED",
				Email:      "",
				SaaSID:     "",
			})
		}

		if len(repositoryUserPermissions.Values) <= 0 {
			page = 1
			break
		}
		page++
	}

	for {
		repositoryGroupPermissions, err := bitbucketOperations.ListRepositoryGroupPermissions(bitbucketCredentials.Token, strconv.Itoa(page), bitbucketCredentials.Workspace, repoSlug)
		if err != nil {
			c.Response.Status = 401
			return c.RenderJSON(err)
		}

		for _, groupPermission := range repositoryGroupPermissions.Values {
			listRepositoryPermissions = append(listRepositoryPermissions, bitbucketModel.RepositoryPermissionsFE{
				Type:       groupPermission.Group.Type,
				Slug:       groupPermission.Group.Slug,
				Name:       groupPermission.Group.Name,
				Permission: groupPermission.Permission,
				Status:     "NOT_SYNCED",
				SaaSID:     "",
				SaaSName:   "",
			})
		}

		if len(repositoryGroupPermissions.Values) <= 0 {
			break
		}
		page++
	}

	// for _, cRepo := range connectedRepos {
	go ListConnectedRepositoryWithRoutines(bitbucketCredentials, saasMembers, *getRepository, bitbucketRepositoryChannel, memberUsers, memberGroups, listRepositoryPermissions, emails)
	// }
	// for i := 0; i < len(connectedRepos); i++ {
	formattedRepository = <-bitbucketRepositoryChannel
	// }

	c.Response.Status = 200
	return c.RenderJSON(formattedRepository)
}

func ListConnectedRepositoryWithRoutines(bitbucketCredentials bitbucketModel.BitbucketCredentials, saasMembers []string, cRepo bitbucketModel.Repository, repoChannel chan<- bitbucketModel.ConnectedRepositoriesFormatFE, users []models.GroupMember, groups []models.GroupMember, permissionsList []bitbucketModel.RepositoryPermissionsFE, pendingInvitations []string) {
	repositoryPermissions, nonExistingPermissionsCount, bErr := GetBitbucketRepositoryGroupPermissionsHelper(GetBBRGPInput{
		Token:              bitbucketCredentials.Token,
		Workspace:          bitbucketCredentials.Workspace,
		RepoSlug:           cRepo.Slug,
		GroupMembers:       saasMembers,
		Users:              users,
		Groups:             groups,
		PermissionList:     permissionsList,
		PendingInvitations: pendingInvitations,
	})
	if bErr != nil {

	}

	formattedRepo := bitbucketModel.ConnectedRepositoriesFormatFE{
		ID:                          cRepo.UUID,
		Label:                       cRepo.Name,
		Value:                       cRepo.Slug,
		Name:                        cRepo.Name,
		Slug:                        cRepo.Slug,
		Description:                 cRepo.Description,
		IsPrivate:                   cRepo.IsPrivate,
		CreatedOn:                   cRepo.CreatedOn,
		LastUpdated:                 cRepo.LastUpdated,
		RepositoryPermissions:       permissionsList,
		SaaSMembers:                 *repositoryPermissions,
		Links:                       cRepo.Links,
		NonExistingPermissionsCount: nonExistingPermissionsCount,
		Project: bitbucketModel.ProjectsFormatFE{
			ID:          cRepo.Project.UUID,
			Label:       cRepo.Project.Name,
			Value:       cRepo.Project.Key,
			Key:         cRepo.Project.Key,
			Name:        cRepo.Project.Name,
			Description: cRepo.Project.Description,
			IsPrivate:   cRepo.Project.IsPrivate,
			CreatedOn:   cRepo.Project.CreatedOn,
			UpdatedOn:   cRepo.Project.UpdatedOn,
		},
	}

	repoChannel <- formattedRepo
}
func (c BitbucketRepositoriesController) GetBitbucketRepository(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	repoSlug := c.Params.Query.Get("repoSlug")

	if utils.FindEmptyStringElement([]string{repoSlug, groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "GetBitbucketRepository Error: Missing required parameter - RepoSlug/GroupID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	connectedGroups, err := ops.GetConnectedItemBySlug(groupID, constants.INTEG_SLUG_BITBUCKET_GROUPS)
	if err != nil {
		c.Response.Status = 500
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 500,
			Message:        "GetBitbucketGroups Error: GetConnectedItemBySlug : " + err.Error(),
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_500),
		})
	}

	//* test data
	// connectedGroups := []string {
	// 	"developers",
	// 	"sample",
	// }

	trimmedRepoSlug := strings.TrimSpace(strings.ToLower(repoSlug))

	//* get repo
	repository, bErr := bitbucketOperations.GetRepository(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		trimmedRepoSlug,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	repositoryPermissions, nonExistingPermissionsCount, bErr := GetBitbucketRepositoryGroupPermissionsHelper(GetBBRGPInput{
		Token:           bitbucketCredentials.Token,
		Workspace:       bitbucketCredentials.Workspace,
		RepoSlug:        repoSlug,
		ConnectedGroups: connectedGroups,
	})
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	formattedRepos := bitbucketModel.ConnectedRepositoriesFormatFE{
		ID:                          repository.UUID,
		Value:                       repository.Slug,
		Label:                       repository.Name,
		Name:                        repository.Name,
		CreatedOn:                   repository.CreatedOn,
		LastUpdated:                 repository.LastUpdated,
		IsPrivate:                   repository.IsPrivate,
		Slug:                        repository.Slug,
		Description:                 repository.Description,
		RepositoryPermissions:       *repositoryPermissions,
		NonExistingPermissionsCount: nonExistingPermissionsCount,
		Project: bitbucketModel.ProjectsFormatFE{
			ID:          repository.Project.UUID,
			Label:       repository.Project.Name,
			Value:       repository.Project.Key,
			Key:         repository.Project.Key,
			Name:        repository.Project.Name,
			Description: repository.Project.Description,
			IsPrivate:   repository.Project.IsPrivate,
			CreatedOn:   repository.Project.CreatedOn,
			UpdatedOn:   repository.Project.UpdatedOn,
		},
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repository": formattedRepos,
	})
}

type CreateBitbucketRepositoryInput struct {
	Name        string   `json:"name"`
	RepoSlug    string   `json:"slug"`
	Description string   `json:"description"`
	IsPrivate   bool     `json:"is_private"`
	ProjectKey  string   `json:"project_key"`
	Members     []string `json:"uuids"`
	Users       []string `json:"group_users"`
	Groups      []string `json:"group_sub_groups"`
}

func (c BitbucketRepositoriesController) CreateBitbucketRepository(groupID string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	var input CreateBitbucketRepositoryInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Name, input.ProjectKey, input.RepoSlug, groupID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "CreateBitbucketRepository Error: Missing required parameter - Name/RepoSlug/Key/GroupID.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedRepositoryName := strings.TrimSpace(input.Name)
	trimmedRepositorySlug := strings.TrimSpace(input.RepoSlug)
	trimmedRepositoryDescription := strings.TrimSpace(input.Description)
	trimmedProjectKey := strings.TrimSpace(strings.ToUpper(input.ProjectKey))

	postBody, marshalError := json.Marshal(map[string]interface{}{
		"name":        trimmedRepositoryName,
		"description": trimmedRepositoryDescription,
		"is_private":  input.IsPrivate,
		"project": map[string]interface{}{
			"key": trimmedProjectKey,
		},
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "CreateBitbucketRepository Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	createdRepository, bErr := bitbucketOperations.CreateRepository(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		trimmedRepositorySlug,
		postBody,
	)
	if bErr != nil {
		c.Response.Status = bErr.HTTPStatusCode
		return c.RenderJSON(bErr)
	}

	var addedUserPermissions []*bitbucketModel.RepositoryUserPermission
	var addedUserPermissionsError []*bitbucketModel.RepositoryUserPermission
	var addedGroupPermissions []*bitbucketModel.RepositoryGroupPermission
	var addedGroupPermissionsError []*bitbucketModel.RepositoryGroupPermission

	if len(input.Users) != 0 {
		postBodyRGP, marshalError := json.Marshal(map[string]interface{}{
			"permission": "read", // read is the default permission
		})
		if marshalError != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        "UpdateRepositoryGroupPermissions Error: Error on marshalling data.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}

		for _, id := range input.Users {
			updateRUP, bErr := bitbucketOperations.UpdateRepositoryUserPermission(
				bitbucketCredentials.Token,
				bitbucketCredentials.Workspace,
				createdRepository.Slug,
				id,
				postBodyRGP,
			)
			if bErr != nil {
				addedUserPermissionsError = append(addedUserPermissionsError, &bitbucketModel.RepositoryUserPermission{
					User: bitbucketModel.RepositoryUserPermissionInfo{
						AccountID: id,
					},
					Permission: "read",
				})
			}

			bitbucketUserTemp := bitbucketModel.Users{}

			// if errCache := cache.Get("bitbucketUserInfo#"+id, &bitbucketUserTemp); errCache != nil {

			userInfo, err := bitbucketOperations.GetBitbucketUser(bitbucketCredentials.Token, id)
			if err == nil {
				if userInfo.AccountStatus == "active" {
					bitbucketUserTemp = bitbucketModel.Users{
						DisplayName:   userInfo.DisplayName,
						UUID:          userInfo.UUID,
						Type:          "user",
						AccountID:     userInfo.AccountID,
						Nickname:      userInfo.Nickname,
						Links:         userInfo.Links,
						AccountStatus: userInfo.AccountStatus,
					}
					// go cache.Set("bitbucketUserInfo#"+id, bitbucketUserTemp, 10*time.Minute)

				}
			}

			// }
			if updateRUP != nil {
				repositoryPermission := bitbucketModel.RepositoryPermissionsFE{
					Type:       updateRUP.User.Type,
					Name:       updateRUP.User.Name,
					Slug:       id,
					Permission: "read",
					IsExisting: true,
					Status:     utils.IfThenElse(bitbucketUserTemp.AccountStatus == "active", "ACTIVE", "INACTIVE").(string),
				}
				go cache.Set("repositoryPermission#"+createdRepository.Slug+"#"+id, repositoryPermission, 10*time.Minute)
				addedUserPermissions = append(addedUserPermissions, updateRUP)
			} else {
				userError := &bitbucketModel.RepositoryUserPermission{
					User: bitbucketModel.RepositoryUserPermissionInfo{
						AccountID: id,
						Type:      "",
					},
					Permission: "",
				}
				addedUserPermissionsError = append(addedUserPermissionsError, userError)
			}
		}
	}

	if len(input.Groups) != 0 {
		postBodyRGP, marshalError := json.Marshal(map[string]interface{}{
			"permission": "read", // read is the default permission
		})
		if marshalError != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				HTTPStatusCode: 400,
				Message:        "UpdateRepositoryGroupPermissions Error: Error on marshalling data.",
				Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			})
		}

		for _, groupID := range input.Groups {
			updateRUP, bErr := bitbucketOperations.UpdateRepositoryGroupPermission(
				bitbucketCredentials.Token,
				bitbucketCredentials.Workspace,
				createdRepository.Slug,
				groupID,
				postBodyRGP,
			)
			if bErr != nil {
				addedGroupPermissionsError = append(addedGroupPermissionsError, &bitbucketModel.RepositoryGroupPermission{
					Group: bitbucketModel.RepositoryGroupPermissionInfo{
						Slug: groupID,
					},
					Permission: "read",
				})
			}

			if updateRUP != nil {
				addedGroupPermissions = append(addedGroupPermissions, updateRUP)
			} else {
				groupError := &bitbucketModel.RepositoryGroupPermission{
					Group: bitbucketModel.RepositoryGroupPermissionInfo{
						Slug: groupID,
						Type: "",
					},
					Permission: "",
				}
				addedGroupPermissionsError = append(addedGroupPermissionsError, groupError)
			}

		}
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repository":                     createdRepository,
		"user_permissions_added":         addedUserPermissions,
		"user_permissions_added_errors":  addedUserPermissionsError,
		"group_permissions_added":        addedGroupPermissions,
		"group_permissions_added_errors": addedGroupPermissionsError,
	})
}

type UpdateRepositoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	ProjectKey  string `json:"project_key"`
}

func (c BitbucketRepositoriesController) UpdateBitbucketRepository(repoSlug string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	var input UpdateRepositoryInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Name, repoSlug, input.ProjectKey}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketRepository Error: Missing required parameter - Name/RepoSlug/ProjectKey.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedRepositoryName := strings.TrimSpace(input.Name)
	trimmedRepositorySlug := strings.TrimSpace(repoSlug)
	trimmedRepositoryDescription := strings.TrimSpace(input.Description)
	trimmedProjectKey := strings.TrimSpace(strings.ToUpper(input.ProjectKey))

	postBody, marshalError := json.Marshal(map[string]interface{}{
		"name":        trimmedRepositoryName,
		"description": trimmedRepositoryDescription,
		"is_private":  input.IsPrivate,
		"project": map[string]interface{}{
			"key": trimmedProjectKey,
		},
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "UpdateBitbucketRepository Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	updatedRepository, bErr := bitbucketOperations.UpdateRepository(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		trimmedRepositorySlug,
		postBody,
	)
	if bErr != nil {
		c.Response.Status = bErr.HTTPStatusCode
		return c.RenderJSON(bErr)
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repository": updatedRepository,
	})
}

type UpdateBitbucketRepositoryGroupPermissionInput struct {
	GroupSlug  string `json:"group_slug"`
	Permission string `json:"permission"`
}
type UpdateBitbucketRepositoryUserPermissionInput struct {
	Account    string `json:"id"`
	Permission string `json:"permission"`
}

func (c BitbucketRepositoriesController) UpdateBitbucketRepositoryGroupPermission(repoSlug string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input UpdateBitbucketRepositoryGroupPermissionInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.GroupSlug, input.Permission, repoSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketRepositoryGroupPermission Error: Missing required parameter - GroupSlug/Permission/RepoSlug.",
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
			Message:        "UpdateBitbucketRepositoryGroupPermission Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	updatedRGP, bErr := bitbucketOperations.UpdateRepositoryGroupPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		repoSlug,
		trimmedGroupSlug,
		postBodyPGP,
	)

	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repo_gp": updatedRGP,
	})
}

func (c BitbucketRepositoriesController) RemoveBitbucketRepositoryGroupPermission(repoSlug string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	groupSlug := c.Params.Query.Get("group_slug")

	if utils.FindEmptyStringElement([]string{groupSlug, repoSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "RemoveBitbucketRepositoryGroupPermissions Error: Missing required parameter - GroupSlug/RepoSlug.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedGroupSlug := strings.TrimSpace(groupSlug)

	_, bErr := bitbucketOperations.RemoveRepositoryGroupPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		repoSlug,
		trimmedGroupSlug,
	)
	if bErr != nil {
		// c.Response.Status = 401
		c.Response.Status = 400
		return c.RenderJSON(bErr)
	}

	c.Response.Status = 204
	return nil
}
func (c BitbucketRepositoriesController) UpdateBitbucketRepositoryUserPermission(repoSlug string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	var input UpdateBitbucketRepositoryUserPermissionInput
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Account, input.Permission, repoSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "UpdateBitbucketRepositoryUserPermission Error: Missing required parameter - Account/Permission/RepoSlug.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedID := strings.TrimSpace(input.Account)
	trimmedPermission := strings.TrimSpace(input.Permission)

	postBodyPGP, marshalError := json.Marshal(map[string]interface{}{
		"permission": trimmedPermission,
	})
	if marshalError != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 400,
			Message:        "UpdateBitbucketRepositoryUserPermission Error: Error on marshalling data.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	updatedRGP, bErr := bitbucketOperations.UpdateRepositoryUserPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		repoSlug,
		trimmedID,
		postBodyPGP,
	)

	if bErr != nil {
		return c.RenderJSON(bErr)
	}

	bitbucketUserTemp := bitbucketModel.Users{}

	// if errCache := cache.Get("bitbucketUserInfo#"+input.Account, &bitbucketUserTemp); errCache != nil {

	userInfo, err := bitbucketOperations.GetBitbucketUser(bitbucketCredentials.Token, input.Account)
	if err == nil {
		if userInfo.AccountStatus == "active" {
			bitbucketUserTemp = bitbucketModel.Users{
				DisplayName:   userInfo.DisplayName,
				UUID:          userInfo.UUID,
				Type:          "user",
				AccountID:     userInfo.AccountID,
				Nickname:      userInfo.Nickname,
				Links:         userInfo.Links,
				AccountStatus: userInfo.AccountStatus,
			}
			// go cache.Set("bitbucketUserInfo#"+input.Account, bitbucketUserTemp, 10*time.Minute)

		}
	}
	// }
	repositoryPermission := bitbucketModel.RepositoryPermissionsFE{
		Type:       updatedRGP.User.Type,
		Name:       updatedRGP.User.Name,
		Slug:       trimmedID,
		Permission: input.Permission,
		IsExisting: true,
		Status:     utils.IfThenElse(bitbucketUserTemp.AccountStatus == "active", "ACTIVE", "INACTIVE").(string),
	}
	go cache.Set("repositoryPermission#"+repoSlug+"#"+input.Account, repositoryPermission, 10*time.Minute)

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"repo_gp": repositoryPermission,
	})
}

func (c BitbucketRepositoriesController) RemoveBitbucketRepositoryUserPermission(repoSlug string) revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)
	account := c.Params.Query.Get("id")

	if utils.FindEmptyStringElement([]string{account, repoSlug}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "RemoveBitbucketRepositoryGroupPermissions Error: Missing required parameter - AccountID/RepoSlug.",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}

	trimmedAccount := strings.TrimSpace(account)

	_, bErr := bitbucketOperations.RemoveRepositoryUserPermission(
		bitbucketCredentials.Token,
		bitbucketCredentials.Workspace,
		repoSlug,
		trimmedAccount,
	)
	if bErr != nil {
		c.Response.Status = 401
		return c.RenderJSON(bErr)
	}
	go cache.Delete("repositoryPermission#" + repoSlug + "#" + account)

	c.Response.Status = 204
	return nil
}
func ListBitbucketRepositoryGroupPermissionsHelper(token, workspace, repoSlug string) (*[]bitbucketModel.RepositoryGroupPermission, *models.ErrorResponse) {
	listRepositoryGroupPermissions := []bitbucketModel.RepositoryGroupPermission{}

	page := 1
	for {
		repositoryGroupPermissions, err := bitbucketOperations.ListRepositoryGroupPermissions(token, strconv.Itoa(page), workspace, repoSlug)
		if err != nil {
			return nil, err
		}
		listRepositoryGroupPermissions = append(listRepositoryGroupPermissions, repositoryGroupPermissions.Values...)
		if len(repositoryGroupPermissions.Values) <= 0 {
			break
		}
		page++
	}
	return &listRepositoryGroupPermissions, nil
}
func ListBitbucketRepositoryUsersPermissionsHelper(token, workspace, repoSlug string) (*[]bitbucketModel.RepositoryUserPermission, *models.ErrorResponse) {
	listRepositoryUserPermissions := []bitbucketModel.RepositoryUserPermission{}
	page := 1
	for {
		repositoryUserPermissions, err := bitbucketOperations.ListRepositoryUserPermissions(token, strconv.Itoa(page), workspace, repoSlug)
		if err != nil {
			return nil, err
		}
		listRepositoryUserPermissions = append(listRepositoryUserPermissions, repositoryUserPermissions.Values...)
		if len(repositoryUserPermissions.Values) <= 0 {
			break
		}
		page++
	}
	return &listRepositoryUserPermissions, nil
}

type GetBBRGPInput struct {
	Token              string   `json:"token"`
	Workspace          string   `json:"workspace"`
	RepoSlug           string   `json:"repo_slug"`
	ConnectedGroups    []string `json:"connected_groups"`
	GroupMembers       []string `json:"group_members"`
	Users              []models.GroupMember
	Groups             []models.GroupMember
	PermissionList     []bitbucketModel.RepositoryPermissionsFE
	PendingInvitations []string
}
type ByRepoUserPermissionID []bitbucketModel.RepositoryUserPermission

func (a ByRepoUserPermissionID) Len() int           { return len(a) }
func (a ByRepoUserPermissionID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRepoUserPermissionID) Less(i, j int) bool { return a[i].User.AccountID < a[j].User.AccountID }
func GetBitbucketRepositoryGroupPermissionsHelper(input GetBBRGPInput) (*[]bitbucketModel.RepositoryPermissionsFE, int, *models.ErrorResponse) {
	// listFormattedRepositoryPermissions := []bitbucketModel.RepositoryPermissionsFE{}
	// // listRGU, bErr := ListBitbucketRepositoryUsersPermissionsHelper(
	// // 	input.Token,
	// // 	input.Workspace,
	// // 	input.RepoSlug,
	// // )
	// // if bErr != nil {
	// // 	return nil, 0, bErr
	// // }
	// // sort.Sort(ByRepoUserPermissionID(*listRGU))
	// nonExistingPermissionsCount := 0
	// if len(input.GroupMembers) != 0 {
	// 	userRepositoryPermission := make(chan bitbucketModel.RepositoryPermissionsFE)
	// 	for _, userAccount := range input.GroupMembers {
	// 		listFormattedRepositoryPermissions = append(listFormattedRepositoryPermissions, bitbucketModel.RepositoryPermissionsFE{})
	// 		go ListBitbucketRepositoryUserPermissionsHelperWithGoRoutines(userAccount, input.Token, input.Workspace, input.RepoSlug, userRepositoryPermission)
	// 	}

	// 	for i := 0; i < len(listFormattedRepositoryPermissions); i++ {
	// 		listFormattedRepositoryPermissions[i] = <-userRepositoryPermission
	// 		if !listFormattedRepositoryPermissions[i].IsExisting && listFormattedRepositoryPermissions[i].Status == "NOT_SYNCED" {
	// 			nonExistingPermissionsCount++
	// 		}
	// 	}
	// }

	var (
		listFormattedRepositoryPermissions []bitbucketModel.RepositoryPermissionsFE
		nonExistingPermissionsCount        int
		wg                                 sync.WaitGroup
		mu                                 sync.Mutex
	)

	if len(input.Users) != 0 {
		userPermissionChan := make(chan bitbucketModel.RepositoryPermissionsFE)
		errorChan := make(chan *models.ErrorResponse, len(input.Users))

		for _, user := range input.Users {
			wg.Add(1)
			go func(user models.GroupMember) {
				defer wg.Done()
				processRepositoryUserPermissions(user, input, nonExistingPermissionsCount, userPermissionChan, errorChan)
			}(user)
		}

		go func() {
			wg.Wait()
			close(userPermissionChan)
			close(errorChan)
		}()

		for perm := range userPermissionChan {
			mu.Lock()
			listFormattedRepositoryPermissions = append(listFormattedRepositoryPermissions, perm)
			mu.Unlock()
		}

	}

	if len(input.Groups) != 0 {
		groupList, bErr := bitbucketOperations.ListGroups(input.Workspace, input.Token)
		if bErr != nil {
			return nil, 0, bErr
		}

		groupPermissionsChan := make(chan bitbucketModel.RepositoryPermissionsFE)
		for _, group := range input.Groups {
			wg.Add(1)
			go func(group models.GroupMember) {
				defer wg.Done()
				processRepositoryGroupPermissions(group, input, groupList, nonExistingPermissionsCount, groupPermissionsChan)
			}(group)
		}

		go func() {
			wg.Wait()
			close(groupPermissionsChan)
		}()

		for perm := range groupPermissionsChan {
			mu.Lock()
			listFormattedRepositoryPermissions = append(listFormattedRepositoryPermissions, perm)
			mu.Unlock()
		}
	}

	wg.Wait()

	return &listFormattedRepositoryPermissions, nonExistingPermissionsCount, nil
}
func ListBitbucketRepositoryUserPermissionsHelperWithGoRoutines(userAccount, token, workspace, repoSlug string, userRepoPermissionChannel chan<- bitbucketModel.RepositoryPermissionsFE) {
	repositoryPermission := bitbucketModel.RepositoryPermissionsFE{}
	bitbucketUserTemp := bitbucketModel.Users{}

	userPermission, opsErr := bitbucketOperations.GetRepositoryUserPermissions(token, workspace, repoSlug, userAccount)
	if opsErr != nil {

	}
	// if errCache := cache.Get("bitbucketUserInfo#"+userAccount, &bitbucketUserTemp); errCache != nil {
	userInfo, err := bitbucketOperations.GetBitbucketUser(token, userAccount)
	if err == nil {
		bitbucketUserTemp = bitbucketModel.Users{
			DisplayName:   userInfo.DisplayName,
			UUID:          userInfo.UUID,
			Type:          "user",
			AccountID:     userInfo.AccountID,
			Nickname:      userInfo.Nickname,
			Links:         userInfo.Links,
			AccountStatus: userInfo.AccountStatus,
		}
		// go cache.Set("bitbucketUserInfo#"+userAccount, bitbucketUserTemp, 10*time.Minute)
	}
	// }
	if userPermission != nil && userPermission.Permission != "none" {

		repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
			Type:       userPermission.User.Type,
			Name:       userPermission.User.Name,
			Slug:       userPermission.User.AccountID,
			Permission: userPermission.Permission,
			IsExisting: true,
			Status:     utils.IfThenElse(bitbucketUserTemp.AccountStatus == "active", "ACTIVE", "DEACTIVATED").(string),
		}
	} else {
		if bitbucketUserTemp.AccountStatus == "active" {
			repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
				Name:       bitbucketUserTemp.DisplayName,
				Slug:       userAccount,
				Type:       "user",
				IsExisting: false,
				Status:     "NOT_SYNCED",
			}
		}
		if bitbucketUserTemp.AccountStatus == "inactive" {
			repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
				Name:       bitbucketUserTemp.DisplayName,
				Slug:       userAccount,
				Type:       "user",
				IsExisting: false,
				Status:     "DEACTIVATED",
			}
		}
	}
	// userRepoPermissionChannel <- repositoryPermission
	if userRepoPermissionChannel != nil {
		select {
		case userRepoPermissionChannel <- repositoryPermission:
			// Value successfully sent
		default:
			// Channel is full or closed, handle accordingly
		}
	} else {
		// Handle the case when the channel is nil
	}
}
func ListBitbucketRepositoryUsersPermissionsHelperWithGoRoutines(userAccount, token, workspace, repoSlug string, listRGU *[]bitbucketModel.RepositoryUserPermission, userRepoPermissionChannel chan<- bitbucketModel.RepositoryPermissionsFE, errorChan chan<- *models.ErrorResponse) {
	repositoryPermission := bitbucketModel.RepositoryPermissionsFE{}
	bitbucketUserTemp := bitbucketModel.Users{}

	isExistOnRepo := false
	isExistOnBitbucket := false
	// if errCache := cache.Get("bitbucketUserInfo#"+userAccount, &bitbucketUserTemp); errCache != nil {
	userInfo, err := bitbucketOperations.GetBitbucketUser(token, userAccount)
	if err == nil {
		bitbucketUserTemp = bitbucketModel.Users{
			DisplayName:   userInfo.DisplayName,
			UUID:          userInfo.UUID,
			Type:          "user",
			AccountID:     userInfo.AccountID,
			Nickname:      userInfo.Nickname,
			Links:         userInfo.Links,
			AccountStatus: userInfo.AccountStatus,
		}
		// go cache.Set("bitbucketUserInfo#"+userAccount, bitbucketUserTemp, 10*time.Minute)
	}
	// }

	var membership bitbucketModel.Membership
	// errCacheMembership := "bitbucketMembershipUserInfo#" + userAccount
	// if errCache := cache.Get(errCacheMembership, &membership); errCache != nil {

	membershipBitbucket, err := bitbucketOperations.GetBitbucketUserMembership(token, workspace, userAccount)
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
	if isExistOnBitbucket {

		index := sort.Search(len(*listRGU), func(i int) bool {
			return (*listRGU)[i].User.AccountID >= userAccount
		})
		if index < len((*listRGU)) && (*listRGU)[index].User.AccountID == userAccount {
			// if errCache := cache.Get("repositoryPermission#"+repoSlug+"#"+userAccount, &repositoryPermission); errCache != nil {

			foundUserRepoPermission := (*listRGU)[index]
			repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
				Type:       foundUserRepoPermission.User.Type,
				Name:       foundUserRepoPermission.User.Name,
				Slug:       foundUserRepoPermission.User.AccountID,
				Permission: foundUserRepoPermission.Permission,
				IsExisting: true,
				Status:     utils.IfThenElse(bitbucketUserTemp.AccountStatus == "active", "ACTIVE", "DEACTIVATED").(string),
			}
			// go cache.Set("repositoryPermission#"+repoSlug+"#"+userAccount, repositoryPermission, 10*time.Minute)
			isExistOnRepo = true
			// }
		}

		if repositoryPermission.Slug != "" {
			isExistOnRepo = true
		}
		if !isExistOnRepo {
			if bitbucketUserTemp.AccountStatus == "active" {
				repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
					Name:       bitbucketUserTemp.DisplayName,
					Slug:       userAccount,
					Type:       "user",
					IsExisting: false,
					Status:     "NOT_SYNCED",
				}
			}
			if bitbucketUserTemp.AccountStatus == "inactive" {
				repositoryPermission = bitbucketModel.RepositoryPermissionsFE{
					Name:       bitbucketUserTemp.DisplayName,
					Slug:       userAccount,
					Type:       "user",
					IsExisting: false,
					Status:     "DEACTIVATED",
				}
			}
		}
	}

	userRepoPermissionChannel <- repositoryPermission
}

func processRepositoryUserPermissions(user models.GroupMember, input GetBBRGPInput, nonExistingPermissionsCount int, userPermissionsChan chan<- bitbucketModel.RepositoryPermissionsFE, errorChan chan<- *models.ErrorResponse) {
	associatedAccount := user.MemberInformation.AssociatedAccounts[constants.INTEG_SLUG_BITBUCKET]
	accountStatus := "ACTIVE"
	if len(associatedAccount) == 0 {

		invited := false

		for _, str := range input.PendingInvitations {
			if str == user.MemberInformation.Email {
				invited = true
			}
		}

		accountStatus = "NOT_SYNCED"

		if invited {
			accountStatus = "PENDING"
		}

		userPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
		if permission.Slug == associatedAccount[0] {
			index = i
			break
		}
	}

	if index == -1 {
		index = len(input.PermissionList) + 1
	}

	if index < len(input.PermissionList) && input.PermissionList[index].Slug == associatedAccount[0] {

		if input.PermissionList[index].Permission != "none" {
			input.PermissionList[index].ConnectedToSaaS = true
			input.PermissionList[index].Status = bitbucketUserTemp.AccountStatus
			input.PermissionList[index].Email = user.MemberInformation.Email
			input.PermissionList[index].SaaSID = user.MemberID
			input.PermissionList[index].IsExisting = true
			userPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
			handleNonExistingUserPermissionsRepository(user, bitbucketUserTemp, associatedAccount[0], nonExistingPermissionsCount, userPermissionsChan)
		}
	} else {
		handleNonExistingUserPermissionsRepository(user, bitbucketUserTemp, associatedAccount[0], nonExistingPermissionsCount, userPermissionsChan)
	}

}

func handleNonExistingUserPermissionsRepository(user models.GroupMember, bitbucketUserTemp bitbucketModel.Users, associatedAccount string, nonExistingPermissionsCount int, userPermissionsChan chan<- bitbucketModel.RepositoryPermissionsFE) {
	if bitbucketUserTemp.AccountStatus == "ACTIVE" {
		userPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
			Name:       bitbucketUserTemp.DisplayName,
			Slug:       associatedAccount,
			Type:       "user",
			IsExisting: false,
			Status:     "NOT_SYNCED",
			Email:      user.MemberInformation.Email,
			SaaSID:     user.MemberID,
		}
	} else if bitbucketUserTemp.AccountStatus == "INACTIVE" {
		userPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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

func processRepositoryGroupPermissions(group models.GroupMember, input GetBBRGPInput, groupList *[]bitbucketModel.GroupsResponse, nonExistingPermissionsCount int, groupPermissionsChan chan<- bitbucketModel.RepositoryPermissionsFE) {

	groupIntegrationIndex := -1
	for i, integration := range group.GroupIntegrations {
		if integration.IntegrationSlug == "bitbucket" {
			groupIntegrationIndex = i
			break
		}
	}

	if groupIntegrationIndex == -1 {
		groupPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
			groupPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
			handleNonExistingGroupPermissionsRepository(group, groupList, groupAssociatedAccount[0], nonExistingPermissionsCount, groupPermissionsChan)
		}
	} else {
		handleNonExistingGroupPermissionsRepository(group, groupList, groupAssociatedAccount[0], nonExistingPermissionsCount, groupPermissionsChan)
	}
}

func handleNonExistingGroupPermissionsRepository(group models.GroupMember, groupList *[]bitbucketModel.GroupsResponse, groupAccount string, nonExistingPermissionsCount int, groupPermissionsChan chan<- bitbucketModel.RepositoryPermissionsFE) {

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

	if existingGroupIndex < len(*groupList) && (*groupList)[existingGroupIndex].Slug == groupAccount {
		groupPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
		groupPermissionsChan <- bitbucketModel.RepositoryPermissionsFE{
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
