package models

import (
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Movie struct with correct field types for ULID and JSON tags
type Movie struct {
	ID     ulid.ULID `json:"id"`
	Title  string    `json:"title"`
	Plot   string    `json:"plot"`
	Rating float64   `json:"rating"` // Ratings are typically floats to accommodate decimal values
	Year   string    `json:"year"`
}

// NewMovie creates a new Movie instance with an auto-generated ULID
func NewMovie(title, plot string, rating float64, year string) (*Movie, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), ulid.Monotonic(nil, 0))
	if err != nil {
		return nil, err
	}
	return &Movie{
		ID:     id,
		Title:  title,
		Plot:   plot,
		Rating: rating,
		Year:   year,
	}, nil
}

// MarshalJSON customizes JSON serialization for ULID
func (m Movie) MarshalJSON() ([]byte, error) {
	type Alias Movie // Create an alias to avoid recursion in JSON marshalling
	return json.Marshal(&struct {
		ID string `json:"id"`
		*Alias
	}{
		ID:    m.ID.String(), // Convert ULID to string for JSON
		Alias: (*Alias)(&m),
	})
}

// UnmarshalJSON customizes JSON deserialization for ULID
func (m *Movie) UnmarshalJSON(data []byte) error {
	type Alias Movie
	aux := &struct {
		ID string `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ID != "" { // Only parse if ID is present
		var err error
		m.ID, err = ulid.Parse(aux.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
