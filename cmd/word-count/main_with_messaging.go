package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/net/http2"
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
func (W *WordCounter) sortByCount(m map[string]int) []WordCount {
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

func (W *WordCounter) CountWordsFromURL(ctx context.Context, url string) error {
	fmt.Println("Counting words from URL:", url)

	// request with context
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// handle error
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, W.Client.Timeout)
	defer cancel()
	req = req.WithContext(reqCtx)

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

	localCounter := make(map[string]int)
	for _, word := range strings.Fields(string(body)) {
		w := strings.ToLower(word)
		if W.isValidWord(w) && W.WordBank[w] {
			localCounter[w]++
		}
	}

	W.Sync.Lock()
	for word, count := range localCounter {
		W.Counter[word] += count
	}
	W.Sync.Unlock()

	return nil
}

// Producer
func producer(queue chan<- string, urls []string) {
	for _, url := range urls {
		queue <- url
	}
	close(queue)
}

// Consumer
func consumer(ctx context.Context, queue <-chan string, wg *sync.WaitGroup, sharedCounter *WordCounter) {
	for url := range queue {
		// Word count logic
		sharedCounter.CountWordsFromURL(context.TODO(), url)
	}
	wg.Done()
}

func main() {

	var timeout time.Duration
	var globalTimeout time.Duration
	var wordBankUrl string
	var essaysPath string
	var concurrencyLimit int
	var numConsumers int
	// var retries int

	flag.IntVar(&numConsumers, "numConsumers", 20, "Number of consumers")
	// flag.IntVar(&retries, "retries", 3, "Number of retries")
	flag.DurationVar(&timeout, "timeout", 90*time.Second, "HTTP client timeout")
	flag.DurationVar(&globalTimeout, "globalTimeout", 120*time.Second, "Global context timeout")
	flag.StringVar(&wordBankUrl, "wordBankUrl", "https://raw.githubusercontent.com/dwyl/english-words/master/words.txt", "Word bank URL")
	flag.StringVar(&essaysPath, "essaysPath", "./resources/endg-urls-copy.txt", "Path to essays")
	flag.IntVar(&concurrencyLimit, "concurrencyLimit", 50, "Concurrency limit")
	flag.Parse()

	httpTransport := &http.Transport{
		//IdleConnTimeout is the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit.
		IdleConnTimeout:       globalTimeout,
		ResponseHeaderTimeout: timeout,
		MaxIdleConnsPerHost:   concurrencyLimit,
	}

	// Upgrade it to HTTP/2
	http2.ConfigureTransport(httpTransport)

	sharedCounter := NewWordCounter(concurrencyLimit, &http.Client{
		Transport: httpTransport,
		Timeout:   timeout,
	})

	// Load bank words from url, and store in memory
	bankOfWords, err := sharedCounter.LoadBankWord(wordBankUrl)
	if err != nil {
		fmt.Println("Error loading words:", err)
		return
	}

	if len(bankOfWords) == 0 {
		fmt.Println("Error: no words found")
		return
	}
	sharedCounter.WordBank = bankOfWords

	// Open the file and push the messages to the consumers
	file, err := os.Open(essaysPath)
	if err != nil {
		fmt.Printf("Error opening file %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	urls := []string{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		urls = append(urls, strings.TrimSpace(line))
	}

	// Initialize queue and wait group
	fmt.Println("Number of urls:", len(urls))
	queue := make(chan string, len(urls))
	var wg sync.WaitGroup

	// Producer pushes URLs into the queue
	go producer(queue, urls)

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	// Spin up multiple consumers
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go consumer(ctx, queue, &wg, sharedCounter)
	}

	// Wait for all consumers to finish
	wg.Wait()

	// Print the top 10 most used words in order from shared counter
	mostUsedWords := sharedCounter.sortByCount(sharedCounter.Counter)

	// Convert the array of maps to a pretty JSON string
	prettyJSON, err := json.MarshalIndent(mostUsedWords, "", "  ")
	if err != nil {
		fmt.Println("Failed to generate json", err)
		return
	}

	// Print the pretty JSON string
	fmt.Println(string(prettyJSON))

}
