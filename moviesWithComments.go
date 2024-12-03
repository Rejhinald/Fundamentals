// Package controllers handles the routing and business logic for the application
package controllers

// Import necessary packages and dependencies
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
	"github.com/oklog/ulid/v2"      // Package for generating unique, lexicographically sortable IDs
	"github.com/revel/revel"        // Web framework for Go
)

// MoviesController struct embeds the revel.Controller for handling movie-related HTTP requests
type MoviesController struct {
	*revel.Controller
}

// GetMovies retrieves all movies from the DynamoDB table
func (c MoviesController) GetMovies() revel.Result {
	// Check if DynamoDB client is properly initialized
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Perform a scan operation to get all items from the Movies table
	result, err := app.DynamoDBClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("Movies"),
	})
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	// Create a slice to store the movies
	var movies []models.Movie
	// Iterate through each item in the scan result
	for _, item := range result.Items {
		var movie models.Movie
		// Create a temporary map to store item attributes excluding ID
		tempItem := make(map[string]types.AttributeValue)
		for k, v := range item {
			if k != "ID" {
				tempItem[k] = v
			}
		}

		// Unmarshal the DynamoDB attributes into a Movie struct
		if err := attributevalue.UnmarshalMap(tempItem, &movie); err != nil {
			return c.RenderJSON(map[string]string{"error": "Error processing movie data"})
		}

		// Handle the ID field separately, ensuring it's the correct type
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

	// Return the list of movies as JSON
	return c.RenderJSON(movies)
}

// CreateMovie handles the creation of a new movie in the DynamoDB table
func (c MoviesController) CreateMovie() revel.Result {
	// Create a movie struct to store the incoming data
	var movie models.Movie
	// Bind the JSON request body to the movie struct
	if err := c.Params.BindJSON(&movie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid input"})
	}

	// Generate a new ULID for the movie
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	movie.ID = ulid.MustNew(ulid.Now(), entropy)

	// Check if DynamoDB client is properly initialized
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Prepare the input for DynamoDB PutItem operation
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

	// Execute the PutItem operation
	if _, err := app.DynamoDBClient.PutItem(context.TODO(), input); err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	// Return the created movie as JSON
	return c.RenderJSON(movie)
}

// GetMovie retrieves a specific movie by its ID from the DynamoDB table
func (c MoviesController) GetMovie(id string) revel.Result {
	// Check if DynamoDB client is properly initialized
	if app.DynamoDBClient == nil {
		return c.RenderJSON(map[string]string{"error": "Server configuration error"})
	}

	// Validate that the provided ID is a valid ULID
	if _, err := ulid.Parse(id); err != nil {
		return c.RenderJSON(map[string]string{"error": "Invalid ID format"})
	}

	// Retrieve the movie item from DynamoDB
	result, err := app.DynamoDBClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return c.RenderJSON(map[string]string{"error": err.Error()})
	}

	// Check if the movie exists
	if result.Item == nil {
		return c.RenderJSON(map[string]string{"error": "Movie not found"})
	}

	// Create a movie struct to store the result
	var movie models.Movie
	// Create a temporary map to store item attributes excluding ID
	tempItem := make(map[string]types.AttributeValue)
	for k, v := range result.Item {
		if k != "ID" {
			tempItem[k] = v
		}
	}

	// Unmarshal the DynamoDB attributes into a Movie struct
	if err := attributevalue.UnmarshalMap(tempItem, &movie); err != nil {
		return c.RenderJSON(map[string]string{"error": "Error processing movie data"})
	}

	// Parse and set the movie ID
	if parsedID, err := ulid.Parse(id); err == nil {
		movie.ID = parsedID
	}

	// Return the movie as JSON
	return c.RenderJSON(movie)
}
