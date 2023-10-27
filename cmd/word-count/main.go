package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/peterbourgon/ff"
	"golang.org/x/net/http2"
	"golang.org/x/time/rate"
)

type ErrorReporter struct {
	url string
	Err error
}

type WordCounter struct {
	Sync             *sync.Mutex
	Client           *http.Client
	Counter          map[string]int
	WordBank         map[string]bool
	ErrorReporter    []ErrorReporter
	RateLimiter      *rate.Limiter
	ConcurrencyLimit int
}

type WordCount struct {
	Word  string
	Count int
}

func NewWordCounter(concurrencyLimit int, httpClient *http.Client, rateLimiter *rate.Limiter) *WordCounter {
	return &WordCounter{
		Sync:             &sync.Mutex{},
		Client:           httpClient,
		RateLimiter:      rateLimiter,
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
	fmt.Println("loading words from url", url)
	resp, err := W.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	words := make(map[string]bool) // Initialize the map

	// Using bufio to simply and efficiently read through the list of words
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

// Sort the list of words by the count
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
	fmt.Println("counting words from URL:", url)

	// Wait for a token. This will block until a token becomes available
	// or the context is cancelled.
	if err := W.RateLimiter.Wait(ctx); err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for url %s: %w", url, err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, W.Client.Timeout)
	defer cancel()
	req = req.WithContext(reqCtx)

	resp, err := W.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting response from url %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response body:", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response status code %d", resp.StatusCode)
	}

	// TODO: make this faster by using binary search
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

func isValidURL(testURL string) bool {
	u, err := url.Parse(testURL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// Producer
func producer(queue chan<- string, urls []string) {
	for _, url := range urls {
		// TODO: check if url is already in the queue
		if isValidURL(url) {
			queue <- url
		} else {
			fmt.Println("Invalid URL:", url)
		}
	}
	close(queue)
}

// Program paus after 5000 urls
var globalCounter int
var globalCounterMutex sync.Mutex

// Consumer
func consumer(ctx context.Context, queue <-chan string, wg *sync.WaitGroup, sharedCounter *WordCounter, split int) {
	for url := range queue {
		// Word count logic
		err := sharedCounter.CountWordsFromURL(ctx, url)
		if err != nil {
			// Add to error reporter
			// TODO: add functionality to put on a deadletter queue
			sharedCounter.Sync.Lock() // Locking the mutex
			sharedCounter.ErrorReporter = append(sharedCounter.ErrorReporter, ErrorReporter{url, err})
			sharedCounter.Sync.Unlock() // Unlocking the mutex
			fmt.Println("Error counting words:", err)
			//  TODO: improve error handling
			if err == context.DeadlineExceeded {
				fmt.Println("context deadline exceeded")
				wg.Done()
				return
			}
			if err.Error() == "error response status code 999" {
				fmt.Println("overloading the domain")
				wg.Done()
				return
			}

		}

		// TODO: add a rate limiter instead
		// Ensures that the global counter variable is accessed by one goroutine at a time, making it thread-safe.
		globalCounterMutex.Lock()
		globalCounter++
		if globalCounter >= split {
			time.Sleep(60 * time.Second) // Pause TODO: make this configurable
			globalCounter = 0            // Reset the counter
		}
		globalCounterMutex.Unlock()
	}
	wg.Done()
}

func main() {

	fs := flag.NewFlagSet("wordcounter", flag.ExitOnError)

	var (
		timeout          = fs.Duration("timeout", 120*time.Second, "HTTP client timeout")
		globalTimeout    = fs.Duration("global_timeout", 500*time.Second, "Global context timeout for operation of all processing URLs")
		wordBankUrl      = fs.String("word_bank_url", "https://raw.githubusercontent.com/dwyl/english-words/master/words.txt", "Word bank URL")
		essaysPath       = fs.String("essays_path", "./resources/endg-urls.txt", "Path to essays")
		concurrencyLimit = fs.Int("concurrency_limit", 50, "Concurrency limit")
		numConsumers     = fs.Int("num_consumers", 40, "Number of consumers")
		errorReporter    = fs.Bool("error_reporting", false, "Display errors in the end")
		essaySplit       = fs.Int("eassays_split", 2000, "After processing 2000 essays wait for a bit")
	)

	err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarPrefix("wordcounter"))
	if err != nil {
		fmt.Println("error occurent when parsing env vars")
		return
	}

	if *timeout > *globalTimeout {
		fmt.Println("timeout cannot be greater than global timeout")
		return
	}

	if *numConsumers > 100 {
		fmt.Println("number of consumers cannot be greater than 100")
		return
	}

	httpTransport := &http.Transport{
		//IdleConnTimeout is the maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.
		IdleConnTimeout:       *globalTimeout,
		ResponseHeaderTimeout: *timeout,
		MaxIdleConnsPerHost:   *concurrencyLimit,
	}

	// Upgrade it to HTTP/2
	http2.ConfigureTransport(httpTransport)

	// Rate limiter
	r := rate.NewLimiter(rate.Every(time.Second), *concurrencyLimit)

	sharedCounter := NewWordCounter(*concurrencyLimit, &http.Client{
		Transport: httpTransport,
		Timeout:   *timeout,
	}, r)

	// Load bank words from url, and store in memory
	bankOfWords, err := sharedCounter.LoadBankWord(*wordBankUrl)
	if err != nil {
		fmt.Println("error loading words:", err)
		return
	}

	if len(bankOfWords) == 0 {
		fmt.Println("error: no words found")
		return
	}
	sharedCounter.WordBank = bankOfWords

	// Open the file and push the messages to the consumers
	file, err := os.Open(*essaysPath)
	if err != nil {
		fmt.Printf("error opening file %v", err)
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
		// TODO: validate url
		urls = append(urls, strings.TrimSpace(line))
	}

	// Initialize queue and wait group
	fmt.Println("number of urls:", len(urls))
	// Since the channel is buffered, the producer can put all the URLs into the channel without waiting
	// for a consumer to be ready to take one. This ensures that the producer doesn't get blocked.
	queue := make(chan string, len(urls))
	var wg sync.WaitGroup

	// Producer pushes URLs into the queue
	go producer(queue, urls)

	ctx, cancel := context.WithTimeout(context.Background(), *globalTimeout)
	defer cancel()

	startTime := time.Now()
	fmt.Println("start time for creating consumers:", startTime)

	// Spin up multiple consumers
	for i := 0; i < *numConsumers; i++ {
		wg.Add(1)
		go consumer(ctx, queue, &wg, sharedCounter, *essaySplit)
	}

	// Wait for all consumers to finish
	wg.Wait()

	// Print the top 10 most used words in order from shared counter
	mostUsedWords := sharedCounter.sortByCount(sharedCounter.Counter)

	// Convert the array of maps to a pretty JSON string
	prettyJSON, err := json.MarshalIndent(mostUsedWords, "", "  ")
	if err != nil {
		fmt.Println("failed to generate json", err)
		return
	}

	// Print the pretty JSON string
	fmt.Println(string(prettyJSON))

	fmt.Println("time taken to process", len(urls)+1, "URLs:", time.Since(startTime))

	// If you want to use the flag to see errors
	// TODO: explan functionality to put back the urls that wasn't processed in the queue
	if *errorReporter {
		fmt.Println("errors:", sharedCounter.ErrorReporter)
	}

}
