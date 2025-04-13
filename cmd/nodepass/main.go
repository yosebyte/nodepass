package main

import (
	"net/url"
	"os"
	"runtime"

	"github.com/yosebyte/x/log"
)

var (
	logger  = log.NewLogger(log.Info, true)
	version = "dev"
)

func main() {
	parsedURL := getParsedURL(os.Args)
	initLogLevel(parsedURL.Query().Get("log"))
	coreDispatch(parsedURL)
}

func getParsedURL(args []string) *url.URL {
	if len(args) < 2 {
		getExitInfo()
	}
	parsedURL, err := url.Parse(args[1])
	if err != nil {
		logger.Fatal("URL parse: %v", err)
		getExitInfo()
	}
	return parsedURL
}

func initLogLevel(level string) {
	switch level {
	case "debug":
		logger.SetLogLevel(log.Debug)
		logger.Debug("Init log level: DEBUG")
	case "warn":
		logger.SetLogLevel(log.Warn)
		logger.Warn("Init log level: WARN")
	case "error":
		logger.SetLogLevel(log.Error)
		logger.Error("Init log level: ERROR")
	case "fatal":
		logger.SetLogLevel(log.Fatal)
		logger.Fatal("Init log level: FATAL")
	default:
		logger.SetLogLevel(log.Info)
		logger.Info("Init log level: INFO")
	}
}

func getExitInfo() {
	logger.SetLogLevel(log.Info)
	logger.Info(`Version: %v %v/%v

╭─────────────────────────────────────────────────────────────╮
│             ░░█▀█░█▀█░░▀█░█▀▀░█▀█░█▀█░█▀▀░█▀▀░░             │
│             ░░█░█░█░█░█▀█░█▀▀░█▀▀░█▀█░▀▀█░▀▀█░░             │
│             ░░▀░▀░▀▀▀░▀▀▀░▀▀▀░▀░░░▀░▀░▀▀▀░▀▀▀░░             │
├─────────────────────────────────────────────────────────────┤
│            >Universal TCP/UDP Tunneling Solution            │
│            >https://github.com/yosebyte/nodepass            │
├─────────────────────────────────────────────────────────────┤
│ Usage:                                                      │
│ nodepass <core>://<tunnel>/<target>?<log>&<tls>&<crt>&<key> │
├──────────┬───────────────────────────┬──────────────────────┤
│ Keys     │ Values                    │ Description          │
├──────────┼───────────────────────────┼──────────────────────┤
│ <core>   │ server | client | master  │ Operating mode       │
│ <tunnel> │ host:port (IP | domain)   │ Tunnel address       │
│ <target> │ host:port | API prefix    │ Target addr | prefix │
│ <log>    │ debug | info | warn | ... │ Default level info   │
│ * <tls>  │ 0 off | 1 on | 2 verify   │ Default TLS code-0   │
│ * <crt>  │ <path/to/crt.pem>         │ Custom certificate   │
│ * <key>  │ <path/to/key.pem>         │ Custom private key   │
├──────────┴───────────────────────────┴──────────────────────┤
│ * Vaild for server and master mode only                     │
╰─────────────────────────────────────────────────────────────╯
`, version, runtime.GOOS, runtime.GOARCH)
	os.Exit(1)
}
