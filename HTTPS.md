# HTTPS Implementation with Post-Quantum Cryptography — Project Draft

## Overview
This project will design and implement a **HTTPS protocol** in **C++**, including **post-quantum cryptography (PQC)** for key exchange and encryption. The system will allow secure client-server communication over TCP, supporting TLS-like handshake, encryption/decryption, and certificate validation. The project emphasizes security, cryptography, performance, and concurrency.

---

## Objectives

### Core Objectives
- **TCP Secure Layer:**
  - Implement TCP-based client-server communication through sockets.
- **TLS-like Handshake:**
  - Implement a handshake protocol similar to TLS 1.3.
  - Support certificate exchange, authentication, and key derivation.
- **Post-Quantum Cryptography:**
  - Integrate PQC algorithms (e.g., Kyber, Dilithium) for key exchange and digital signatures.
  - Provide hybrid PQC + classical algorithms for backward compatibility.
- **Symmetric Encryption:**
  - Implement authenticated encryption (e.g., AES-GCM, ChaCha20-Poly1305) for session traffic.
- **Configuration Management:**
  - Load server configuration, supported algorithms, and keys from a configuration file.
- **Concurrency & Performance:**
  - Use asynchronous I/O or multi-threaded design for handling multiple concurrent secure sessions.
- **Observability & Logging:**
  - Log handshake details, encryption usage, and session statistics.
- **Client Connectivity to HTTPS
  - Client should be able to connect to all HTTPS website on the web, not just our server.
  - Client shoud be able to authenticate the certificates sent by the server using Certificate Authority (CA)

### Advanced Features
- Post-quantum certificate validation.
- Session resumption (like TLS session tickets).
- Secure renegotiation for long-lived connections.
---

## Team Roles & Responsibilities

### **Core Networking & Handshake Engine**
**Responsibilities:**
- Implement TCP acceptor and client connection handler.
- Handle TLS-like handshake protocol with proper message ordering.
- Integrate session key derivation and symmetric encryption initialization.
- Implement concurrency (Boost.Asio or epoll).

**Challenges that could come up:**
- Handling partial reads/writes and message framing correctly.
- Designing handshake state machine to prevent replay or downgrade attacks.
- Ensuring non-blocking TLS/PQC handshake.

---

### **Post-Quantum Cryptography & Key Management**
**Responsibilities:**
- Implement PQC algorithms for key exchange (e.g., Kyber) and digital signatures (e.g., Dilithium).
- Support hybrid classical + PQC key exchange.
- Manage session keys, ephemeral keys, and certificate verification.
- Integrate PQC operations into handshake protocol securely.

**Challenges that could come up:**
- Understanding and correctly implementing PQC algorithms.
- Preventing side-channel attacks during key operations.
- Combining classical and PQC cryptography in hybrid mode securely.

---

### **Observability, Security, & Reliability**
**Responsibilities:**
- Add structured logging for handshake, session establishment, and encryption events.
- Implement runtime metrics for active secure sessions and handshake latencies.
- Support graceful shutdown and cleanup of secure connections.
- Implement optional session resumption and renegotiation.
- Ensure correct error handling for handshake failures or invalid certificates.

**Challenges that could come up:**
- Maintaining performance under multiple concurrent secure sessions.
- Designing meaningful metrics without leaking sensitive information.
- Handling dynamic session state while ensuring security properties.
- Correctly logging and monitoring without introducing security risks.

---
