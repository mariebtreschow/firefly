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

Install dependencies

`go mod download`

Run locally with default flags

`go run cmd/word-count/main.go`

## How to run the docker

`docker run -e TIMEOUT=60 -e GLOABL_TIMEOUT=120 -e CONCURRENCY_LIMIT=40 -e NUM_CONSUMERS=10 -d wordcounter`

optional flags:

- timeout
- global_timeout
- word_bank_url
- essays_path
- concurrency_limit
- num_consumers

## Run tests

From root folder:

`go test firefly.ai/word-counter/cmd/word-count`









