# Reverse Proxy and Load Balancer (C++) — Project Draft

## Overview
This project will design and implement a **reverse proxy and load balancer** using **C++**. The system will accept HTTP client requests, route them to backend servers using configurable load balancing algorithms, and return responses. It emphasizes performance, concurrency, fault tolerance, and observability.

---

### Objectives
- **Reverse Proxy:**
  - Accept and forward client HTTP requests to backend servers transparently.
  - Maintain correct HTTP semantics and response streaming.
- **Load Balancer:**
  - Implement a load balancing algorithms based on existing ones such as **Round Robin** and **Least Connections**.
  - Dynamically select the best backend based on current load and availability.
  - Support failover when a backend becomes unavailable.
- **Health Monitoring:**
  - Periodically check backend health via HTTP or TCP probes.
  - Automatically mark backends as UP/DOWN based on probe results.
- **Concurrency & Performance:**
  - Use asynchronous I/O or a thread pool for efficient concurrent request handling.
  - Support connection reuse (keep-alive) and timeout management.
- **Configuration Management:**
  - Load configuration (e.g., backends, ports, strategies) from an external JSON file.
  - Support runtime configuration reload without restarting the proxy.
---

### (Optional Objectives)
- **Reliability & Resilience:**
  - Implement retry and failover logic when a backend fails.
  - Ensure graceful shutdown and recovery mechanisms
- **Observability & Logging:**
  - Implement structured logging for proxy and load balancer activity.
  - Expose performance metrics such as throughput, latency, error rates, and backend health.
---

### Advanced Features
- **TLS Termination** using OpenSSL for HTTPS support.
- **Sticky Sessions** based on client IP or cookies.
- **Weighted Load Balancing** or slow-start for recovering backends.
- **Rate Limiting** per client to prevent overload.
- **/metrics or /status Endpoint** for real-time monitoring (Prometheus or JSON format).
---

## Team Roles & Responsibilities

### **Core Networking & Proxy Engine**
**Responsibilities:**
- Implement TCP acceptor and client connection handler.
- Handle HTTP request/response parsing and streaming.
- Manage client-backend connections and data forwarding.
- Implement concurrency (Boost.Asio or epoll).
- Implement **TLS termination** (OpenSSL) for secure client connections.

**Challenges that could come up:**
- Efficiently handle concurrent client connections.
- Manage non-blocking I/O and avoid race conditions.
- Correctly parse and reconstruct HTTP traffic (headers, chunked encoding).
- Integrating OpenSSL with asynchronous sockets safely.

---

### **Load Balancer & Backend Management**
**Responsibilities:**
- Implement load balancing algorithms: Round Robin, Least Connections, and optionally weighted.
- Manage backend pool, active connection counts, and health check states.
- Implement sticky sessions (cookie-based or IP-hash) to support session affinity.
- Implement configuration loader and dynamic reloading.

**Challenges that could come up:**
- Designing a robust backend state machine (UP/DOWN/PROBING).
- Balancing fairness and performance in load balancing.
- Handling backend recovery and failover gracefully.
- Implementing weighted and sticky load balancing correctly.

---

### **Observability, Metrics & Reliability**
**Responsibilities:**
- Add structured logging for proxy and load balancer activity.
- Implement runtime metrics collection (throughput, latency, error rates, backend health).
- Create `/status` or `/metrics` endpoint for monitoring.
- Implement rate limiting per client to prevent overload.
- Support graceful shutdown and safe cleanup of connections.

**Challenges that could come up:**
- Designing test cases for concurrency, fault tolerance, and load balancing correctness.
- Ensuring metrics collection is accurate without significantly impacting performance.
- Handling dynamic configuration reload and shutdown without dropping active connections.
- Designing meaningful observability and monitoring for system performance.

---
