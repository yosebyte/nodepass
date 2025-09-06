package main

import (
	"net/url"
	"os"
	"runtime"

	"github.com/NodePassProject/logs"
)

var (
	// 全局日志记录器
	logger = logs.NewLogger(logs.Info, true)
	// 程序版本
	version = "dev"
)

// main 程序入口
func main() {
	parsedURL := getParsedURL(os.Args)
	initLogLevel(parsedURL.Query().Get("log"))
	coreDispatch(parsedURL)
}

// getParsedURL 解析URL参数
func getParsedURL(args []string) *url.URL {
	if len(args) < 2 {
		getExitInfo()
	}

	parsedURL, err := url.Parse(args[1])
	if err != nil {
		logger.Error("URL parse: %v", err)
		getExitInfo()
	}

	return parsedURL
}

// initLogLevel 初始化日志级别
func initLogLevel(level string) {
	switch level {
	case "none":
		logger.SetLogLevel(logs.None)
	case "debug":
		logger.SetLogLevel(logs.Debug)
		logger.Debug("Init log level: DEBUG")
	case "warn":
		logger.SetLogLevel(logs.Warn)
		logger.Warn("Init log level: WARN")
	case "error":
		logger.SetLogLevel(logs.Error)
		logger.Error("Init log level: ERROR")
	case "event":
		logger.SetLogLevel(logs.Event)
		logger.Event("Init log level: EVENT")
	default:
		logger.SetLogLevel(logs.Info)
		logger.Info("Init log level: INFO")
	}
}

// getExitInfo 输出帮助信息并退出程序
func getExitInfo() {
	logger.SetLogLevel(logs.Info)
	logger.Info(`Version: %v %v/%v

╭─────────────────────────────────────────────╮
│     ░░█▀█░█▀█░░▀█░█▀▀░█▀█░█▀█░█▀▀░█▀▀░░     │
│     ░░█░█░█░█░█▀█░█▀▀░█▀▀░█▀█░▀▀█░▀▀█░░     │
│     ░░▀░▀░▀▀▀░▀▀▀░▀▀▀░▀░░░▀░▀░▀▀▀░▀▀▀░░     │
├─────────────────────────────────────────────┤
│    >Universal TCP/UDP Tunneling Solution    │
│    >https://github.com/yosebyte/nodepass    │
├─────────────────────────────────────────────┤
│ Usage: nodepass "<your-unique-URL-command>" │
├─────────────────────────────────────────────┤
│ server://password@host/host?<query>&<query> │
│ client://password@host/host?<query>&<query> │
│ master://hostname:port/path?<query>&<query> │
╰─────────────────────────────────────────────╯
`, version, runtime.GOOS, runtime.GOARCH)
	os.Exit(1)
}
