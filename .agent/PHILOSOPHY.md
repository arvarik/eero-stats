# Product Philosophy & Vision

## 1. Core Purpose
`eero-stats` is designed to extract real-time metrics from an Eero Mesh Network and visualize them in Grafana, filling the gap of advanced telemetry missing from the consumer Eero app.

## 2. Target Persona
Homelab enthusiasts, NAS operators (TrueNAS, Unraid, Proxmox), and network administrators who want deep visibility into their home network performance without relying solely on cloud dashboards.

## 3. Core Beliefs & Principles
1.  **Minimal Hardware Wear:** NAS systems often run on NVMe or USB flash drives. Aggressive memory batching is non-negotiable to prevent premature hardware failure from continuous metric writes.
2.  **API Etiquette:** The Eero cloud API is undocumented and rate-limited. Tiered polling (Fast/Medium/Slow) balances data freshness with API safety.
3.  **Resilience:** The daemon must survive network partitions and API rate limits via exponential backoff and localized retries without crashing.
4.  **Zero-Configuration Dashboards:** The Grafana dashboard should be fully provisioned and ready to use immediately upon container start, requiring no manual metric mapping by the user.

## 4. What This Is NOT (Anti-Goals)
*   It is **NOT** a tool to modify network settings or control Eero devices (read-only telemetry).
*   It is **NOT** designed for direct, real-time streaming (metrics are batched and delayed slightly to optimize writes).
