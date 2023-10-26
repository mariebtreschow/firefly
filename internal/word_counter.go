package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type WordCounter struct {
	Sync             *sync.Mutex
	Client           *http.Client
	Counter          map[string]int
	WordBank         map[string]bool
	ConcurrencyLimit int
}

type WordCount struct {
	Word  string
	Count int
}

func NewWordCounter(concurrencyLimit int, httpClient *http.Client) *WordCounter {
	return &WordCounter{
		Sync:             &sync.Mutex{},
		Client:           httpClient,
		Counter:          make(map[string]int),
		WordBank:         make(map[string]bool),
		ConcurrencyLimit: concurrencyLimit,
	}
}

func (W *WordCounter) isValidWord(word string) bool {
	return len(word) >= 3 && isAlphabetic(word)
}

func isAlphabetic(s string) bool {
	for _, char := range s {
		if !unicode.IsLetter(char) {
			return false
		}
	}
	return true
}

// we want to load it every time the program starts: because we do not know if it has changed
// TODO: make alternative where we cache the words
func (W *WordCounter) LoadBankWord(url string) (map[string]bool, error) {
	fmt.Println("Loading words from url", url)
	resp, err := W.Client.Get(url)
	if err != nil {
		return nil, err
	}
	// close the body when we are done
	defer resp.Body.Close()

	words := make(map[string]bool) // Initialize the map

	// using bufio to simply and efficiently read through the list of words
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {

		if W.isValidWord(scanner.Text()) {
			if words[strings.ToLower(scanner.Text())] {
				continue
			}
			words[strings.ToLower(scanner.Text())] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

// sort the list of words by the count
func (W *WordCounter) SortByCount(m map[string]int) []WordCount {
	var sorted []WordCount
	for key, value := range m {
		sorted = append(sorted, WordCount{key, value})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	var topWords []WordCount
	count := 0
	for _, v := range sorted {
		topWords = append(topWords, v)
		if count >= 10 {
			break
		}
		count++
	}
	return topWords

}

func (W *WordCounter) CountWordsFromURL(ctx context.Context, url string) (map[string]int, error) {
	fmt.Println("Counting words from URL:", url)

	// request with context
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// handle error
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, W.Client.Timeout)
	defer cancel()
	req = req.WithContext(reqCtx)

	resp, err := W.Client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	localCounter := make(map[string]int)
	for _, word := range strings.Fields(string(body)) {
		w := strings.ToLower(word)
		if W.isValidWord(w) && W.WordBank[w] {
			localCounter[w]++
		}
	}

	return localCounter, nil
}
