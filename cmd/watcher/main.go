package main

import (
	"log"
	"os"
	"strconv"

	"github.com/tmax-cloud/approval-watcher/pkg/watcher"
)

func main() {
	stopChan := make(chan bool)

	// Launch Pod watcher
	go watcher.WatchPods(stopChan)

	// Launch web server
	var port int
	portStr := os.Getenv("APPROVE_PORT")
	if portStr == "" {
		port = watcher.DefaultPort
	} else {
		var err error
		if port, err = strconv.Atoi(portStr); err != nil {
			log.Fatal(err)
		}
	}
	go watcher.LaunchServer(port, watcher.DefaultPath, stopChan)

	success := <-stopChan
	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
