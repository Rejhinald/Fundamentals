package controllers

import (
	"errors"
	"grooper/app"
	"grooper/app/constants"
	slackoperations "grooper/app/integrations/slack"
	"grooper/app/mail"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	"github.com/revel/revel/cache"
)

type RequestController struct {
	*revel.Controller
}

/*
****************
AcceptRequest()
- Accept request depending on type (group, role, integration).
****************
*/
func (c RequestController) AcceptRequest() revel.Result {
	var userIDs []string
	var requestType string
	var notificationID string
	companyID := c.ViewArgs["companyID"].(string)
	c.Params.Bind(&userIDs, "user_id")
	c.Params.Bind(&notificationID, "notification_id")
	c.Params.Bind(&requestType, "requestType")
	data := make(map[string]interface{})

	company, opsError := ops.GetCompanyByID(companyID)
	_, _ = company, opsError
	if opsError != nil {
		return c.RenderJSON(opsError)
	}

	companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
	_, _ = companyAdmins, err
	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to retrieve Company Admins",
		})
	}

	for _, userID := range userIDs {
		user, opsErr := ops.GetUserByIDNew(userID)
		_, _ = user, opsErr
		if opsErr != nil {
			return c.RenderJSON(opsErr)
		}

		requesterInfo, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    userID,
			CompanyID: companyID,
		}, c.Controller)
		if err != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "Unable to retrieve Company user",
			})
		}
		_, _ = requesterInfo, err
		notificationContent := models.NotificationContentType{
			RequesterUserID: requesterInfo.UserID,
			ActiveCompany:   companyID,
			IsAccepted:      "ACCEPTED",
		}
		if groupID := c.Params.Get("group_id"); groupID != "" {
			groupID := c.Params.Form.Get("group_id")
			memberType := c.Params.Form.Get("member_type")
			var memberRole, typeGroup, prefix string
			memberRole = constants.MEMBER_TYPE_USER
			typeGroup = constants.ENTITY_TYPE_GROUP_MEMBER
			prefix = constants.PREFIX_USER
			var recipients []mail.Recipient
			var members []models.GroupMember
			var inputRequest []*dynamodb.WriteRequest
			var membersInput *dynamodb.BatchWriteItemInput
			requestUserId := c.ViewArgs["userID"].(string)
			checked := ops.CheckPermissions(constants.ADD_GROUP_MEMBER, requestUserId, companyID)
			if !checked {
				data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
				return c.RenderJSON(data)
			}
			group, err := GetGroupByID(groupID)
			if err != nil {
				data["error"] = "Group not exists."
				data["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(data)
			}
			members = append(members, models.GroupMember{
				PK:         utils.AppendPrefix(constants.PREFIX_GROUP, groupID),
				SK:         utils.AppendPrefix(prefix, userID),
				CompanyID:  c.ViewArgs["companyID"].(string),
				GroupID:    groupID,
				MemberID:   userID,
				Status:     constants.ITEM_STATUS_ACTIVE,
				MemberType: memberType,
				MemberRole: memberRole,
				CreatedAt:  utils.GetCurrentTimestamp(),
				UpdatedAt:  utils.GetCurrentTimestamp(),
				Type:       typeGroup,
			})
			recipients = append(recipients, mail.Recipient{
				Name:       user.FirstName + " " + user.LastName,
				Email:      user.Email,
				GroupName:  group.GroupName,
				ActionType: "added",
			})
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
			jobs.Now(mail.SendEmail{
				Subject:    "You have been added to a group",
				Recipients: recipients,
				Template:   "notify_group_member.html",
			})
			membersInput = &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{
					app.TABLE_NAME: inputRequest,
				},
			}
			batchRes, err := app.SVC.BatchWriteItem(membersInput)
			_ = batchRes

			//ERROR AT INSERTING
			if err != nil {
				data["message"] = "Got error in put item (MEMBERS)"
				data["error"] = err.Error()
				data["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(data)
			}
			notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to join " + group.GroupName + " has been accepted."
			notificationContent.GroupID = groupID
			////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		} else if roleIDs := c.Params.Values["role_id[]"]; len(roleIDs) > 0 {
			c.Params.Bind(&roleIDs, "role_id")
			var rolesRequested []string
			for _, roleID := range roleIDs {

				role, opsErr := ops.GetRoleByID(roleID, companyID)
				_, _ = role, opsErr
				if opsErr != nil {
					return c.RenderJSON(opsErr)
				}
				rolesRequested = append(rolesRequested, role.RoleName)
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
				_, err = app.SVC.PutItem(input)
				if err != nil {
					data["error"] = "Cannot assign role due to server error"
					data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
					return c.RenderJSON(data)
				}
				if len(rolesRequested) == 1 {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to take on the role of " + rolesRequested[0] + " has been accepted."
				} else {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to take on the following roles: " + strings.Join(rolesRequested, ", ") + " has been accepted."
				}
				notificationContent.RolesRequested = roleIDs
			}
			/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		} else if integrationIDs := c.Params.Values["integration_id[]"]; len(integrationIDs) > 0 {
			c.Params.Bind(&integrationIDs, "integration_id")
			requestUserId := c.ViewArgs["userID"].(string)
			//authToken := c.Params.Form.Get("authToken")
			var requestAction string
			if requestType == constants.REQUEST_CONNECT_INTEGRATION {
				requestAction = "connect"
			} else {
				requestAction = "disconnect"
			}
			var integrationsRequested []string
			for _, integ := range integrationIDs {
				integInfo, err := ops.GetIntegrationByID(integ)
				if err != nil {
					c.Response.Status = 400
					return c.RenderJSON(models.ErrorResponse{
						Code:    "400",
						Message: "Unable to retrieve Integrations",
					})
				}
				integrationsRequested = append(integrationsRequested, integInfo.IntegrationName)
				if requestType == constants.REQUEST_CONNECT_INTEGRATION {

				} else {
					checked := ops.CheckPermissions(constants.DISCONNECT_INTEGRATION, requestUserId, companyID)
					if !checked {
						data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_401)
						return c.RenderJSON(data)
					}
					res, getErr := app.SVC.GetItem(&dynamodb.GetItemInput{
						TableName: aws.String(app.TABLE_NAME),
						Key: map[string]*dynamodb.AttributeValue{
							"PK": {
								S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
							},
							"SK": {
								S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integInfo.IntegrationID)),
							},
						},
					})
					if getErr != nil {
						data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
						return c.RenderJSON(data)
					}

					if res.Item == nil {
						data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
						return c.RenderJSON(data)
					}
					integrationConnected, operationErr := ops.GetIntegrationByID(integInfo.IntegrationID)
					if operationErr != nil {
						data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_404)
						return c.RenderJSON(c.Result)
					}
					integrationSlug := integrationConnected.IntegrationSlug
					if integrationConnected.IntegrationSlug == constants.INTEG_SLUG_SLACK {
						integrationInfo, opssError := ops.GetCompanyIntegration(companyID, integrationConnected.IntegrationID)
						if opssError != nil || integrationInfo.IntegrationToken == nil || integrationInfo.IntegrationToken.AccessToken == "" {
							c.Response.Status = 401
							return c.RenderJSON(models.ErrorResponse{
								HTTPStatusCode: 401,
								Message:        "GetSlackToken Error: Slack token not found",
								Status:         utils.GetHTTPStatus(constants.HTTP_STATUS_401),
							})
						}

						_, slackErr := slackoperations.UninstallAppFromSlackWorkspace("Bearer " + integrationInfo.IntegrationToken.AccessToken)
						if slackErr != nil {
							data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
							return c.RenderJSON(c.Result)
						}
					}
					input := &dynamodb.DeleteItemInput{
						Key: map[string]*dynamodb.AttributeValue{
							"PK": {
								S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
							},
							"SK": {
								S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationConnected.IntegrationID)),
							},
						},
						TableName: aws.String(app.TABLE_NAME),
					}
					_, err = app.SVC.DeleteItem(input)
					if err != nil {
						data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
						return c.RenderJSON(data)
					}
					subIntegrations, err := GetSubIntegrations(integrationConnected.IntegrationID)
					if err == nil {
						var connectedGroups []models.Integration
						for _, sub := range subIntegrations {
							cg := GetGroupsIntegrationConnection(sub.IntegrationID, companyID)
							connectedGroups = append(connectedGroups, cg...)
						}
						emptyCg := GetGroupsIntegrationEmptySubConnection(integrationConnected.IntegrationID, companyID)
						connectedGroups = append(connectedGroups, emptyCg...)

						if len(connectedGroups) != 0 {
							batchLimit := 25
							var batches [][]*dynamodb.WriteRequest
							var currentBatch []*dynamodb.WriteRequest

							if integrationSlug == constants.INTEG_SLUG_ORACLE {

								// Remove Connected Oracle Autonomous DB - OAuth
								for i, g := range connectedGroups {

									connectedDBs, err := ops.GetConnectedItemBySlug(g.GroupID, constants.INTEG_SLUG_ORACLE_AUTONOMOUS_DB)
									if err != nil {
										data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
										return c.RenderJSON(err.Error())
									}

									for _, dbID := range connectedDBs {
										currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
											Key: map[string]*dynamodb.AttributeValue{
												"PK": {
													S: aws.String(utils.AppendPrefix(constants.PREFIX_COMPANY, companyID)),
												},
												"SK": {
													S: aws.String(utils.AppendPrefix(constants.PREFIX_OAUTH, dbID)),
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

									_, err = ops.BatchWriteItemHandler(batches)
									if err != nil {
										data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
										return c.RenderJSON(err.Error())
									}

								}
							}

							// remove group and integration connection
							for i, g := range connectedGroups {
								currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
									Key: map[string]*dynamodb.AttributeValue{
										"PK": {
											S: aws.String(utils.AppendPrefix(constants.PREFIX_GROUP, g.GroupID)),
										},
										"SK": {
											S: aws.String(utils.AppendPrefix(constants.PREFIX_INTEGRATION, integrationConnected.IntegrationID)),
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

							_, err = ops.BatchWriteItemHandler(batches)
							if err != nil {
								data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
								return c.RenderJSON(err.Error())
							}

							// remove group and sub integration connection
							for i, g := range connectedGroups {
								currentBatch = append(currentBatch, &dynamodb.WriteRequest{DeleteRequest: &dynamodb.DeleteRequest{
									Key: map[string]*dynamodb.AttributeValue{
										"PK": {
											S: aws.String(g.PK),
										},
										"SK": {
											S: aws.String(g.SK),
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
								data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
								return c.RenderJSON(err.Error())
							}

						}
					}
				}
				if len(integrationsRequested) == 1 {
					if requestType == constants.REQUEST_CONNECT_INTEGRATION {
						notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " " + integrationsRequested[0] + " has been checked and is in progress for connection."
					} else {
						notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " " + integrationsRequested[0] + " has been accepted."
					}
				} else {
					if requestType == constants.REQUEST_CONNECT_INTEGRATION {
						notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " the following integrations: " + strings.Join(integrationsRequested, ", ") + " has been checked and is in progress for connection."
					} else {
						notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " the following integrations: " + strings.Join(integrationsRequested, ", ") + " has been accepted."
					}

				}
				notificationContent.RequestedIntegrations = integrationIDs
			}
		} else if requestType == "REQUEST_TO_CREATE_ACCOUNT" {
			originalNotification, opsErr := ops.GetNotificationByID(notificationID)
			if opsErr != nil {
				c.Response.Status = 400
				return c.RenderJSON(models.ErrorResponse{
					Code:    "400",
					Message: "Unable to retrieve original notification",
				})
			}

			notificationContent.Integration = originalNotification.NotificationContent.Integration
			integration := notificationContent.Integration

			switch integration.IntegrationSlug {
			case constants.INTEG_SLUG_GOOGLE_CLOUD:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account on Google Cloud has been accepted."
			case constants.INTEG_SLUG_BITBUCKET:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to be invited in Bitbucket has been accepted."
			case constants.INTEG_SLUG_JIRA:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account on Jira has been accepted."
			default:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account has been accepted."
			}
		} else {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "No data received",
			})
		}
		// if len(companyAdmins) != 0 {
		// 	for _, compAdmin := range companyAdmins {
		// 		data["compAdmin"] = compAdmin.UserID
		for _, userID := range userIDs {
			sendNotificationToUser(c, userID, requesterInfo, requestType, true, notificationID, notificationContent)
		}
		// 	}
		// }
	}

	notification, opsErr := ops.GetNotificationByID(notificationID)
	if opsErr == nil {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":s": {
					BOOL: aws.Bool(true),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(notification.PK),
				},
				"SK": {
					S: aws.String(notification.SK),
				},
			},
			UpdateExpression: aws.String("SET Seen = :s, UpdatedAt = :ua"),
		}

		res, err := app.SVC.UpdateItem(input)
		if err != nil {
		}
		_ = res
	}

	data["action"] = "Accept"
	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)

	return c.RenderJSON(data)
}

/*
****************
RejectRequest()
- Reject request depending on type (group, role, integration).
****************
*/

func (c RequestController) RejectRequest() revel.Result {
	var userIDs []string
	var requestType string
	var notificationID string
	c.Params.Bind(&requestType, "requestType")
	companyID := c.ViewArgs["companyID"].(string)
	c.Params.Bind(&userIDs, "user_id")
	c.Params.Bind(&requestType, "requestType")
	c.Params.Bind(&notificationID, "notification_id")
	data := make(map[string]interface{})
	company, opsError := ops.GetCompanyByID(companyID)
	_, _ = company, opsError
	if opsError != nil {
		return c.RenderJSON(opsError)
	}
	for _, userID := range userIDs {
		user, opsErr := ops.GetUserByIDNew(userID)
		_, _ = user, opsErr
		if opsErr != nil {
			return c.RenderJSON(opsErr)
		}

		// companyAdmins, err := ops.GetCompanyAdminsByRoleID(companyID, constants.ROLE_ID_COMPANY_ADMIN)
		// _, _ = companyAdmins, err
		// if err != nil {
		// 	c.Response.Status = 400
		// 	return c.RenderJSON(models.ErrorResponse{
		// 		Code:    "400",
		// 		Message: "Unable to retrieve Company Admins",
		// 	})
		// }
		requesterInfo, err := ops.GetCompanyMember(ops.GetCompanyMemberParams{
			UserID:    userID,
			CompanyID: companyID,
		}, c.Controller)
		if err != nil {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "Unable to retrieve Company user",
			})
		}
		_, _ = requesterInfo, err
		notificationContent := models.NotificationContentType{
			RequesterUserID: requesterInfo.UserID,
			ActiveCompany:   companyID,
			IsAccepted:      "REJECTED",
		}
		if groupID := c.Params.Get("group_id"); groupID != "" {
			groupID := c.Params.Form.Get("group_id")
			group, err := GetGroupByID(groupID)
			if err != nil {
				data["error"] = "Group not exists."
				data["status"] = utils.GetHTTPStatus(err.Error())
				return c.RenderJSON(data)
			}
			notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to join " + group.GroupName + " has been rejected."
			notificationContent.GroupID = groupID
		} else if roleIDs := c.Params.Values["role_id[]"]; len(roleIDs) > 0 {
			c.Params.Bind(&roleIDs, "role_id")
			var rolesRequested []string
			for _, roleID := range roleIDs {
				role, opsErr := ops.GetRoleByID(roleID, companyID)
				_, _ = role, opsErr
				if opsErr != nil {
					return c.RenderJSON(opsErr)
				}
				rolesRequested = append(rolesRequested, role.RoleName)
				if len(rolesRequested) == 1 {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to take on the role of " + rolesRequested[0] + " has been rejected."
				} else {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to take on the following roles: " + strings.Join(rolesRequested, ", ") + " has been rejected."
				}
				notificationContent.RolesRequested = roleIDs
			}
		} else if integrationIDs := c.Params.Values["integration_id[]"]; len(integrationIDs) > 0 {
			c.Params.Bind(&integrationIDs, "integration_id")
			var requestAction string
			if requestType == constants.REQUEST_CONNECT_INTEGRATION {
				requestAction = "connect"
			} else {
				requestAction = "disconnect"
			}
			var integrationsRequested []string
			for _, integ := range integrationIDs {
				integInfo, err := ops.GetIntegrationByID(integ)
				if err != nil {
					c.Response.Status = 400
					return c.RenderJSON(models.ErrorResponse{
						Code:    "400",
						Message: "Unable to retrieve Integrations",
					})
				}
				integrationsRequested = append(integrationsRequested, integInfo.IntegrationName)
				if len(integrationsRequested) == 1 {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " to " + integrationsRequested[0] + " has been rejected."
				} else {
					notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to " + requestAction + " to the following integrations: " + strings.Join(integrationsRequested, ", ") + " has been rejected."
				}
				notificationContent.RequestedIntegrations = integrationIDs
			}
		} else if requestType == "REQUEST_TO_CREATE_ACCOUNT" {
			originalNotification, opsErr := ops.GetNotificationByID(notificationID)
			if opsErr != nil {
				c.Response.Status = 400
				return c.RenderJSON(models.ErrorResponse{
					Code:    "400",
					Message: "Unable to retrieve original notification",
				})
			}

			notificationContent.Integration = originalNotification.NotificationContent.Integration
			integration := notificationContent.Integration

			switch integration.IntegrationSlug {
			case constants.INTEG_SLUG_GOOGLE_CLOUD:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account on Google Cloud has been rejected."
			case constants.INTEG_SLUG_BITBUCKET:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to be invited in Bitbucket has been rejected."
			case constants.INTEG_SLUG_JIRA:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account on Jira has been rejected."
			default:
				notificationContent.Message = requesterInfo.FirstName + " " + requesterInfo.LastName + "'s request to create an account has been rejected."
			}
		} else {
			c.Response.Status = 400
			return c.RenderJSON(models.ErrorResponse{
				Code:    "400",
				Message: "No data received",
			})
		}

		// if len(companyAdmins) != 0 {
		// 	for _, compAdmin := range companyAdmins {
		sendNotificationToUser(c, user.UserID, requesterInfo, requestType, false, notificationID, notificationContent)
		// 	}
		// }
	}

	notification, opsErr := ops.GetNotificationByID(notificationID)
	if opsErr == nil {
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":s": {
					BOOL: aws.Bool(true),
				},
				":ua": {
					S: aws.String(utils.GetCurrentTimestamp()),
				},
			},
			TableName: aws.String(app.TABLE_NAME),
			Key: map[string]*dynamodb.AttributeValue{
				"PK": {
					S: aws.String(notification.PK),
				},
				"SK": {
					S: aws.String(notification.SK),
				},
			},
			UpdateExpression: aws.String("SET Seen = :s, UpdatedAt = :ua"),
		}

		res, err := app.SVC.UpdateItem(input)
		if err != nil {
		}
		_ = res
	}

	data["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(data)
}

