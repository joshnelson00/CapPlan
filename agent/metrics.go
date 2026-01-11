package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	//"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promauto"
)

func shutdownServers(node *exec.Cmd, prometheus *exec.Cmd) {
	if node.Process != nil {
		node.Process.Signal(os.Interrupt)
		node.Wait()
	}
	if prometheus.Process != nil {
		prometheus.Process.Signal(os.Interrupt)
		prometheus.Wait()
	}
}

func main() {
	// Start servers
	nodeServer := exec.Command("../node_exporter/node_exporter")
	prometheusServer := exec.Command(
		"../prometheus/prometheus",
		"--config.file=../prometheus/prometheus.yml",
	)

	nodeServer.Start()
	prometheusServer.Start()
	defer shutdownServers(nodeServer, prometheusServer)

	fmt.Println("Running Local HTTP Server...")

	// HTTP server
	httpServer := &http.Server{Addr: ":2112"}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Println("Hello From Function 1!")
	})

	// Signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Println("\nShutting down...")
		httpServer.Close() // unblocks ListenAndServe
	}()

	// Blocks until Close() is called
	httpServer.ListenAndServe()

	//NEXT STEPS
	// 1. Request from http://localhost:9100/metrics
	// 2. Automate the timing in between (user defined intervals in config or go. RESEARCH)
	// 3. Print all content from request to the screen in formatted text (will handle business logic later)
	// 4. Automate the Download of Node Exporter and Prometheus based on Distro/Architecture/
}
