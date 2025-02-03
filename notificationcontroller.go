package controllers

import (
	"errors"
	"fmt"

	"grooper/app"
	"grooper/app/constants"
	"grooper/app/models"
	ops "grooper/app/operations"
	"grooper/app/utils"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/revel/revel"
)

type NotificationController struct {
	*revel.Controller
}

/*
****************
GetCurrentUserNotifications()
- Returns all notifications of the current user who requested this enpoint
****************
*/
// @Summary Get All the Current Notifications of the User
// @Description This endpoint retrieves the current Notifications of the User, returning a 200 OK response, notifications if there are any, and the unread notifications in total.
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationsSuccessResponse "Successful retrieval of the Users current Notifications"
// @Router /notifications/me [get]
// @Security Authentication
func (c NotificationController) GetCurrentUserNotifications() revel.Result {
	result := make(map[string]interface{})

	currentUser := c.ViewArgs["userID"].(string)
	companyID := c.ViewArgs["companyID"].(string)

	notifications, err := ops.GetUserNotifications(currentUser, companyID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	globalNotifications, err := ops.GetGlobalUserNotifications(currentUser)
	if err != nil {
		fmt.Println("ERROR")
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	// utils.PrintJSON(globalNotifications, "Global Notifs")

	unread := 0

	// insert notification content details
	for _, notification := range notifications {
		// 	if notification.NotificationContent.Integration.IntegrationID != "" {
		// 		result, err := GetIntegrationByID(notification.NotificationContent.Integration.IntegrationID)
		// 		if err == nil {
		// 			notifications[i].Content = result
		// 		}
		// 	}
		if !notification.Seen {
			unread = unread + 1
		}
	}

	for _, notification := range globalNotifications {
		// 	if notification.NotificationContent.Integration.IntegrationID != "" {
		// 		result, err := GetIntegrationByID(notification.NotificationContent.Integration.IntegrationID)
		// 		if err == nil {
		// 			notifications[i].Content = result
		// 		}
		// 	}
		if !notification.Seen {
			unread = unread + 1
		}
	}

	notifcationCollection := append(notifications, globalNotifications...)

	result["unread"] = unread
	result["notifications"] = notifcationCollection
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

// func GetUserNotifications(userID string) ([]models.Notification, error) {
// 	var notifications []models.Notification

// 	params := &dynamodb.QueryInput{
// 		TableName: aws.String(app.TABLE_NAME),
// 		KeyConditions: map[string]*dynamodb.Condition{
// 			"SK": {
// 				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
// 				AttributeValueList: []*dynamodb.AttributeValue{
// 					{
// 						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, userID)),
// 					},
// 				},
// 			},
// 			"PK": {
// 				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
// 				AttributeValueList: []*dynamodb.AttributeValue{
// 					{
// 						S: aws.String(constants.PREFIX_NOTIFICATION),
// 					},
// 				},
// 			},
// 		},
// 		IndexName: aws.String(constants.INDEX_NAME_INVERTED_INDEX),
// 	}

// 	res, err := app.SVC.Query(params)
// 	if err != nil {
// 		e := errors.New(constants.HTTP_STATUS_500)
// 		return notifications, e
// 	}

// 	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &notifications)
// 	if err != nil {
// 		e := errors.New(constants.HTTP_STATUS_400)
// 		return notifications, e
// 	}

// 	return notifications, nil
// }

/*
****************
SeenNotification()
- Mark notification as read by updating Seen to true
****************
*/
// @Summary When the User Seen the notification.
// @Description This endpoint Patches the Notifications status when it's seen. Returns a status 200 OK upon sucess.
// @Tags notifications
// @Produce json
// @Param notificationID path string true "Notification ID"
// @Success 200 {object} models.NotificationsSuccessStatus "Successfully updated notification status"
// @Router /notifications/seen/:notificationID [patch]
// @Security Authentication
func (c NotificationController) SeenNotification(notificationID string) revel.Result {
	result := make(map[string]interface{})

	notification, err := GetNotificationByID(notificationID)
	if err != nil || notification.NotificationID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

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

	_, err = app.SVC.UpdateItem(input)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
MarkAllAsRead()
****************
*/
// @Summary Marks all notifications as read.
// @Description This endpoint updates the status of all notifications to "seen" for the current user. returns a status 200 OK upon success updated notifications.
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationsSuccessStatus "All notifications marked as read successfully"
// @Router /notifications/me/read_all [put]
// @Security Authentication
func (c NotificationController) MarkAllAsRead() revel.Result {
	result := make(map[string]interface{})

	currentUser := c.ViewArgs["userID"].(string)
	companyID := c.ViewArgs["companyID"].(string)

	notifications, err := ops.GetUserNotifications(currentUser, companyID)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	for _, notification := range notifications {
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
		// handle errors
		_, err = app.SVC.UpdateItem(input)
		if err != nil {
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
			return c.RenderJSON(result)
		}
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
DeleteNotification()
- Remove notification by id
****************
*/
// @Summary Deletes a notification.
// @Description This endpoint Deletes a notification from the current user, returns a status 200 OK upon the success deletion of the notification.
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationsSuccessResponse "Successfully deleted a notification"
// @Router /notifications/:notificationID [delete]
// @Security Authentication
func (c NotificationController) DeleteNotification(notificationID string) revel.Result {
	result := make(map[string]interface{})

	notification, err := GetNotificationByID(notificationID)
	if err != nil || notification.NotificationID == "" {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
		return c.RenderJSON(result)
	}

	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(notification.PK),
			},
			"SK": {
				S: aws.String(notification.SK),
			},
		},
		TableName: aws.String(app.TABLE_NAME),
	}

	_, err = app.SVC.DeleteItem(input)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_500)
		return c.RenderJSON(result)
	}

	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}

/*
****************
GetNotificationByID()
- Return notification details
****************
*/
func GetNotificationByID(notificationID string) (models.Notification, error) {
	var notification models.Notification

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_NOTIFICATION, notificationID)),
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
	}

	res, err := app.SVC.Query(params)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return notification, e
	}

	if len(res.Items) != 0 {
		err = dynamodbattribute.UnmarshalMap(res.Items[0], &notification)
		if err != nil {
			e := errors.New(constants.HTTP_STATUS_400)
			return notification, e
		}
	}

	return notification, nil
}

