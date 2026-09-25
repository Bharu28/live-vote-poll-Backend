package handlers

import (
	"encoding/json"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestIsDuplicateVoteError(t *testing.T) {
	if !isDuplicateVoteError(mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000}}}) {
		t.Fatal("expected duplicate vote error to be recognized")
	}
	if isDuplicateVoteError(errors.New("not a duplicate")) {
		t.Fatal("expected non-duplicate error to remain false")
	}
}

func TestNormalizePollOptionsFromStrings(t *testing.T) {
	options := normalizePollOptions([]string{"Alpha", "Beta"})
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
	if options[0].Label != "Alpha" || options[0].Votes != 0 {
		t.Fatalf("unexpected first option: %#v", options[0])
	}
	if options[1].Label != "Beta" {
		t.Fatalf("unexpected second option: %#v", options[1])
	}
}

func TestNormalizePollOptionsFromObjects(t *testing.T) {
	var raw []struct {
		ID    string `json:"id"`
		Text  string `json:"text"`
		Label string `json:"label"`
	}
	if err := json.Unmarshal([]byte(`[{"id":"a","label":"Alpha"},{"text":"Beta"}]`), &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	options := normalizePollOptionObjects(raw)
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
	if options[0].Label != "Alpha" || options[1].Label != "Beta" {
		t.Fatalf("unexpected normalized values: %#v", options)
	}
}
