package main

// import (
// 	"bufio"
// 	"context"
// 	"flag"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"os"
// 	"sort"
// 	"strings"
// 	"sync"
// 	"time"
// 	"unicode"
// )

// type WordCounter struct {
// 	Sync             *sync.Mutex
// 	Client           *http.Client
// 	Counter          map[string]int
// 	WordBank         map[string]bool
// 	ConcurrencyLimit int
// }

// type WordCount struct {
// 	Word  string
// 	Count int
// }

// func NewWordCounter(concurrencyLimit int, httpClient *http.Client) *WordCounter {
// 	return &WordCounter{
// 		Sync:             &sync.Mutex{},
// 		Client:           httpClient,
// 		Counter:          make(map[string]int),
// 		WordBank:         make(map[string]bool),
// 		ConcurrencyLimit: concurrencyLimit,
// 	}
// }

// func (W *WordCounter) isValidWord(word string) bool {
// 	return len(word) >= 3 && isAlphabetic(word)
// }

// func isAlphabetic(s string) bool {
// 	for _, char := range s {
// 		if !unicode.IsLetter(char) {
// 			return false
// 		}
// 	}
// 	return true
// }

// // we want to load it every time the program starts: because we do not know if it has changed
// // TODO: make alternative where we cache the words
// func (W *WordCounter) LoadBankWord(url string) (map[string]bool, error) {
// 	fmt.Println("Loading words from url", url)
// 	resp, err := W.Client.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// close the body when we are done
// 	defer resp.Body.Close()

// 	words := make(map[string]bool) // Initialize the map

// 	// using bufio to simply and efficiently read through the list of words
// 	scanner := bufio.NewScanner(resp.Body)
// 	for scanner.Scan() {

// 		if W.isValidWord(scanner.Text()) {
// 			if words[scanner.Text()] {
// 				continue
// 			}
// 			words[scanner.Text()] = true
// 		}
// 	}
// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	return words, nil
// }

// // sort the list of words by the count
// func (W *WordCounter) sortByCount(m map[string]int) []WordCount {
// 	var sorted []WordCount
// 	for key, value := range m {
// 		sorted = append(sorted, WordCount{key, value})
// 	}

// 	sort.Slice(sorted, func(i, j int) bool {
// 		return sorted[i].Count > sorted[j].Count
// 	})
// 	return sorted

// }

// func (W *WordCounter) CountWordsFromURL(ctx context.Context, url string) error {
// 	fmt.Println("Counting words from URL:", url)

// 	// request with context
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		// handle error
// 		return err
// 	}

// 	reqCtx, cancel := context.WithTimeout(ctx, W.Client.Timeout)
// 	defer cancel()
// 	req = req.WithContext(reqCtx)

// 	resp, err := W.Client.Do(req)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return err
// 	}

// 	localCounter := make(map[string]int)
// 	for _, word := range strings.Fields(string(body)) {
// 		if W.isValidWord(word) && W.WordBank[word] {
// 			localCounter[word]++
// 		}
// 	}

// 	W.Sync.Lock()
// 	for word, count := range localCounter {
// 		W.Counter[word] += count
// 	}
// 	W.Sync.Unlock()

// 	return nil
// }

// func main() {

// 	var timeout time.Duration
// 	var globalTimeout time.Duration
// 	var wordBankUrl string
// 	var essaysPath string
// 	var concurrencyLimit int
// 	var retries int

// 	flag.IntVar(&retries, "retries", 3, "Number of retries")
// 	flag.DurationVar(&timeout, "timeout", 60*time.Second, "HTTP client timeout")
// 	flag.DurationVar(&globalTimeout, "globalTimeout", 120*time.Second, "Global context timeout")
// 	flag.StringVar(&wordBankUrl, "wordBankUrl", "https://raw.githubusercontent.com/dwyl/english-words/master/words.txt", "Word bank URL")
// 	flag.StringVar(&essaysPath, "essaysPath", "./resources/endg-urls-copy.txt", "Path to essays")
// 	flag.IntVar(&concurrencyLimit, "concurrencyLimit", 50, "Concurrency limit")
// 	flag.Parse()

// 	w := NewWordCounter(concurrencyLimit, &http.Client{
// 		Timeout: timeout,
// 	})

// 	// 1. load bank words from url, and store in memory
// 	bankOfWords, err := w.LoadBankWord(wordBankUrl)
// 	if err != nil {
// 		fmt.Println("Error loading words:", err)
// 		return
// 	}

// 	if len(bankOfWords) == 0 {
// 		fmt.Println("Error: no words found")
// 		return
// 	}
// 	w.WordBank = bankOfWords

// 	// 2. load essays from text file and count all the valid words
// 	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
// 	defer cancel()

// 	// Define a buffered channel and a worker pool
// 	workChannel := make(chan string, concurrencyLimit)
// 	var wg sync.WaitGroup

// 	for i := 0; i < concurrencyLimit; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for url := range workChannel {

// 				for i := 0; i < retries; i++ {
// 					err := w.CountWordsFromURL(ctx, url)
// 					if err == nil {
// 						break
// 					}
// 					fmt.Println("Retrying to count words from URL:", err)
// 					time.Sleep(time.Second * 2) // Sleep for 2 seconds before retryin
// 				}

// 			}
// 		}()
// 	}

// 	// Open the file and feed the URLs to the work channel
// 	file, err := os.Open(essaysPath)
// 	if err != nil {
// 		fmt.Printf("Error opening file %v", err)
// 		return
// 	}
// 	defer file.Close()

// 	reader := bufio.NewReader(file)
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			break
// 		}
// 		workChannel <- strings.TrimSpace(line)
// 	}
// 	close(workChannel)
// 	wg.Wait()

// 	// 3. print the top 10 most used words in order
// 	sortedKeys := w.sortByCount(w.Counter)
// 	count := 0
// 	for _, v := range sortedKeys {
// 		if count >= 10 {
// 			break
// 		}
// 		fmt.Printf("%s: %d\n", v.Word, v.Count)
// 		count++
// 	}

// }