func sendNotificationToUser(c RequestController, companyAdminUserID string, requesterUserInfo models.CompanyUser, typeOfRequest string, methodOfRequest bool, notificationID string, notificationContentFromRequest models.NotificationContentType) revel.Result {
	companyID := c.ViewArgs["companyID"].(string)
	data := make(map[string]interface{})
	var method string
	if methodOfRequest {
		method += "ACCEPTED"
	} else {
		method += "REJECTED"
	}

	if err := UpdateNotification(notificationID, method, notificationContentFromRequest); err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to update notification: " + err.Error(),
		})
	}
	notificationContent := models.NotificationContentType{
		RequesterUserID: requesterUserInfo.UserID,
		ActiveCompany:   companyID,
	}
	switch typeOfRequest {
	case constants.REQUEST_COMPANY_ROLE_UPDATE:
		var rolesRequested []string
		message := "Your request to take on the "
		for _, roleID := range notificationContentFromRequest.RolesRequested {
			role, opsErr := ops.GetRoleByID(roleID, companyID)
			_, _ = role, opsErr
			if opsErr != nil {
				return c.RenderJSON(opsErr)
			}
			rolesRequested = append(rolesRequested, role.RoleName)
			if len(rolesRequested) == 1 {
				message += "role of " + rolesRequested[0] + " has been "
			} else {
				message += "following roles: " + strings.Join(rolesRequested, ", ") + " has been "
			}
			if methodOfRequest {
				message += "accepted."
			} else {
				message += "rejected."
			}
		}
		notificationContent.Message = message
	case constants.REQUEST_TO_JOIN_GROUP:
		group, err := GetGroupByID(notificationContentFromRequest.GroupID)
		if err != nil {
			data["error"] = "Group not exists."
			data["status"] = utils.GetHTTPStatus(err.Error())
			return c.RenderJSON(data)
		}
		message := "Your request to join " + group.GroupName + " has been "
		if methodOfRequest {
			message += "accepted."
		} else {
			message += "rejected."
		}
		notificationContent.Message = message
	case constants.REQUEST_CONNECT_INTEGRATION, constants.REQUEST_DISCONNECT_INTEGRATION:
		var requestAction string
		if typeOfRequest == constants.REQUEST_CONNECT_INTEGRATION {
			requestAction = "connect"
		} else {
			requestAction = "disconnect"
		}
		var integrationsRequested []string
		message := "Your request to " + requestAction + " to  "
		for _, integ := range notificationContentFromRequest.RequestedIntegrations {
			integInfo, err := ops.GetIntegrationByID(integ)
			if err != nil {
				c.Response.Status = 400
				return c.RenderJSON(models.ErrorResponse{
					Code:    "400",
					Message: "Unable to retrieve Integrations",
				})
			}
			integrationsRequested = append(integrationsRequested, integInfo.IntegrationName)
			if len(integrationsRequested) == 1 {
				message += integrationsRequested[0] + " has been "
			} else {
				message += " the following integrations: " + strings.Join(integrationsRequested, ", ") + " has been "
			}
		}
		if methodOfRequest {
			message += "checked and is in progress for connection."
		} else {
			message += "rejected."
		}
		notificationContent.Message = message
	case constants.REQUEST_TO_CREATE_ACCOUNT:
		notificationContent.Integration = models.NotificationIntegration{
			IntegrationID:   notificationContentFromRequest.Integration.IntegrationID,
			IntegrationSlug: notificationContentFromRequest.Integration.IntegrationSlug,
			IntegrationName: notificationContentFromRequest.Integration.IntegrationName,
		}
		integration := notificationContent.Integration

		var message string
		switch integration.IntegrationSlug {
		case constants.INTEG_SLUG_GOOGLE_CLOUD:
			if methodOfRequest {
				message = "Your request to create an account on Google Cloud has been accepted. Please wait for your account to be created."
			} else {
				message = "Your request to create an account on Google Cloud has been rejected."
			}
		case constants.INTEG_SLUG_BITBUCKET:
			if methodOfRequest {
				message = "Your request to be invited in Bitbucket has been accepted. Please wait for the invitation email."
			} else {
				message = "Your request to be invited in Bitbucket has been rejected."
			}
		case constants.INTEG_SLUG_JIRA:
			if methodOfRequest {
				message = "Your request to create an account on Jira has been accepted. Please wait for your account to be created."
			} else {
				message = "Your request to create an account on Jira has been rejected."
			}
		default:
			if methodOfRequest {
				message = "Your request to create an account has been accepted. Please wait for your account to be created."
			} else {
				message = "Your request to create an account has been rejected."
			}
		}
		notificationContent.Message = message
	default:
		notificationContent.Message = "Error reply failed."
	}
	// utils.PrintJSON(companyAdminUserID, requesterUserInfo.UserID, notificationID, typeOfRequest, notificationContent.Message)
	createdNotification, err := ops.CreateNotification(ops.CreateNotificationInput{
		UserID:              requesterUserInfo.UserID,
		NotificationType:    constants.REQUEST_STATUS_UPDATE,
		NotificationContent: notificationContent,
		Global:              false,
	}, c.Controller)

	if err != nil {
		c.Response.Status = 400
		return c.RenderJSON(models.ErrorResponse{
			Code:    "400",
			Message: "Unable to create notification for " + companyAdminUserID,
		})
	}
	// utils.PrintJSON(createdNotification)
	return c.RenderJSON(createdNotification)
}

func UpdateNotification(notificationID string, requestMethod string, notificationContentFromRequest models.NotificationContentType) error {
	notification, err := GetNotificationByID(notificationID)
	if err != nil || notification.NotificationID == "" {
		return errors.New("notification not found")
	}
	notificationContentMap, err := dynamodbattribute.MarshalMap(notificationContentFromRequest)
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":nt": {
				M: notificationContentMap,
			},
			":ct": {
				S: aws.String(utils.GetCurrentTimestamp()),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(notification.PK),
			},
			"SK": {
				S: aws.String(notification.SK),
			},
		},
		TableName:        aws.String(app.TABLE_NAME),
		UpdateExpression: aws.String("SET NotificationContent = :nt, UpdatedAt = :ct"),
	}
	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		return err
	}

	return nil
}
