package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
)

var version = "dev"

// main 程序入口
func main() {
	runCore(getParsedURL(os.Args))
}

// getParsedURL 解析URL参数
func getParsedURL(args []string) *url.URL {
	if len(args) != 2 {
		printExitInfo()
	}
	parsedURL, err := url.Parse(args[1])
	if err != nil {
		printExitInfo()
	}
	return parsedURL
}

// printExitInfo 打印退出信息
func printExitInfo() {
	fmt.Printf(`
╭─────────────────────────────────────╮
│ ░░█▀█░█▀█░░▀█░█▀▀░█▀█░█▀█░█▀▀░█▀▀░░ │
│ ░░█░█░█░█░█▀█░█▀▀░█▀▀░█▀█░▀▀█░▀▀█░░ │
│ ░░▀░▀░▀▀▀░▀▀▀░▀▀▀░▀░░░▀░▀░▀▀▀░▀▀▀░░ │
├─────────────────────────────────────┤
│%*s │
│%*s │
├─────────────────────────────────────┤
│ server://password@host/host?<query> │
│ client://password@host/host?<query> │
│ master://hostname:port/path?<query> │
╰─────────────────────────────────────╯

`, 36, version, 36, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	os.Exit(1)
}
