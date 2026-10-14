package reviews

import (
	"strings"
	"testing"
)

func TestReviewInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   reviewInput
		wantErr bool
	}{
		{"score only", reviewInput{TMDBId: 123, Score: new(85)}, false},
		{"text only", reviewInput{TMDBId: 123, Text: "Great movie!"}, false},
		{"score and text", reviewInput{TMDBId: 123, Score: new(0), Text: "meh"}, false},
		{"max score", reviewInput{TMDBId: 123, Score: new(100)}, false},
		{"missing tmdbid", reviewInput{Score: new(85)}, true},
		{"negative tmdbid", reviewInput{TMDBId: -1, Score: new(85)}, true},
		{"empty review", reviewInput{TMDBId: 123}, true},
		{"score too low", reviewInput{TMDBId: 123, Score: new(-1)}, true},
		{"score too high", reviewInput{TMDBId: 123, Score: new(101)}, true},
		{"text too long", reviewInput{TMDBId: 123, Text: strings.Repeat("a", maxTextRune+1)}, true},
		{"text at limit", reviewInput{TMDBId: 123, Text: strings.Repeat("a", maxTextRune)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.input.validate()
			if gotErr := msg != ""; gotErr != tt.wantErr {
				t.Errorf("validate() = %q, wantErr %v", msg, tt.wantErr)
			}
		})
	}
}
