package main

import (
	"github.com/tmax-cloud/approval-watcher/pkg/apiserver"
	"log"
	"os"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/tmax-cloud/approval-watcher/pkg/server"
	"github.com/tmax-cloud/approval-watcher/pkg/watcher"
)

func main() {
	logf.SetLogger(zap.Logger())
	stopChan := make(chan bool)

	// Launch Pod watcher - Restart when watcher dies
	go func() {
		for {
			watcher.WatchPods(stopChan)
		}
	}()

	// Launch web server
	var port int
	portStr := os.Getenv("APPROVE_PORT")
	if portStr == "" {
		port = server.DefaultPort
	} else {
		var err error
		if port, err = strconv.Atoi(portStr); err != nil {
			log.Fatal(err)
		}
	}
	// DEPRECATED : Use api-aggregation
	go server.LaunchServer(port, server.DefaultPath, stopChan)

	// API-aggregation
	apiServer := apiserver.New()
	go apiServer.Start()

	success := <-stopChan
	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
