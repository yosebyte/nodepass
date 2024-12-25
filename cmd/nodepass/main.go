package main

import (
	"net/url"
	"os"

	"github.com/yosebyte/x/log"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		helpInfo()
		os.Exit(1)
	}
	rawURL := os.Args[1]
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Fatal("Unable to parse raw URL: %v", err)
	}
	coreSelect(parsedURL)
}
