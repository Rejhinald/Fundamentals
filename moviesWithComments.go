
// GetMovie retrieves a specific movie by its ID from the DynamoDB table
/*
Key Terms:
- DynamoDB: A database service by Amazon that stores and retrieves data
- ULID: A unique identifier format (like an ID number) that's time-sorted
- JSON: A way to format data that's easy for both humans and computers to read
- Unmarshal: Converting data from one format (like DynamoDB) to another (like Go structs)
- Struct: A collection of fields that group related data together
- Context: A way to carry deadlines and cancellation signals across API boundaries
- Map: A collection of key-value pairs, like a dictionary

// GetMovie retrieves a single movie record from the DynamoDB database.
//
// Parameters:
//   - id: A string containing the ULID of the movie to retrieve
//
// Returns:
//   - revel.Result: Contains either:
//     * The movie data as JSON if found
//     * An error message if:
//       - The DynamoDB client isn't initialized
//       - The ID format is invalid
//       - The movie doesn't exist
//       - There's an error accessing the database
//       - There's an error processing the data
//
// The function performs the following steps:
// 1. Validates the DynamoDB client configuration
// 2. Verifies the provided ID is a valid ULID
// 3. Queries DynamoDB for the movie
// 4. If found, converts the DynamoDB data to a Movie struct
// 5. Returns the movie data or appropriate error message
*/

// GetMovie retrieves a single movie from DynamoDB by its ULID.
//
// Parameters:
//   - id: string - A ULID (Universally Unique Lexicographically Sortable Identifier) as string
//
// Returns:
//   - revel.Result - Contains either the movie data as JSON or an error message
//
// Flow:
// 1. Validates DynamoDB client initialization
// 2. Validates ULID format using ulid.Parse()
// 3. Queries DynamoDB using GetItem operation
// 4. Processes the DynamoDB response
//
// Uses:
//   - context.Background() - Creates an empty context for AWS operations
//   - dynamodb.GetItemInput - AWS SDK struct for GetItem parameters
//   - types.AttributeValue - DynamoDB attribute value interface
//   - map[string]types.AttributeValue - Go map for DynamoDB item attributes
//   - attributevalue.UnmarshalMap - AWS SDK function to convert DynamoDB map to Go struct
//   - revel.RenderJSON - Revel framework method to return JSON response
//
// Notes:
//   - Implements error handling for various failure scenarios
//   - Uses range iteration over map to process DynamoDB attributes
//   - Excludes ID field from temporary map during processing
//   - Follows RESTful API patterns





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
