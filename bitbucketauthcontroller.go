package controllers

import (
	"encoding/base64"
	"grooper/app/constants"
	bitbucketOperations "grooper/app/integrations/bitbucket"
	"grooper/app/models"
	"grooper/app/utils"
	"strconv"

	bitbucketModel "grooper/app/models/bitbucket"
	ops "grooper/app/operations"

	"github.com/revel/revel"
)

type BitbucketAuthController struct {
	*revel.Controller
}

func GetBitBucketCredentials(c *revel.Controller) revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	integrationID := c.Params.Query.Get("integration_id")

	if utils.FindEmptyStringElement([]string{companyID, integrationID}) {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "GetBitBucketCredentials Error: Missing required parameter - CompanyID/IntegrationID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	integration, err := ops.GetCompanyIntegration(companyID, integrationID)
	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "Bitbucket token not found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}
	workspaceFound, bitError := FindWorkspaceHelper(integration.BitbucketCredentials.Token, integration.BitbucketCredentials.Workspace)
	if bitError != nil {
		c.Response.Status = 400
		return c.RenderJSON(bitError)
	}
	if !workspaceFound {
		c.Response.Status = 404
		return c.RenderJSON(bitbucketModel.BitbucketWorkspaceErrorResponse{
			HTTPStatusCode: 404,
			Message:        "TestConnection: Workspace was not found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
			Username:       integration.BitbucketCredentials.Username,
			Workspace:      integration.BitbucketCredentials.Workspace,
		})
	}

	bitbucketCredentials := bitbucketModel.BitbucketCredentials{
		Username:  integration.BitbucketCredentials.Username,
		Password:  integration.BitbucketCredentials.Password,
		Workspace: integration.BitbucketCredentials.Workspace,
		Token:     integration.BitbucketCredentials.Token,
	}

	c.ViewArgs["bitbucketCredentials"] = bitbucketCredentials

	return nil
}

// Save App Password
func (c BitbucketAuthController) SaveBitbucketCredentials() revel.Result {
	var input bitbucketModel.BitbucketCredentials
	c.Params.BindJSON(&input)

	userID := c.ViewArgs["userID"].(string)
	companyID := c.ViewArgs["companyID"].(string)

	integrationID := c.Params.Query.Get("integration_id")
	if utils.FindEmptyStringElement([]string{companyID, userID, integrationID}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "SaveBitbucketCredentials Error: Missing required parameter - UserID/CompanyID/IntegrationID",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}

	auth := input.Username + ":" + input.Password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	token := "Basic " + encodedAuth

	input.Token = token

	//save credentials
	err := ops.ConnectCompanyIntegration(input, models.TokenExtra{}, "bitbucket", userID, companyID, integrationID)
	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "Error  while saving bitbucket credentials",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		})
	}
	bitbucketCredentials := bitbucketModel.BitbucketCredentials{
		Username:  input.Username,
		Password:  input.Password,
		Workspace: input.Workspace,
		Token:     input.Token,
	}

	c.ViewArgs["bitbucketCredentials"] = bitbucketCredentials

	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"workspace": input.Workspace,
		"username":  input.Username,
	})
}

// Save App Password
func (c BitbucketAuthController) ConnectBitbucket() revel.Result {
	var input bitbucketModel.BitbucketCredentials
	c.Params.BindJSON(&input)

	if utils.FindEmptyStringElement([]string{input.Username, input.Password}) {
		c.Response.Status = 422
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 422,
			Message:        "ConnectBitbucket Error: Missing required parameter - Username/Password",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_422),
		})
	}
	auth := input.Username + ":" + input.Password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	token := "Basic " + encodedAuth

	workspaces, err := ListWorkspacesHelper(token)
	if err != nil {
		c.Response.Status = 401
		return c.RenderJSON(err)
	}
	if len(*workspaces) == 0 {
		c.Response.Status = 404
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 404,
			Message:        "ConnectBitbucket Error: No Workspace Found",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_404),
		})
	}
	var finalWorkspace []bitbucketModel.Workspace

	for _, workspace := range *workspaces {
		_, err := bitbucketOperations.ListGroups(workspace.Slug, token)
		if err == nil {
			finalWorkspace = append(finalWorkspace, workspace)
		}
	}
	if len(finalWorkspace) == 0 {
		c.Response.Status = 403
		return c.RenderJSON(models.ErrorResponse{
			HTTPStatusCode: 403,
			Message:        "ConnectBitbucket Error: Insufficient permission to access this action",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_403),
		})
	}
	c.Response.Status = 200
	return c.RenderJSON(map[string]interface{}{
		"workspaces": finalWorkspace,
	})
}

// **HELPER**//
// List Workspaces Helper
func ListWorkspacesHelper(token string) (*[]bitbucketModel.Workspace, *models.ErrorResponse) {
	listWorkspaces := []bitbucketModel.Workspace{}

	page := 1
	for {
		workspaces, err := bitbucketOperations.ListWorkspace(token, strconv.Itoa(page))
		if err != nil {
			return nil, err
		}
		listWorkspaces = append(listWorkspaces, workspaces.Values...)
		if len(workspaces.Values) <= 0 {
			break
		}
		page++
	}
	return &listWorkspaces, nil
}

// Find workspace
func FindWorkspaceHelper(token, workspace string) (bool, *models.ErrorResponse) {
	workspaces, bitError := ListWorkspacesHelper(token)
	if bitError != nil {
		return false, &models.ErrorResponse{
			HTTPStatusCode: 401,
			Message:        "FindWorkspaceHelper Error: ListWorkspaces",
			Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
		}
	}
	var isFound bool
	isFound = false
	for _, item := range *workspaces {
		if workspace == item.Slug {
			isFound = true
			break
		}
	}
	return isFound, nil
}
