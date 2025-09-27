package main

import "os"

var version = "dev"

func main() {
	if err := start(os.Args); err != nil {
		exit(err)
	}
}
