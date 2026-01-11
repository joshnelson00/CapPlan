# Overview

CapPlan is a system-agnostic Site Reliability Engineering (SRE) capacity planning platform designed to forecast future infrastructure demand, model realistic failure scenarios, and recommend cost-efficient capacity configurations.

Unlike platform-specific autoscaling tools, CapPlan operates at the host and service level, making it applicable to virtual machines, bare-metal servers, containers, cloud environments, on-prem deployments, and edge systems.

The project focuses on predictive planning rather than reactive scaling, enabling teams to answer the question:

**How much capacity do we need in the future to remain reliable, and what is the cheapest way to achieve it?**

## Project Goals

CapPlan is designed to:

- Forecast future resource demand using historical telemetry
- Identify peak usage and long-term growth trends
- Model realistic infrastructure failure scenarios
- Quantify required headroom to maintain reliability
- Optimize infrastructure capacity for cost and resilience
- Operate independently of any orchestration platform

This project emphasizes SRE fundamentals over platform-specific abstractions.

## Scope and Non-Goals

### In Scope

- Host-level and service-level capacity planning
- Time-series forecasting of resource usage
- Failure-aware capacity modeling
- Cost-aware optimization strategies
- Portable metric collection and ingestion

### Out of Scope

- Real-time autoscaling
- Kubernetes-specific logic
- Application-level business logic
- Full production observability stacks

CapPlan is a planning and decision-support system, not a runtime controller.

## High-Level Architecture

```
┌────────────────────┐
│   Target Systems   │
│ (VMs / Bare Metal) │
│  Containers / Edge │
└─────────┬──────────┘
          │ telemetry
┌─────────▼──────────┐
│ Lightweight Agent  │
│ (Metrics Collector)│
└─────────┬──────────┘
          │ batched metrics
┌─────────▼──────────┐
│ Ingest API         │
│ (Normalization)    │
└─────────┬──────────┘
          │ time-series data
┌─────────▼──────────┐
│ Forecast Engine    │
│ (Predictive Models)│
└─────────┬──────────┘
          │ demand estimates
┌─────────▼──────────┐
│ Capacity Optimizer │
│ (Failures + Cost)  │
└─────────┬──────────┘
          │ recommendations
┌─────────▼──────────┐
│ CLI / Reports      │
│ (Decision Output)  │
└────────────────────┘
```

## System Pipeline

### 1. Metric Collection

A lightweight, portable agent runs on each target system and periodically collects infrastructure metrics such as:

- CPU utilization
- Memory usage
- Disk I/O
- Network throughput
- Optional service-level metrics (e.g., request rate)

The agent is designed to be:

- Low overhead
- Platform independent
- Safe to run on production-like systems

### 2. Metric Ingestion and Normalization

Collected metrics are sent to a centralized ingestion service, where they are:

- Validated and normalized
- Grouped by host, service, and failure domain
- Stored as structured time-series data

Normalization ensures consistent analysis across heterogeneous environments.

### 3. Demand Forecasting

Historical metrics are analyzed to predict future resource demand over configurable horizons (e.g., 7, 14, or 30 days).

The forecasting process accounts for:

- Daily and weekly seasonality
- Long-term growth trends
- Short-term bursts and spikes
- Noise inherent in real systems

Outputs include:

- Mean expected demand
- High-percentile demand (p95 / p99)
- Confidence intervals

### 4. Failure-Aware Capacity Modeling

CapPlan simulates realistic failure scenarios to determine required capacity under degraded conditions, such as:

- Loss of a single host
- Loss of a percentage of the fleet
- Rack or availability zone failure
- Sudden traffic surges

Capacity requirements are recalculated assuming failures occur during peak demand.

This step ensures recommendations are reliability-aware, not optimistic.

### 5. Cost-Aware Optimization

Given predicted demand and failure-adjusted requirements, CapPlan evaluates different capacity configurations by considering:

- Horizontal vs vertical scaling strategies
- Machine sizing tradeoffs
- Overhead and headroom requirements
- Infrastructure pricing models

The optimizer selects the lowest-cost configuration that satisfies reliability constraints.

### 6. Recommendation Output

Final recommendations are presented through a CLI and report formats, including:

- Required total capacity
- Recommended host count and sizing
- Expected headroom
- Failure tolerance summary
- Estimated infrastructure cost

Outputs are designed to be:

- Human-readable
- Machine-parsable
- Auditable and explainable

## Data Strategy

Due to limited availability of public production infrastructure telemetry, CapPlan uses a hybrid data approach:

- Real metrics collected from controlled environments
- Synthetic data extensions to model long-term growth
- Injected failure and spike scenarios for robustness testing

All assumptions and data generation methods are documented and reproducible.

## Evaluation Criteria

CapPlan is evaluated based on:

- Forecast accuracy (e.g., RMSE, MAPE)
- Ability to anticipate peak demand
- Correct handling of failure scenarios
- Cost efficiency of recommendations
- Clarity and explainability of outputs

## Intended Audience

- Site Reliability Engineers
- Infrastructure Engineers
- Platform Engineers
- Students studying distributed systems and reliability

## Summary

CapPlan demonstrates how predictive modeling, failure analysis, and cost optimization can be combined into a practical, system-agnostic capacity planning platform.

The project emphasizes SRE thinking, engineering tradeoffs, and production realism, making it suitable as both a capstone project and a foundation for real-world infrastructure planning tools.
