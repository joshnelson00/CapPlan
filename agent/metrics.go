package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joshnelson00/CapPlan/database"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func waitForPrometheus(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	readyURL := url + "/-/ready"
	fmt.Printf("⏳ Waiting for Prometheus at %s ", readyURL)
	for time.Now().Before(deadline) {
		resp, err := http.Get(readyURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Println(" ready!")
				return nil
			}
		}
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out after %s waiting for Prometheus at %s", timeout, readyURL)
}

func waitForFirstScrape(promAPI v1.API, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	fmt.Print("⏳ Waiting for first Prometheus scrape ")
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		targets, err := promAPI.Targets(ctx)
		cancel()
		if err == nil && len(targets.Active) > 0 {
			allScraped := true
			for _, t := range targets.Active {
				if t.LastScrape.IsZero() || t.Health != v1.HealthGood {
					allScraped = false
					break
				}
			}
			if allScraped {
				fmt.Println(" done!")
				return nil
			}
		}
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out after %s waiting for first scrape", timeout)
}

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
	interval := flag.Duration("interval", 5*time.Minute, "How often to collect metrics (e.g. 30s, 5m, 1h)")
	flag.Parse()

	fmt.Println("\n╔════════════════════════════════════╗")
	fmt.Println("║       CapPlan Metrics Agent        ║")
	fmt.Println("║    Prometheus → PostgreSQL         ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Printf("  Collection interval: %s\n", *interval)

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
	// Kill any stale processes left from a previous run
	for _, port := range []string{"9090", "9100"} {
		if out, err := exec.Command("fuser", "-k", port+"/tcp").CombinedOutput(); err == nil {
			fmt.Printf("⚠ Killed stale process on port %s: %s\n", port, string(out))
			time.Sleep(500 * time.Millisecond)
		}
	}

	nodeServer := exec.Command("../node_exporter/node_exporter")
	nodeServer.Stderr = os.Stderr
	prometheusServer := exec.Command(
		"../prometheus/prometheus",
		"--config.file=../prometheus/prometheus.yml",
	)
	prometheusServer.Stderr = os.Stderr
	if err := nodeServer.Start(); err != nil {
		log.Fatalf("Failed to start Node Exporter: %v", err)
	}
	if err := prometheusServer.Start(); err != nil {
		log.Fatalf("Failed to start Prometheus: %v", err)
	}
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

	// Wait for Prometheus to be ready (up to 60 seconds)
	if err := waitForPrometheus("http://localhost:9090", 60*time.Second); err != nil {
		log.Fatalf("Prometheus not ready: %v", err)
	}
	if err := waitForFirstScrape(v1api, 60*time.Second); err != nil {
		log.Fatalf("Scrape never completed: %v", err)
	}
	fmt.Println("✓ Servers ready")

	// Load metric list once
	getMetricList()

	// Signal handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	collectAndStore := func() {
		samples = nil // reset each cycle
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		getMetrics(v1api, ctx)
		// Import metrics
		fmt.Println("\n┌────────────────────────────────────┐")
		fmt.Println("│  Importing to Database...          │")
		fmt.Println("└────────────────────────────────────┘")
		if err := database.ImportMetricSamples(db, samples); err != nil {
			log.Printf("Warning: Failed to import samples: %v", err)
		} else {
			fmt.Printf("\n╔════════════════════════════════════╗\n")
			fmt.Printf("║  ✓ Successfully imported %4d      ║\n", len(samples))
			fmt.Printf("║    samples to PostgreSQL!          ║\n")
			fmt.Printf("╚════════════════════════════════════╝\n")
		}
	}

	// Collect immediately on startup
	collectAndStore()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	fmt.Printf("\n⌨  Next collection in %s. Press Ctrl+C to stop...\n", *interval)
	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n⏰ Collecting metrics (interval: %s)...\n", *interval)
			collectAndStore()
			fmt.Printf("\n⌨  Next collection in %s. Press Ctrl+C to stop...\n", *interval)
		case <-sig:
			fmt.Println("\n\n╔════════════════════════════════════╗")
			fmt.Println("║      Graceful Shutdown...          ║")
			fmt.Println("╚════════════════════════════════════╝")
			return
		}
	}
}
