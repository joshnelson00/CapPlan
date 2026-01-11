package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime/metrics"
)

func main() {
	metricsList, err := loadMetrics()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("┌────────────────────┐")
	fmt.Println("│  Tracked Metrics   │")
	fmt.Println("└────────────────────┘")
	for i := 0; i < len(metricsList); i++ {
		fmt.Printf("%d. %s\n", i+1, metricsList[i])
	}

	// NEXT STEPS
	// 1. Track Metrics from "metricsList" slice
	// 2. Return and store metrics in Structs/Formatted JSON/etc.
}

func loadMetrics() ([]string, error) {
	var metricsSlice []string

	file, err := os.Open("tracked-metrics.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		metricsSlice = append(metricsSlice, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Name of the metric we want to read.
	const myMetric = "/cpu/classes/total:cpu-seconds"
	// Create a sample for the metric.
	sample := make([]metrics.Sample, 1)
	sample[0].Name = myMetric

	// Sample the metric.
	metrics.Read(sample)

	// Check if the metric is actually supported.
	// If it's not, the resulting value will always have
	// kind KindBad.
	if sample[0].Value.Kind() == metrics.KindBad {
		panic(fmt.Sprintf("metric %q no longer supported", myMetric))
	}

	// Handle the result.
	//
	// It's OK to assume a particular Kind for a metric;
	// they're guaranteed not to change.
	cpuSecs := sample[0].Value
	fmt.Printf("Seconds CPU spent executing program: %d\n", cpuSecs)

	return metricsSlice, err
}
