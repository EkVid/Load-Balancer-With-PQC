# Reverse Proxy and Load Balancer (C++) — Project Draft

## Overview
This project aims to design and implement a **reverse proxy and load balancer** using **C++**. The system will accept HTTP client requests, route them to backend servers using configurable load balancing algorithms, and return responses. It emphasizes performance, concurrency, fault tolerance, and observability.

---

## Objectives
- Implement a configurable, high-performance reverse proxy.
- Support multiple load balancing strategies.
- Ensure backend health monitoring and automatic failover.
- Implement observability features (logging, metrics, configuration reload).
- Optionally, provide advanced features like TLS termination or sticky sessions.

---

## Core Features
1. **TCP Listener & HTTP Handling**
   - Accept HTTP/1.1 requests.
   - Handle connection reuse (keep-alive).
   - Stream responses back to clients.

2. **Load Balancing**
   - Round Robin and Least Connections algorithms.
   - Extendable load balancing strategy interface.

3. **Backend Health Checks**
   - Periodic health probes (HTTP/TCP).
   - Mark backends up/down based on probe results.

4. **Concurrency Model**
   - Multi-threaded or asynchronous event-driven design.
   - Efficient request handling and I/O operations.

5. **Configuration & Logging**
   - External configuration file (JSON/TOML).
   - Structured logging for requests and health status.

6. **Graceful Shutdown & Config Reload**
   - Signal handling (SIGHUP) for dynamic config reload.
   - Clean connection termination on shutdown.

---

## Stretch Features (Optional)
- **TLS Termination** using OpenSSL.
- **Sticky Sessions** (cookie-based or IP-hash).
- **Weighted Load Balancing**.
- **Rate Limiting** per client.
- **Web UI or Metrics Endpoint** (e.g., Prometheus format).

---

## Team Roles & Responsibilities

### **Person A — Core Networking & Proxy Engine**
**Responsibilities:**
- Implement TCP acceptor and client connection handler.
- Handle HTTP request/response parsing and streaming.
- Manage client-backend connections and data forwarding.
- Implement concurrency (Boost.Asio or epoll).

**Challenges:**
- Efficiently handle concurrent client connections.
- Manage non-blocking I/O and avoid race conditions.
- Correctly parse and reconstruct HTTP traffic (headers, chunked encoding).

---

### **Person B — Load Balancer & Backend Management**
**Responsibilities:**
- Implement load balancing algorithms (Round Robin, Least Connections).
- Manage backend pool and connection counts.
- Develop health check mechanism and backend state transitions.
- Implement configuration loader and dynamic reloading.

**Challenges:**
- Designing a robust backend state machine (UP/DOWN/PROBING).
- Balancing fairness and performance in load balancing.
- Handling backend recovery and failure gracefully.

---

### **Person C — Security, Observability, and Testing**
**Responsibilities:**
- Implement TLS termination (optional).
- Add structured logging and runtime metrics collection.
- Create a `/status` or `/metrics` endpoint for monitoring.
- Design unit tests and integration test harness.
- Build demo and benchmark scripts.

**Challenges:**
- Integrating OpenSSL with asynchronous sockets.
- Designing test cases for concurrency and fault tolerance.
- Building meaningful metrics (latency, throughput, error rate).

---

## Implementation Plan (4 Weeks)

### **Week 1 — Design & Setup**
- Define architecture and modules.
- Create repository and CMake setup.
- Implement simple echo proxy and Round Robin stub.

### **Week 2 — Core Proxy and Load Balancing**
- Develop HTTP forwarding and backend communication.
- Implement Least Connections algorithm.
- Integrate config loader and basic logging.

### **Week 3 — Health Checks & Observability**
- Implement backend health checks.
- Add metrics and structured logs.
- Begin integration and performance testing.

### **Week 4 — Stretch Features & Finalization**
- Implement TLS or sticky sessions (optional).
- Add graceful shutdown and config reload.
- Final debugging, documentation, and demo preparation.

---

## Evaluation Criteria
- Correctness of HTTP proxying and response handling.
- Functional load balancing and backend failover.
- Thread-safety and concurrency performance.
- Quality of logs, metrics, and observability features.
- Code structure, documentation, and testing coverage.

---

## Expected Deliverables
- Source code with clear directory structure and CMake build.
- Sample configuration files.
- Documentation (README and report).
- Automated test scripts and demo instructions.
- Benchmark and metrics results.

---

## Summary
This project combines **network programming**, **system concurrency**, and **distributed systems design**. It demonstrates practical skills in C++ networking, fault tolerance, and performance optimization while being complex enough to challenge a 3-person team over one month.