/*
****************
GetAllNotifications()
- Get all notif for testing only
****************
*/

func GetUserNotifications() ([]models.Notification, error) {
	var notifications []models.Notification

	params := &dynamodb.ScanInput{
		TableName:        aws.String(app.TABLE_NAME),
		FilterExpression: aws.String("begins_with(PK, :prefix)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":prefix": {
				S: aws.String(constants.PREFIX_NOTIFICATION),
			},
		},
	}

	result, err := app.SVC.Scan(params)
	if err != nil {
		return notifications, err
	}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &notifications)
	if err != nil {
		return notifications, err
	}

	return notifications, nil
}

// @Summary Retrieves All the notifications.
// @Description This endpoint retrieves all the users notificatiion. returns the details of the notifications of all the user.
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationsSuccessResponse "Successfully retrieves of all the notification details"
// @Router /all-notifications [get]
// @Security Authentication
func (c NotificationController) GetAllNotifications() revel.Result {
	// Get all notifications
	notifications, err := GetUserNotifications()
	if err != nil {
		// Handle error
		return c.RenderText("Error fetching notifications: " + err.Error())
	}

	// Render the notifications as JSON
	return c.RenderJSON(notifications)
}

func PaginateUserNotifications(currentUser, companyID, exclusiveStartKey string, pageLimit int64, all bool, sortOutput bool) ([]models.Notification, models.Notification, error) {
	notifications := []models.Notification{}
	lastEvaluatedKey := models.Notification{}

	params := &dynamodb.QueryInput{
		TableName: aws.String(app.TABLE_NAME),
		KeyConditions: map[string]*dynamodb.Condition{
			"SK": {
				ComparisonOperator: aws.String(constants.CONDITION_EQUAL),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(utils.AppendPrefix(constants.PREFIX_USER, currentUser)),
					},
				},
			},
			"PK": {
				ComparisonOperator: aws.String(constants.CONDITION_BEGINS_WITH),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(constants.PREFIX_NOTIFICATION),
					},
				},
			},
		},
		FilterExpression: aws.String("NotificationContent.ActiveCompany = :companyID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":companyID": {
				S: aws.String(companyID),
			},
		},
		IndexName:         aws.String(constants.INDEX_NAME_INVERTED_INDEX),
		Limit:             aws.Int64(pageLimit),
		ExclusiveStartKey: utils.MarshalLastEvaluatedKey(lastEvaluatedKey, exclusiveStartKey),
		ScanIndexForward:  aws.Bool(sortOutput),
	}

	result, err := ops.HandleQueryWithLimit(params, int(pageLimit), all)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_500)
		return notifications, lastEvaluatedKey, e
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &notifications)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return notifications, lastEvaluatedKey, e
	}

	key := result.LastEvaluatedKey

	err = dynamodbattribute.UnmarshalMap(key, &lastEvaluatedKey)
	if err != nil {
		e := errors.New(constants.HTTP_STATUS_400)
		return notifications, lastEvaluatedKey, e
	}

	return notifications, lastEvaluatedKey, nil
}

// @Summary Get User Notifications
// @Description This endpoint retrieves all notifications for the current user with pagination support, returning a 200 OK upon success.
// @Tags admin/notifications
// @Produce json
// @Param sort_order query string false "Sort order: ascending/descending"
// @Success 200 {object} models.LoginSuccessResponse
// @Router /notifications/pagination [get]
func (c NotificationController) GetUserNotifications() revel.Result {
	result := make(map[string]interface{})

	currentUser := c.ViewArgs["userID"].(string)
	companyID := c.ViewArgs["companyID"].(string)
	paramLastEvaluatedKey := c.Params.Query.Get("lastEvaluatedKey")
	pageLimit := int64(constants.DEFAULT_PAGE_LIMIT)
	allCondition := c.Params.Query.Get("all")
	all, _ := strconv.ParseBool(allCondition)
	sortOrder := c.Params.Query.Get("sort_order")

	sortOutput := false
	if sortOrder != "" {
		if sortOrder == "descending" {
			sortOutput = true
		} else if sortOrder == "ascending" {
			sortOutput = false
		} else {
			c.Response.Status = 400
			result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_400)
			return c.RenderJSON(result)
		}
	}

	notifications, lastEvaluatedKey, err := PaginateUserNotifications(currentUser, companyID, paramLastEvaluatedKey, pageLimit, all, sortOutput)
	if err != nil {
		result["status"] = utils.GetHTTPStatus(err.Error())
		return c.RenderJSON(result)
	}

	unread := 0

	// insert notification content details
	for _, notification := range notifications {
		if !notification.Seen {
			unread = unread + 1
		}
	}

	result["unread"] = unread
	result["notifications"] = notifications
	result["lastEvaluatedKey"] = lastEvaluatedKey
	result["status"] = utils.GetHTTPStatus(constants.HTTP_STATUS_200)
	return c.RenderJSON(result)
}
