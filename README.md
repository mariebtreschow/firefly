# firefly assignment

Using go 1.20 version

This application fetches the list of essays [a relative link](/resources/endg-urls) and count the top 10 word for all the essays combined.

A valid word is:
1. Contain at least 3 characters 
2. Contain only alphabetic characters
3. Be part of our [bank of words](https://raw.githubusercontent.com/dwyl/english-words/master/words.txt) (where not all the words in the bank are valid according to the previous rules)

Output is in pretty json and written to the stdout. 

## Build a Docker image and run from the command line

Please see Makefile for build command
    `make docker-build`

## How to run locally

 install dependencies
    `go mod download`

 run locally with default flags on test data
    `go run cmd/word-count/main.go`

## How to run the docker

`docker run -e TIMEOUT=60 -e GLOABL_TIMEOUT=120 -e CONCURRENCY_LIMIT=50 -e NUM_CONSUMERS=1000 -d wordcounter`

optional flags:

- timeout
- global_timeout
- word_bank_url
- essays_path
- concurrency_limit
- num_consumers

## Challenges of this application

With such a large workload, it's important to manage system resources efficiently. 

You can do this through:
- Worker Pools: Limit the number of goroutines that are active at any given time.
- Buffered Channels: Use buffered channels to manage the work queue.








Speed Enhancements
Optimized Text Parsing: Instead of splitting strings using strings.Fields, you could use more efficient text scanning libraries or algorithms optimized for your specific case.

Parallelize File Reading: If your file of URLs is large, you could parallelize reading it into memory.

Batch Processing: Instead of sending one URL at a time to the queue, you could batch multiple URLs and send them together to the consumer.

Load Balancing among Consumers: Create a more advanced load-balancing mechanism between consumers based on their current workload.

Caching: Cache results for URLs or words that have already been processed.

HTTP/2: If the server supports it, make sure your HTTP client is configured to use HTTP/2, which could speed up the requests.

Connection Pooling: Reuse HTTP client connections to avoid the overhead of establishing a new connection for each request.

Async IO: Consider using asynchronous IO for network and disk operations.

Quality Enhancements
Error Handling: Add more robust error handling and retries for network failures.

Logging: Add logging at various stages to debug and trace the flow easily.

Metrics: Add metrics to measure queue lengths, processing times, etc., for better observability.

Rate Limiting: Implement rate limiting to avoid hitting rate limits on the server hosting the word list or the URLs you're fetching.

Timeouts: Implement more fine-grained timeout control for different stages of the processing.

Code Modularization: The main function is doing a lot of work. It would be good to break it down into smaller functions to improve readability and maintainability.

Testing: Add unit tests and integration tests.

Documentation: Add comments and documentation to describe the purpose and workings of each function, as well as the overall architecture of your program.

Configurability: Rather than hard-coding settings like timeouts and concurrency limits, make these configurable through a config file or environment variables.

Type Safety: For better type safety and potential performance improvements, consider using slices and arrays instead of maps where the order of elements matters.

Profiling: Use Go's built-in profiling tools to identify bottlenecks and further optimize the code.

Improving both speed and quality usually involves trade-offs, so you'll need to prioritize based on your specific needs.