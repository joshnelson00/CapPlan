package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/joshnelson00/CapPlan/database"
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

func shutdownServers(node *exec.Cmd, prometheus *exec.Cmd) {
	fmt.Println("\n╔════════════════════════════════════╗")
	fmt.Println("║   Shutting Down Servers...        ║")
	fmt.Println("╚════════════════════════════════════╝")
	if node.Process != nil {
		node.Process.Signal(os.Interrupt)
		node.Wait()
	}
	if prometheus.Process != nil {
		prometheus.Process.Signal(os.Interrupt)
		prometheus.Wait()
	}
	fmt.Println("✓ Servers stopped successfully")
}
func getMetricList() {
	fmt.Println("\n┌────────────────────────────────────┐")
	fmt.Println("│  Loading Metric List...           │")
	fmt.Println("└────────────────────────────────────┘")
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
	fmt.Printf("✓ Loaded %d metrics to track\n", len(metricList))
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
		sample := database.MetricSample{
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
	fmt.Println("\n┌────────────────────────────────────┐")
	fmt.Println("│  Querying Prometheus...            │")
	fmt.Println("└────────────────────────────────────┘")
	for i := 0; i < len(metricList); i++ {
		query := metricList[i]
		fmt.Printf("  → Querying: %s\n", query)
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
	fmt.Printf("✓ Collected %d metric samples\n", len(samples))
}
func getDBConfig() database.DatabaseConfig {
	fmt.Println("\n┌────────────────────────────────────┐")
	fmt.Println("│  Loading Database Config...        │")
	fmt.Println("└────────────────────────────────────┘")
	configFile, err := os.ReadFile("../config/db.config")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	var dbConfig database.DatabaseConfig
	if err := json.Unmarshal(configFile, &dbConfig); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	fmt.Println("✓ Database config loaded")
	return dbConfig
}

var samples []database.MetricSample
var metricList []string
var db *database.Database

func main() {
	fmt.Println("\n╔════════════════════════════════════╗")
	fmt.Println("║       CapPlan Metrics Agent        ║")
	fmt.Println("║    Prometheus → PostgreSQL         ║")
	fmt.Println("╚════════════════════════════════════╝")

	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	v1api := v1.NewAPI(client)
	// Start Prometheus servers
	fmt.Println("\n┌────────────────────────────────────┐")
	fmt.Println("│  Starting Prometheus Servers...    │")
	fmt.Println("└────────────────────────────────────┘")
	nodeServer := exec.Command("../node_exporter/node_exporter")
	prometheusServer := exec.Command(
		"../prometheus/prometheus",
		"--config.file=../prometheus/prometheus.yml",
	)
	nodeServer.Start()
	prometheusServer.Start()
	defer shutdownServers(nodeServer, prometheusServer)
	fmt.Println("✓ Node Exporter started")
	fmt.Println("✓ Prometheus started")

	// Start DB
	dbConfig := getDBConfig()
	db, err = database.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)
	fmt.Printf("✓ Connected to PostgreSQL at %s:%d\n", dbConfig.Host, dbConfig.Port)

	// Startup Server Time
	startupTime := 30
	fmt.Printf("\n⏳ Waiting %d seconds for servers to initialize...\n", startupTime)
	time.Sleep(time.Duration(startupTime) * time.Second)
	fmt.Println("✓ Servers ready")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Get Metrics
	getMetricList()
	getMetrics(v1api, ctx)
	// Import metrics
	fmt.Println("\n┌────────────────────────────────────┐")
	fmt.Println("│  Importing to Database...          │")
	fmt.Println("└────────────────────────────────────┘")
	if err := database.ImportMetricSamples(db, samples); err != nil {
		log.Fatalf("Failed to import samples: %v", err)
	}
	fmt.Printf("\n╔════════════════════════════════════╗\n")
	fmt.Printf("║  ✓ Successfully imported %4d      ║\n", len(samples))
	fmt.Printf("║    samples to PostgreSQL!          ║\n")
	fmt.Printf("╚════════════════════════════════════╝\n")

	// Signal handling
	fmt.Println("\n⌨  Press Ctrl+C to stop...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\n\n╔════════════════════════════════════╗")
	fmt.Println("║      Graceful Shutdown...          ║")
	fmt.Println("╚════════════════════════════════════╝")
}
