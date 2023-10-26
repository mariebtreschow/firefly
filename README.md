# firefly assignment

Using go 1.20 version

This application fetches the list of essays [a relative link](/resources/endg-urls) and count the top 10 word for all the essays combined.

A valid word is:
1. Contain at least 3 characters 
2. Contain only alphabetic characters
3. Be part of our [bank of words](https://raw.githubusercontent.com/dwyl/english-words/master/words.txt) (where not all the words in the bank are valid according to the previous rules)

Output is in pretty json and written to the stdout. 

## Build a Docker image and run from the command line

Please see Makefile for available commands 


## Challenges of this application

With such a large workload, it's important to manage system resources efficiently. 

You can do this through:
- Worker Pools: Limit the number of goroutines that are active at any given time.
- Buffered Channels: Use buffered channels to manage the work queue.