package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"movieapi/app"
	"movieapi/app/models"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/revel/revel"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type MoviesController struct {
	*revel.Controller
}

func (c MoviesController) CreateMovie() revel.Result {
	var movie models.Movie
	if err := c.Params.BindJSON(&movie); err != nil {
		revel.AppLog.Warn("Error binding JSON: %v", err)
		return c.RenderJSON(map[string]string{"error": "Invalid input"})
	}

	// Log the received movie data
	revel.AppLog.Info("Movie struct after binding: %v", movie)

	// Generate a new ULID for the movie ID
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	if entropy == nil {
		revel.AppLog.Warn("Failed to initialize ULID entropy")
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}
	id := ulid.MustNew(ulid.Now(), entropy)
	movie.ID = id

	revel.AppLog.Info("Generated ULID for movie ID: %v", movie.ID)

	// Check if DynamoDB client is initialized
	if app.DynamoDBClient == nil {
		revel.AppLog.Warn("DynamoDB client is not initialized")
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("Movies"),
		Item: map[string]types.AttributeValue{
			"ID":     &types.AttributeValueMemberS{Value: movie.ID.String()},
			"Title":  &types.AttributeValueMemberS{Value: movie.Title},
			"Plot":   &types.AttributeValueMemberS{Value: movie.Plot},
			"Rating": &types.AttributeValueMemberN{Value: fmt.Sprintf("%.1f", movie.Rating)},
			"Year":   &types.AttributeValueMemberS{Value: movie.Year},
		},
	}

	revel.AppLog.Info("PutItemInput prepared: %v", input)

	_, err := app.DynamoDBClient.PutItem(context.TODO(), input)
	if err != nil {
		revel.AppLog.Warn("Error inserting item into DynamoDB: %v", err)
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	return c.RenderJSON(movie)
}
