# Greninvestor

This is a **trading simulator** built as a learning project.
It runs on **hourly intervals**, uses **fake money**, and focuses on **swing trading**, not day trading.

The goal is to experiment with strategies over days/weeks without building an over-engineered system.

---

## Core Ideas

- Hourly decision-making (swing trading)
- Multi-day holding periods
- Deterministic, replayable results
- Minimal compute and operational overhead
- Built to run alongside an existing server

---

## High-Level Design

The system consists of:
- **One Go program** (main execution loop)
- **Python scripts** (alternative decision logics)
- **One SQLite database**

There are no always-on services apart from what already exists on the server.

---

## Responsibilities

### Go (Execution Layer)

- Fetch stock data from Yahoo Finance
- Loop through a configured list of symbols
- Store market data in SQLite
- Aggregate data to hourly candles
- Load current portfolio state
- Call Python for buy/sell decisions
- Execute simulated trades
- Record trades and positions in SQLite

---

### Python (Decision Logic)

- Receive market data (candles + state) from Go
- Decide whether to BUY / SELL / HOLD
- If BUY or SELL, specify how much to trade
- Be stateless per run

---

## Trading Style

- Swing trading
- Hourly candles
- Strategies based on:
  - Trend following
  - Breakouts
  - Mean reversion
  - Trend + pullback
- No scalping or high-frequency logic

---

## Currency Handling

- Single base currency: **INR**
- All portfolio accounting is done in INR
- USD trades are converted to INR at execution time
- FX rate used is stored with each trade

---

## Data Storage

Single SQLite database.

SQLite is chosen for simplicity and ease of maintenance.

---

## Execution Model

- The program runs **once per hour**
- Sequence is explicit and sequential
- No concurrent execution
- Python is spawned on demand and exits after use

The system is mostly idle between runs.

---

## Resource Expectations

Designed to run comfortably on:
- ~1 vCPU
- ~512 MB – 1 GB RAM
- < 10 GB disk

---

## Non-Goals (Explicit)

This project is **not**:
- A real-money trading system
- A low-latency execution engine
- A production-grade quant platform

Clarity and maintainability are more important than realism.

---

## Guiding Principle

> Build something small enough that it actually gets finished and understood months later.
