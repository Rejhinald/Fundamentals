package controllers

import (
	"encoding/json"
	"grooper/app/constants"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"strconv"

	bitbucketModel "grooper/app/models/bitbucket"

	"github.com/revel/revel"
)

type BitbucketUsersController struct {
	*revel.Controller
}

/**Test Connection **/
func (c BitbucketUsersController) TestConnection() revel.Result {
	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"username":  bitbucketCredentials.Username,
		"workspace": bitbucketCredentials.Workspace,
	})
}

/**Invite Users**/
func (c BitbucketUsersController) InviteUsers() revel.Result {
	companyId := c.ViewArgs["companyID"].(string)
	var data bitbucketModel.InviteUsersPayload
	c.Params.BindJSON(&data)

	bitbucketCredentials := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials)

	integrationID := c.Params.Query.Get("integration_id")
	if utils.FindEmptyStringElement([]string{integrationID, data.EmailAddress}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "Missing required parameters when saving bitbucket credentials",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}
	// postBody, _ := json.Marshal(map[string]interface{}{
	// 	"accountname": data.AccountName,
	// 	"permission":  data.Permission,
	// 	"email":       data.EmailAddress,
	// })

	postBody, _ := json.Marshal(map[string]interface{}{
		"group_slug": data.RepoSlug,
		"email":      data.EmailAddress,
	})
	// var userData bitbucketModel.InviteUserPayload

	// userData = bitbucketModel.InviteUserPayload{
	// 	Workspace: bitbucketCredentials.Workspace,
	// 	RepoSlug:  data.RepoSlug,
	// 	Token:     bitbucketCredentials.Token,
	// 	PostBody:  postBody,
	// }
	// inviteUsers, err := bitbucketOperations.InviteUsers(userData)
	err := bitbucketOperations.InviteUsersToGroup(bitbucketModel.InviteUserToGroupPayload{
		Workspace:   bitbucketCredentials.Workspace,
		RepoSlug:    data.RepoSlug,
		Token:       bitbucketCredentials.Token,
		PostBody:    postBody,
		AccountName: data.AccountName,
	})

	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(err)
	}

	saveAccountErr := ops.SaveUserIntegrationAccounts(ops.SaveUserIntegrationAccountsInput{
		CompanyID:   companyId,
		UserID:      data.UserID,
		Integration: constants.INTEG_SLUG_BITBUCKET,
		Account:     data.EmailAddress,
	}, c.Controller)

	if saveAccountErr != nil {
	}
	c.Response.Status = 200
	return c.RenderJSON(nil)
}

// **Current User**//
func (c BitbucketUsersController) GetCurrentUser() revel.Result {
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token

	userID := c.ViewArgs["userID"].(string)
	_ = userID
	var resp bitbucketModel.CurrentUser
	// if errCache := cache.Get("bitbucketCurrentUserInfo#"+userID, &resp); errCache != nil {
	me, err := bitbucketOperations.GetCurrentUser(bitbucketToken)
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(err)
	}
	resp = *me
	// go cache.Set("bitbucketCurrentUserInfo#"+userID, *me, 30*time.Minute)
	// }

	c.Response.Status = 200
	return c.RenderJSON(resp)
}
func (c BitbucketUsersController) GetBitbucketUser(userID string) revel.Result {
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	var resp bitbucketModel.CurrentUser
	var membership bitbucketModel.Membership
	// errCacheName := "bitbucketUserInfo#" + userID
	// errCacheMembership := "bitbucketMembershipUserInfo#" + userID
	// if errCache := cache.Get(errCacheName, &resp); errCache != nil {
	userInfo, err := bitbucketOperations.GetBitbucketUser(bitbucketToken, userID)
	if err != nil {
		c.Response.Status = 400

		return c.RenderJSON(err)
	}
	resp = *userInfo
	// 	go cache.Set(errCacheName, *userInfo, 30*time.Minute)
	// }
	// if errCache := cache.Get(errCacheMembership, &membership); errCache != nil {

	membershipBitbucket, err := bitbucketOperations.GetBitbucketUserMembership(bitbucketToken, bitbucketWorkspace, userID)
	if err != nil {
		c.Response.Status = 400

		return c.RenderJSON(err)
	}
	membership = *membershipBitbucket
	// go cache.Set(errCacheMembership, *membershipBitbucket, 30*time.Minute)
	// }
	if membership.User.AccountID != "" {
		if membership.Workspace.Slug != bitbucketWorkspace {
			resp.AccountStatus = "INACTIVE"
		}
	}
	c.Response.Status = 200
	return c.RenderJSON(resp)
}
func (c BitbucketUsersController) ListWorkspaceUsers() revel.Result {
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace
	bitbucketListUsers := []bitbucketModel.Users{}
	resp, err := ListWorkspaceUsersHelper(bitbucketWorkspace, bitbucketToken)
	if err != nil {
		return c.RenderJSON(err)
	}
	for _, bitbucketUser := range *resp {
		bitbucketUserTemp := bitbucketModel.Users{}

		// if errCache := cache.Get("bitbucketWorkspaceUserInfo#"+bitbucketUser.AccountID, &bitbucketUserTemp); errCache != nil {
		userInfo, err := bitbucketOperations.GetBitbucketUser(bitbucketToken, bitbucketUser.UUID)
		if err == nil {
			if userInfo.AccountStatus == "active" {
				bitbucketUserTemp = bitbucketModel.Users{
					DisplayName:   userInfo.DisplayName,
					UUID:          userInfo.UUID,
					Type:          bitbucketUser.Type,
					AccountID:     userInfo.AccountID,
					Nickname:      userInfo.Nickname,
					Links:         userInfo.Links,
					AccountStatus: userInfo.AccountStatus,
				}
				// go cache.Set("bitbucketWorkspaceUserInfo#"+bitbucketUser.AccountID, bitbucketUserTemp, 30*time.Minute)

			}
		}
		// }
		if bitbucketUserTemp.UUID != "" {
			bitbucketListUsers = append(bitbucketListUsers, bitbucketUserTemp)
		}
	}
	return c.RenderJSON(bitbucketListUsers)
}

func (c BitbucketUsersController) ListPendingInvitations() revel.Result {
	bitbucketToken := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Token
	bitbucketWorkspace := c.ViewArgs["bitbucketCredentials"].(bitbucketModel.BitbucketCredentials).Workspace

	pendingInvitations, err := bitbucketOperations.GetPendingInvitations(bitbucketToken, bitbucketWorkspace)
	if err != nil {
		return c.RenderJSON(err)
	}

	emails := []string{}
	for _, invitation := range pendingInvitations {
		emails = append(emails, invitation.Email)
	}

	return c.RenderJSON(emails)
}

// **HELPER**//
// List Workspaces Helper
func ListWorkspaceUsersHelper(workspace, token string) (*[]bitbucketModel.Users, *models.ErrorResponse) {
	var data bitbucketModel.BitbucketDataPayload

	listWorkspaces := []bitbucketModel.Users{}

	page := 1

	data = bitbucketModel.BitbucketDataPayload{
		Workspace: workspace,
		Token:     token,
	}

	for {
		data.Page = strconv.Itoa(page)
		workspaces, err := bitbucketOperations.ListWorkspaceUsers(data)
		if err != nil {
			return nil, err
		}
		for _, item := range workspaces.Values {
			listWorkspaces = append(listWorkspaces, item.User)
		}
		if len(workspaces.Values) <= 0 {
			break
		}
		page++
	}
	return &listWorkspaces, nil
}
