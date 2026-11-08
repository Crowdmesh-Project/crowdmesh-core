# AviaMesh ✈️📡

> Deterministyczny symulator oraz osadzalny silnik protokołu sieci mesh (P2P ad-hoc) bez infrastruktury, zoptymalizowany dla kabin samolotowych i gęstych tłumów.

[![Go Report Card](https://goreportcard.com/badge/avia-mesh)](https://goreportcard.com/report/avia-mesh)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## 🎯 Cel Projektu

AviaMesh to projekt badawczo-inżynieryjny, którego celem jest:
1. Zbudowanie lekkiego silnika protokołu komunikacji w tłumie (np. kabina pasażerska samolotu bez dostępu do Internetu), opartego o bezpołączeniowe rozgłoszenia radiowe (BLE Advertising / Wi-Fi Aware).
2. Zmierzenie kosztów transmisji, zużycia pasma, opóźnień oraz niezawodności w **deterministycznym symulatorze fizyki kabiny**.
3. Utrzymanie rdzenia w architekturze **czystego automatu stanowego (Zero I/O)**, gotowego do bezpośredniego osadzenia w aplikacji mobilnej na Androida i iOS.

Pełny opis koncepcji, modelu propagacji i założeń projektowych znajduje się w dokumencie:
👉 [Dokumentacja Koncepcyjna (docs/concept.md)](docs/concept.md)

---

## 🏗️ Architektura Projektu

Projekt opiera się na standardowym układzie Go (*Go Standard Layout*):

```
avia-mesh/
├── cmd/
│   ├── avia-sim/          # CLI do symulacji wsadowych (badanie hipotez E1–E7)
│   └── avia-tui/          # Interaktywna wizualizacja kabiny w terminalu
├── internal/
│   ├── core/              # Silnik protokołu (Zero I/O): automat, ramka, forwarding, G-Set
│   │   ├── engine/        # Pętla zdarzeń węzła: Node.Handle(Event) -> []Action
│   │   ├── frame/         # Parser i serializator 15-bajtowej ramki binarnej
│   │   ├── strategy/      # Strategie forwardingu (Flooding, Gossip, Trickle)
│   │   ├── store/         # Magazyn CRDT (G-Set), bufor LRU, anti-entropy
│   │   └── hlc/           # Hybrydowy zegar logiczny (HLC)
│   ├── sim/               # Symulator fizyczny
│   │   ├── cabin/         # Model topologii foteli (30 rzędów x 6 miejsc)
│   │   ├── radio/         # Tłumienie fal (Log-distance, shadowing, ciała pasażerów, SINR)
│   │   ├── mobility/      # Generator ruchu wędrowca w alejce i awarii węzłów
│   │   └── runner/        # Deterministyczna dyskretna kolejka zdarzeń czasu wirtualnego
│   └── crypto/            # Prymitywy kryptograficzne (X25519, ChaCha20-Poly1305, HMAC)
├── pkg/
│   └── aviamesh/          # Publiczne API rdzenia
├── experiments/           # Konfiguracje eksperymentów (YAML) i skrypty analizy danych
└── docs/                  # Specyfikacja protokołu, model zagrożeń, ADR, wyniki
```

---

## ⚡ Szybki Start

Wymagania: Go 1.22+.

### Testy jednostkowe i fuzzing
```bash
# Uruchomienie wszystkich testów
go test -v ./...

# Uruchomienie fuzzingu parsera ramek binarnych
go test -fuzz=FuzzParseFrame ./internal/core/frame
```

---

## 📊 Plan Eksperymentów (E1–E5)

- **E1:** Zachowanie i kolizje czystego floodingu przy 180 węzłach w kabinie.
- **E2:** Oszczędność nadawań dzięki tłumieniu licznikowemu (Trickle) w relacji do Packet Delivery Ratio.
- **E3:** Granica perkolacji sieci w zależności od adopcji aplikacji (10%–100%).
- **E4:** Czas konwergencji po rozcięciu sieci dzięki fizycznemu nośnikowi (wędrowiec w alejce).
- **E5:** Koszt pasmowy i energetyczny okresowego beaconingu.

---

## 📜 Licencja
MIT
