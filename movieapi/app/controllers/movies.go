package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"movieapi/app"
	"movieapi/app/models"
	"strings"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := app.DynamoDBClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("Movies"),
	})
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	movies, err := c.processMovies(result.Items)
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	return c.RenderJSON(movies)
}

func (c MoviesController) processMovies(items []map[string]types.AttributeValue) ([]models.Movie, error) {
	var movies []models.Movie
	for _, item := range items {
		var movie models.Movie
		tempItem := make(map[string]types.AttributeValue)
		for k, v := range item {
			if k != "ID" {
				tempItem[k] = v
			}
		}

		if err := attributevalue.UnmarshalMap(tempItem, &movie); err != nil {
			return nil, fmt.Errorf("error processing movie data")
		}

		switch idAttr := item["ID"].(type) {
		case *types.AttributeValueMemberS:
			if parsedID, err := ulid.Parse(idAttr.Value); err == nil {
				movie.ID = parsedID
				movies = append(movies, movie)
			}
		default:
			return nil, fmt.Errorf("invalid ID attribute type")
		}
	}
	return movies, nil
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

func (c MoviesController) DeleteMovie(id string) revel.Result {
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Validate ULID
	if _, err := ulid.Parse(id); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid ID format"})
	}

	_, err := app.DynamoDBClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	return c.RenderJSON(map[string]string{})
}
func (c MoviesController) UpdateMovie(id string) revel.Result {
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Validate ULID
	if _, err := ulid.Parse(id); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid ID format"})
	}

	var movie models.Movie
	if err := c.Params.BindJSON(&movie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid input"})
	}

	updateExpr := "SET"
	exprAttrNames := map[string]string{}
	exprAttrValues := map[string]types.AttributeValue{}
	updateParts := []string{}

	if movie.Title != "" {
		updateParts = append(updateParts, "#title = :title")
		exprAttrNames["#title"] = "Title"
		exprAttrValues[":title"] = &types.AttributeValueMemberS{Value: movie.Title}
	}
	if movie.Plot != "" {
		updateParts = append(updateParts, "#plot = :plot")
		exprAttrNames["#plot"] = "Plot"
		exprAttrValues[":plot"] = &types.AttributeValueMemberS{Value: movie.Plot}
	}
	if movie.Rating > 0 {
		updateParts = append(updateParts, "#rating = :rating")
		exprAttrNames["#rating"] = "Rating"
		exprAttrValues[":rating"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%.1f", movie.Rating)}
	}
	if movie.Year != "" {
		updateParts = append(updateParts, "#year = :year")
		exprAttrNames["#year"] = "Year"
		exprAttrValues[":year"] = &types.AttributeValueMemberS{Value: movie.Year}
	}

	if len(updateParts) == 0 {
		return c.RenderJSON(map[string]string{"error": "No fields to update"})
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          aws.String(updateExpr + " " + strings.Join(updateParts, ", ")),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ReturnValues:              types.ReturnValueUpdatedNew,
	}

	result, err := app.DynamoDBClient.UpdateItem(context.Background(), input)
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	var updatedMovie models.Movie
	if err := attributevalue.UnmarshalMap(result.Attributes, &updatedMovie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Error processing updated data"})
	}

	// Create response with only updated fields
	response := make(map[string]interface{})
	for k, v := range result.Attributes {
		switch av := v.(type) {
		case *types.AttributeValueMemberS:
			response[k] = av.Value
		case *types.AttributeValueMemberN:
			response[k] = av.Value
		}
	}

	return c.RenderJSON(response)
}
