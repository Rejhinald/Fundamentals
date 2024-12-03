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

	result, err := app.DynamoDBClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("Movies"),
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

func (c MoviesController) GetMovie(id string) revel.Result {
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Validate ULID
	if _, err := ulid.Parse(id); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid ID format"})
	}

	result, err := app.DynamoDBClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	if result.Item == nil {
		return c.RenderJSON(map[string]string{"error": "Movie not found"})
	}

	var movie models.Movie
	tempItem := make(map[string]types.AttributeValue)
	for k, v := range result.Item {
		if k != "ID" {
			tempItem[k] = v
		}
	}

	if err := attributevalue.UnmarshalMap(tempItem, &movie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Error processing movie data"})
	}

	if parsedID, err := ulid.Parse(id); err == nil {
		movie.ID = parsedID
	}

	return c.RenderJSON(movie)
}
