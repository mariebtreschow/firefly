package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
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
	fmt.Printf("Loading words from url %s", url)
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
			if words[scanner.Text()] {
				continue
			}
			words[scanner.Text()] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

// sort the list of words by the count
func (W *WordCounter) sortByCount(m map[string]int) []WordCount {
	var sorted []WordCount
	for key, value := range m {
		sorted = append(sorted, WordCount{key, value})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})
	return sorted

}

func (W *WordCounter) CountWordsFromURL(ctx context.Context, url string) error {
	fmt.Println("Counting words from URL:", url)

	// request with context
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// handle error
		return err
	}
	req = req.WithContext(ctx)

	resp, err := W.Client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	for _, word := range strings.Fields(string(body)) {
		if W.isValidWord(word) && W.WordBank[word] {
			W.Sync.Lock()
			W.Counter[word]++
			W.Sync.Unlock()
		}
	}
	return nil
}

func main() {

	// argument from dockerfile: WordBankUrl, EssaysPath, ConcurrencyLimit, Timeout
	// todo: parse as flags
	timeout := 60 * time.Second
	globalTimeout := 120 * time.Second
	wordBankUrl := "https://raw.githubusercontent.com/dwyl/english-words/master/words.txt"
	essaysPath := "./resources/endg-urls.txt"
	concurrencyLimit := 50

	// fs := flag.NewFlagSet("firefly", flag.ExitOnError)

	w := NewWordCounter(concurrencyLimit, &http.Client{
		Timeout: timeout,
	})

	// 1. load bank words from url, and store in memory
	bankOfWords, err := w.LoadBankWord(wordBankUrl)
	if err != nil {
		fmt.Println("Error loading words:", err)
		return
	}

	if len(bankOfWords) == 0 {
		fmt.Println("Error: no words found")
		return
	}
	w.WordBank = bankOfWords

	// 2. load essays from text file and count all the valid words
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	// Define a buffered channel and a worker pool
	workChannel := make(chan string, concurrencyLimit)
	var wg sync.WaitGroup

	for i := 0; i < concurrencyLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range workChannel {
				time.Sleep(1 * time.Second) // 1-second delay between each request
				if err := w.CountWordsFromURL(ctx, url); err != nil {
					fmt.Println("Failed to count words from URL:", err)
				}
			}
		}()
	}

	// Open the file and feed the URLs to the work channel
	file, err := os.Open(essaysPath)
	if err != nil {
		fmt.Printf("Error opening file %v", err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		workChannel <- scanner.Text()
	}
	close(workChannel)
	wg.Wait()

	// 3. print the top 10 most used words in order
	sortedKeys := w.sortByCount(w.Counter)
	count := 0
	for _, v := range sortedKeys {
		if count >= 10 {
			break
		}
		fmt.Printf("%s: %d\n", v.Word, v.Count)
		count++
	}

}
