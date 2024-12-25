package main

import (
	"runtime"

	"github.com/yosebyte/x/log"
)

func helpInfo() {
	log.Info(`Version: %v %v/%v

Usage:
    nodepass <core_mode>://<server_addr>/<target_addr>

Examples:
    # Run as server
    nodepass server://10.0.0.1:10101/10.0.0.1:18080

    # Run as client
    nodepass client://10.0.0.1:10101/127.0.0.1:8080

Arguments:
    <core_mode>    Select between "server" or "client"
    <server_addr>  Server address to listen or connect
    <target_addr>  Target address to expose or forward
`, version, runtime.GOOS, runtime.GOARCH)
}
