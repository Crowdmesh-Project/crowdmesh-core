# CrowdMesh Core

Deterministic simulation engine and embeddable, zero-I/O protocol core for infrastructure-less, connectionless ad-hoc mesh networking in high-density environments.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Project Overview

CrowdMesh is an applied research and engineering project designed to address communication in dense, infrastructure-deprived environments (aircraft cabins, stadiums, off-grid locations) using connectionless broadcasts (Bluetooth Low Energy Advertising, Wi-Fi Aware).

Key architectural goals:
1. **Zero-I/O State Machine:** The core protocol engine does not touch sockets, system clocks (`time.Now()`), or persistent storage. It consumes discrete incoming events (`Event`) and produces a list of deterministic actions (`Action`).
2. **Deterministic Discrete-Event Simulation:** The exact same configuration and seed yield bit-for-bit identical network metrics and execution traces.
3. **Embeddable in Mobile Platforms:** The same core protocol logic can be compiled via `gomobile` or C-ABI bindings directly into Android (Kotlin) and iOS (Swift) applications.
4. **Broadcast Storm Mitigation:** Counter-based suppression (inspired by RFC 6206 / Trickle) and probabilistic gossip to reduce redundant transmissions by 60-80% compared to baseline flooding.
5. **Eventual Consistency without Central Authority:** Messages are stored in a Grow-only Set (G-Set) CRDT ordered by Hybrid Logical Clocks (HLC) and synchronized via periodic anti-entropy digests.

Detailed conceptual documentation and mathematical propagation models are available in [docs/concept.md](docs/concept.md).

---

## Repository Structure

```
crowdmesh-core/
├── cmd/
│   ├── crowdmesh-sim/     # CLI for batch parameter sweeps and Monte Carlo runs
│   └── crowdmesh-tui/     # Terminal user interface for step-by-step propagation playback
├── internal/
│   ├── core/              # Zero-I/O protocol engine
│   │   ├── engine/        # Node event loop: Handle(virtualTime, Event) -> []Action
│   │   ├── frame/         # 15-byte binary frame parser, serializer, and validators
│   │   ├── strategy/      # Forwarding strategies (Flooding, Gossip, Counter-based Trickle)
│   │   ├── store/         # G-Set CRDT storage, LRU deduplication, anti-entropy digests
│   │   └── hlc/           # Hybrid Logical Clock implementation
│   ├── sim/               # Discrete-event simulation environment
│   │   ├── cabin/         # Physical node topology and spatial layouts
│   │   ├── radio/         # Log-distance path loss, passenger attenuation, SINR collision model
│   │   ├── mobility/      # Store-and-forward courier model (e.g., passenger moving along aisle)
│   │   └── runner/        # Priority queue virtual time scheduler (container/heap)
│   └── crypto/            # Cryptographic primitives (X25519, ChaCha20-Poly1305, rotating HMAC tag)
├── pkg/
│   └── crowdmesh/         # Public stable API for mobile embedding
├── experiments/           # Configuration files and evaluation scripts
└── docs/                  # Architectural decisions, threat model, protocol specification
```

---

## Getting Started

### Prerequisites

* Go 1.22 or higher

### Running Unit Tests

```bash
# Run all unit tests
go test -v ./...

# Run tests in the core frame package
go test -v ./internal/core/frame
```

### Running the Fuzzing Engine

The binary frame parser uses native Go fuzzing to verify robustness against arbitrary malformed packets:

```bash
go test -fuzz=FuzzParseFrame -fuzztime=10s ./internal/core/frame
```

---

## Experimental Agenda

* **E1:** Baseline broadcast storm characterization of naive flooding under dense topology (180 nodes).
* **E2:** Transmission overhead reduction using counter-based suppression (k=1..3) vs Packet Delivery Ratio (PDR).
* **E3:** Network percolation threshold as a function of application adoption density (10% to 100%).
* **E4:** Convergence latency across network partitions assisted by physical couriers (Store-and-Forward / DTN).
* **E5:** Channel utilization and energy cost of periodic beaconing intervals (0.5s vs 2.0s).

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
