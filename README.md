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

run  with optional flags:
- timeout
- global_timeout
- word_bank_url
- essays_path
- concurrency_limit
- num_consumers

If you wanna run with these flags easiest is to update the .env file 
and include it when you run command can be found in Makefile.

`make wordcounter`

## How to run the docker

`make docker-run`

Switch out tag for the tag you built with

## Run tests

`make tests`









