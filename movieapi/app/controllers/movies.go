package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"movieapi/app"
	"movieapi/app/models"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/oklog/ulid/v2"
	"github.com/revel/revel"
)

type MoviesController struct {
	*revel.Controller
}

func (c MoviesController) GetMovies() revel.Result {
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	result, err := app.DynamoDBClient.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:                 aws.String("Movies"),
		AttributesToGet:           []string{},
		ConditionalOperator:       "",
		ConsistentRead:            new(bool),
		ExclusiveStartKey:         map[string]types.AttributeValue{},
		ExpressionAttributeNames:  map[string]string{},
		ExpressionAttributeValues: map[string]types.AttributeValue{},
		FilterExpression:          new(string),
		IndexName:                 new(string),
		Limit:                     new(int32),
		ProjectionExpression:      new(string),
		ReturnConsumedCapacity:    "",
		ScanFilter:                map[string]types.Condition{},
		Segment:                   new(int32),
		Select:                    "",
		TotalSegments:             new(int32),
	})
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	var movies []models.Movie
	for _, item := range result.Items {
		var movie models.Movie
		tempItem := make(map[string]types.AttributeValue)
		for k, v := range item {
			if k != "ID" {
				tempItem[k] = v
			}
		}

		if err := attributevalue.UnmarshalMap(tempItem, &movie); err != nil {
			return c.RenderJSON(map[string]string{"error": "Error processing movie data"})
		}

		switch idAttr := item["ID"].(type) {
		case *types.AttributeValueMemberS:
			if parsedID, err := ulid.Parse(idAttr.Value); err == nil {
				movie.ID = parsedID
				movies = append(movies, movie)
			}
		default:
			return c.RenderJSON(map[string]string{"error": "Invalid ID attribute type"})
		}
	}

	return c.RenderJSON(movies)
}

func (c MoviesController) CreateMovie() revel.Result {
	var movie models.Movie
	if err := c.Params.BindJSON(&movie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid input"})
	}

	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	movie.ID = ulid.MustNew(ulid.Now(), entropy)

	if app.DynamoDBClient == nil {
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

	if _, err := app.DynamoDBClient.PutItem(context.TODO(), input); err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	return c.RenderJSON(movie)
}

