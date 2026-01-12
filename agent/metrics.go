package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type MetricSample struct {
	Name      string
	Labels    map[string]string
	Value     float64
	Timestamp time.Time
}

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

func getMetricList() {
	file, err := os.Open("tracked-metrics.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		metricList = append(metricList, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func cleanAndStoreMetrics(result model.Value) error {
	vector, ok := result.(model.Vector)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", result)
	}

	for _, s := range vector {
		labels := make(map[string]string)
		for k, v := range s.Metric {
			labels[string(k)] = string(v)
		}

		sample := MetricSample{
			Name:      labels["__name__"],
			Labels:    labels,
			Value:     float64(s.Value),
			Timestamp: s.Timestamp.Time(),
		}

		samples = append(samples, sample)
	}

	return nil
}

func getMetrics(api v1.API, ctx context.Context) {
	for i := 0; i < len(metricList); i++ {
		query := metricList[i]
		result, warnings, err := api.Query(ctx, query, time.Now())
		if err != nil {
			log.Fatalf("Error querying Prometheus: %v", err)
		}
		if len(warnings) > 0 {
			log.Printf("Warnings: %v\n", warnings)
		}

		if err := cleanAndStoreMetrics(result); err != nil {
			log.Fatalf("Failed to clean/store samples: %v", err)
		}
	}
}

var samples []MetricSample
var metricList []string

func main() {
	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start servers
	nodeServer := exec.Command("../node_exporter/node_exporter")
	prometheusServer := exec.Command(
		"../prometheus/prometheus",
		"--config.file=../prometheus/prometheus.yml",
	)
	nodeServer.Start()
	prometheusServer.Start()
	time.Sleep(5 * time.Second)
	defer shutdownServers(nodeServer, prometheusServer)

	// Get Metrics
	getMetricList()
	getMetrics(v1api, ctx)

	// Signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	<-sig
	fmt.Println("\nShutting down...")
}
