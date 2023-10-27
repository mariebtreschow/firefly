package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsValidWord(t *testing.T) {
	w := NewWordCounter(10, &http.Client{})
	if !w.isValidWord("apple") {
		t.Error("Expected 'apple' to be a valid word")
	}

	if w.isValidWord("a") {
		t.Error("Expected 'a' to be an invalid word")
	}

	if w.isValidWord("123") {
		t.Error("Expected '123' to be an invalid word")
	}
}

func TestSortByCount(t *testing.T) {
	w := NewWordCounter(10, &http.Client{})
	w.Counter = map[string]int{"apple": 2, "banana": 1, "cherry": 3}
	sortedWords := w.sortByCount(w.Counter)
	if len(sortedWords) != 3 {
		t.Errorf("Expected length 3, got %v", len(sortedWords))
	}
	// Cherry has to be the first in the sorted array
	if sortedWords[0].Word != "cherry" || sortedWords[0].Count != 3 {
		t.Errorf("Expected first word to be 'cherry' with count 3, got %v with count %v", sortedWords[0].Word, sortedWords[0].Count)
	}
}

func TestCountWordsFromURL(t *testing.T) {
	// Setup a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("apple apple orange banana"))
	}))
	defer server.Close()

	// Setup your WordCounter
	wordCounter := NewWordCounter(1, &http.Client{
		Timeout: 1 * time.Second,
	})

	// Load your word bank
	// Here you might want to directly set the word bank for the test
	wordCounter.WordBank = map[string]bool{
		"apple":  true,
		"orange": true,
		"banana": true,
	}

	ctx := context.TODO()

	// Call your function
	err := wordCounter.CountWordsFromURL(ctx, server.URL)

	if err != nil {
		t.Fatalf("CountWordsFromURL() returned an error: %s", err)
	}

	// Assert the result
	if wordCounter.Counter["apple"] != 2 {
		t.Errorf("Expected apple to be counted twice, but got: %d", wordCounter.Counter["apple"])
	}

	if wordCounter.Counter["orange"] != 1 {
		t.Errorf("Expected orange to be counted once, but got: %d", wordCounter.Counter["orange"])
	}

	if wordCounter.Counter["banana"] != 1 {
		t.Errorf("Expected banana to be counted once, but got: %d", wordCounter.Counter["banana"])
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://www.example.com", true},
		{"https://www.example.com", true},
		{"http://localhost:8080", true},
		{"", false},                 // empty string
		{"www.example.com", false},  // missing scheme
		{"http:///path", false},     // missing host
		{"ftp://example.com", true}, // different scheme, still valid
		{":::", false},              // invalid URL
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			actual := isValidURL(test.url)
			if actual != test.expected {
				t.Errorf("expected %v; got %v", test.expected, actual)
			}
		})
	}
}
