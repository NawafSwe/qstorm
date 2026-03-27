<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="img/qstorm-banner-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="img/qstorm-banner-light.svg">
    <img alt="QStorm" src="img/qstorm-banner-dark.svg" width="600">
  </picture>
</p>

<p align="center">
  Load testing for async message queues. Built for engineers who test more than just HTTP.
</p>

<p align="center">
  Inspired by <a href="https://k6.io">k6</a>. Same philosophy, different protocol.
</p>

<p align="center">
  <a href="https://github.com/nawafswe/qstorm/actions/workflows/test.yaml"><img src="https://github.com/nawafswe/qstorm/actions/workflows/test.yaml/badge.svg" alt="Build Status"></a>
  <a href="https://codecov.io/gh/nawafswe/qstorm"><img src="https://codecov.io/gh/nawafswe/qstorm/branch/main/graph/badge.svg" alt="Coverage"></a>
  <a href="https://goreportcard.com/report/github.com/nawafswe/qstorm"><img src="https://goreportcard.com/badge/github.com/nawafswe/qstorm" alt="Go Report Card"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.26-blue.svg" alt="Go Version"></a>
  <a href="https://github.com/nawafswe/qstorm/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://github.com/nawafswe/qstorm/releases"><img src="https://img.shields.io/github/v/release/nawafswe/qstorm" alt="Release"></a>
  <a href="https://github.com/nawafswe/qstorm/stargazers"><img src="https://img.shields.io/github/stars/nawafswe/qstorm?style=social" alt="GitHub Stars"></a>
  <a href="https://github.com/nawafswe/qstorm/network/members"><img src="https://img.shields.io/github/forks/nawafswe/qstorm?style=social" alt="GitHub Forks"></a>
</p>

---

## Overview

Modern backend systems rely heavily on async workers: services that consume messages from queues and process them in the background. Tools like **k6** and **Locust** are excellent for HTTP load testing, and while they can be extended to work with queues, the setup isn't always straightforward.

**QStorm** aims to bring that same familiar experience (stages, rates, live metrics) to message queues with zero configuration overhead. Define a config, point it at your queue, and run.

QStorm is a **client-side tool**. It runs from your machine or CI pipeline and publishes to the queue. No need to deploy it alongside your workers.

### Features

- **Stage-based load profiles**: define multi-stage tests with different rates and durations
- **Template variables**: `{{uuid}}` and `{{timestamp}}` generate unique values per message
- **Live progress**: real-time terminal output during test execution
- **Accurate metrics**: HDR Histogram for latency percentiles (p50, p75, p90, p99)
- **Error summary**: distinct errors aggregated by type with counts
- **Graceful shutdown**: `Ctrl+C` stops the test and prints collected results

## Queue Support

|                                                | Queue | Status |
|:----------------------------------------------:|---|:---:|
|   <img src="img/pubsub-gcp.svg" width="16">    | Google Cloud PubSub | Supported |
|  <img src="img/kafka-apache.svg" width="16">   | Apache Kafka | Supported |
|    <img src="img/RabbitMQ.svg" width="16">     | RabbitMQ | Supported |
|  <img src="img/pulsar-apache.svg" width="16">  | Apache Pulsar | Planned |
| <img src="img/activeMQ-apache.svg" width="16"> | Apache ActiveMQ | Planned |

## Installing

### Prerequisites

- Go 1.26+
- Docker (for running queues locally)

### Build from source

```bash
git clone https://github.com/nawafswe/qstorm.git
cd qstorm
make build
```

### Add to PATH

To run `qstorm` from anywhere:

```bash
# Option 1: symlink to a directory already in PATH
sudo ln -s $(pwd)/bin/qstorm /usr/local/bin/qstorm

# Option 2: add the bin directory to PATH (add to ~/.zshrc or ~/.bashrc)
export PATH="$PATH:/path/to/qstorm/bin"
```

Verify:

```bash
qstorm --version
```

## Usage

### 1. Start queues locally

```bash
# Start all services
make environment

# Or start individually
make gcp-pubsub   # PubSub emulator (port 8095)
make kafka         # Kafka plaintext (9092) + SASL (9093)
make rabbitmq      # RabbitMQ (port 5672)
```

### 2. Configure connection credentials

```bash
make env  # copies .env.sample -> .env
```

Edit `.env` to match your queue. See [Queue Configuration](#queue-configuration) for all options.

### 3. Create a test config

See [Queue Configuration](#queue-configuration) for full details per queue type.

### 4. Run

```bash
# positional argument
qstorm config.json

# with flags
qstorm --config config.json --env .env
```

| Flag | Default | Description |
|---|---|---|
| `--config` | _(required)_ | Path to the JSON test config file |
| `--env` | `.env` | Path to the `.env` connection file |

### Example output

```
      ___  ____  _
     / _ \/ ___|| |_ ___  _ __ _ __ ___
    | | | \___ \| __/ _ \| '__| '_ ` _ \
    | |_| |___) | || (_) | |  | | | | | |
     \__\_\____/ \__\___/|_|  |_| |_| |_|

  execution: local
  queue:     apache-kafka
  topic:     qstorm-topic
  stages:    2 configured, ~40s total
  expected:  ~600 messages

    -> stage 1: 30s @ 10 msg/s
    -> stage 2: 10s @ 30 msg/s

  ------------------------------------------------------------------

     v published......: 598
     x failed.........: 2

         connection refused.......: 2

       success_rate...: 99.67%
       error_rate.....: 0.33%

       publish_latency: avg=2.1ms  p50=1.9ms  p75=2.4ms  p90=3.2ms  p99=8.1ms

       duration.......: 40.012s

  ------------------------------------------------------------------
```

## Configuration Reference

QStorm uses two separate sources:
- A **JSON config file** defines the test (what to publish, how fast, for how long)
- A **`.env` file** provides connection credentials

### Shared Config

Every queue type uses the same top-level structure:

```json
{
  "QUEUE": {
    "TYPE": "gcp-pubsub | apache-kafka | rabbitmq",
    "PAYLOAD": "...",
    "ATTRIBUTES": "...",
    "PUBSUB": { },
    "KAFKA": { },
    "RABBITMQ": { }
  },
  "STAGES": [ ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `TYPE` | string | yes | Queue type: `gcp-pubsub`, `apache-kafka`, `rabbitmq` |
| `PAYLOAD` | string | yes | Message body. Supports [template variables](#template-variables) |
| `ATTRIBUTES` | string | no | JSON key-value pairs attached to each message. Supports [template variables](#template-variables). Maps to PubSub attributes, Kafka headers, RabbitMQ headers |

#### STAGES

Each stage runs sequentially. Define as many as needed.

| Field | Type | Required | Description |
|---|---|---|---|
| `DURATION` | string | yes | How long the stage runs (e.g. `"30s"`, `"2m"`) |
| `RATE` | int | yes | Messages per second during this stage |

---

## Queue Configuration

### Google Cloud PubSub

#### Test config (JSON)

```json
{
  "QUEUE": {
    "TYPE": "gcp-pubsub",
    "PAYLOAD": "{\"order_id\": \"{{uuid}}\"}",
    "ATTRIBUTES": "{\"SOURCE\": \"qstorm\"}",
    "PUBSUB": {
      "TOPIC": "qstorm-topic",
      "ORDERING_KEY": "customer-123"
    }
  },
  "STAGES": [
    { "DURATION": "30s", "RATE": 50 }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `TOPIC` | string | yes | Full topic name (e.g. `projects/my-project/topics/my-topic` for real GCP, or just `my-topic` for emulator) |
| `ORDERING_KEY` | string | no | Enables ordered delivery. Messages with the same key are delivered in order |

#### Connection config (`.env`)

| Variable | Required | Description |
|---|---|---|
| `PUBSUB__PROJECT_ID` | yes | GCP project ID |
| `PUBSUB__EMULATOR_HOST` | no | Emulator address (e.g. `localhost:8095`). When set, connects to emulator instead of GCP |
| `PUBSUB__CREDENTIALS_FILE` | no | Service account JSON credentials |

> See [Google Cloud PubSub documentation](https://cloud.google.com/pubsub/docs) for more details.

---

### Apache Kafka

#### Test config (JSON)

```json
{
  "QUEUE": {
    "TYPE": "apache-kafka",
    "PAYLOAD": "{\"order_id\": \"{{uuid}}\"}",
    "ATTRIBUTES": "{\"SOURCE\": \"qstorm\"}",
    "KAFKA": {
      "TOPIC": "qstorm-topic",
      "KEY": "order-service",
      "PARTITION": -1,
      "PRODUCER": {
        "ACKS": -1,
        "COMPRESSION_TYPE": "snappy",
        "LINGER_MS": 10,
        "BATCH_SIZE": 32768
      }
    }
  },
  "STAGES": [
    { "DURATION": "30s", "RATE": 50 }
  ]
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `TOPIC` | string | yes | - | Kafka topic name |
| `KEY` | string | no | empty (round-robin) | Message key. Used for partition assignment. Messages with the same key go to the same partition. When empty, Kafka distributes messages via round-robin |
| `PARTITION` | int | no | -1 (any) | Explicit partition number. `0` or unset defaults to automatic assignment via key hash |

#### Producer tuning (`KAFKA.PRODUCER`)

All optional. Sensible defaults are used when omitted.

| Field | Type | Default | Description |
|---|---|---|---|
| `ACKS` | int (nullable) | -1 (all) | Broker acknowledgments. `0` = fire-and-forget, `1` = leader only, `-1` = all in-sync replicas |
| `COMPRESSION_TYPE` | string | none | Compression: `none`, `gzip`, `snappy`, `lz4`, `zstd` |
| `LINGER_MS` | int | 0 | Batching delay in ms. Higher = more batching = higher throughput |
| `BATCH_SIZE` | int | librdkafka default | Max bytes per batch |

> See [librdkafka configuration reference](https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md) for all available producer options.

#### Connection config (`.env`)

| Variable | Required | Description |
|---|---|---|
| `KAFKA__BOOTSTRAP_SERVERS` | yes | Comma-separated broker addresses (e.g. `localhost:9092`) |
| `KAFKA__SECURITY_PROTOCOL` | no | `PLAINTEXT` (default), `SASL_PLAINTEXT`, `SASL_SSL`, `SSL` |
| `KAFKA__SASL_MECHANISM` | no | `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512` |
| `KAFKA__SASL_USERNAME` | no | SASL username |
| `KAFKA__SASL_PASSWORD` | no | SASL password (redacted in logs) |

**Local (no auth):**
```env
KAFKA__BOOTSTRAP_SERVERS=localhost:9092
```

**Cloud (Confluent Cloud, AWS MSK, Aiven, etc.):**
```env
KAFKA__BOOTSTRAP_SERVERS=broker.cloud:9093
KAFKA__SECURITY_PROTOCOL=SASL_SSL
KAFKA__SASL_MECHANISM=PLAIN
KAFKA__SASL_USERNAME=api-key
KAFKA__SASL_PASSWORD=api-secret
```

> See [Apache Kafka documentation](https://kafka.apache.org/documentation/) for more details on security configuration.

---

### RabbitMQ

#### Test config (JSON)

```json
{
  "QUEUE": {
    "TYPE": "rabbitmq",
    "PAYLOAD": "{\"order_id\": \"{{uuid}}\"}",
    "ATTRIBUTES": "{\"SOURCE\": \"qstorm\"}",
    "RABBITMQ": {
      "QUEUE": {
        "NAME": "qstorm-queue",
        "DURABLE": true,
        "AUTO_DELETE": false,
        "EXCLUSIVE": false,
        "NO_WAIT": false,
        "ARGS": {
          "x-queue-type": "quorum"
        }
      },
      "EXCHANGE": {
        "NAME": "qstorm-exchange",
        "KIND": "direct",
        "DURABLE": true,
        "AUTO_DELETE": false,
        "INTERNAL": false,
        "NO_WAIT": false
      },
      "CHANNEL": {
        "CONFIRM_MODE": true
      },
      "PUBLISHER": {
        "ROUTING_KEY": "qstorm-routing-key",
        "CONTENT_TYPE": "application/json",
        "DELIVERY_MODE": 2
      }
    }
  },
  "STAGES": [
    { "DURATION": "30s", "RATE": 50 }
  ]
}
```

#### Queue config (`RABBITMQ.QUEUE`)

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `NAME` | string | yes | - | Queue name |
| `DURABLE` | bool | no | false | Queue survives broker restarts. Required for quorum queues |
| `AUTO_DELETE` | bool | no | false | Queue deleted when last consumer disconnects. Not supported with quorum queues |
| `EXCLUSIVE` | bool | no | false | Queue only accessible by declaring connection. Not supported with quorum queues |
| `NO_WAIT` | bool | no | false | Skip server confirmation of declaration |
| `ARGS` | object | no | nil | Queue arguments. Set `"x-queue-type"` to `"classic"`, `"quorum"`, or `"stream"` |

#### Exchange config (`RABBITMQ.EXCHANGE`)

Optional. When `NAME` is empty, messages are published to the default exchange (routed directly by queue name).

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `NAME` | string | no | "" (default exchange) | Exchange name |
| `KIND` | string | no | - | Exchange type: `direct`, `fanout`, `topic`, `headers` |
| `DURABLE` | bool | no | false | Exchange survives broker restarts |
| `AUTO_DELETE` | bool | no | false | Exchange deleted when no bindings remain |
| `INTERNAL` | bool | no | false | Exchange cannot receive publishes directly |
| `NO_WAIT` | bool | no | false | Skip server confirmation |
| `ARGS` | object | no | nil | Exchange arguments |

#### Channel config (`RABBITMQ.CHANNEL`)

| Field | Type | Default | Description |
|---|---|---|---|
| `CONFIRM_MODE` | bool | false | Enable publisher confirms. When true, each published message is acknowledged by the broker. Adds overhead but gives accurate delivery success metrics |

#### Publisher config (`RABBITMQ.PUBLISHER`)

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `ROUTING_KEY` | string | yes | - | Routing key. For default exchange, this must be the queue name |
| `MANDATORY` | bool | no | false | Return unroutable messages instead of silently dropping them |
| `CONTENT_TYPE` | string | no | "" | MIME type (e.g. `application/json`). Informational only |
| `DELIVERY_MODE` | int | no | 0 (transient) | `1` = transient (in-memory, fastest), `2` = persistent (written to disk, survives broker restart) |
| `PRIORITY` | int | no | 0 | Message priority (0-9). Requires queue with `x-max-priority` argument |

#### Connection config (`.env`)

| Variable | Required | Description |
|---|---|---|
| `RABBITMQ__URL` | yes | AMQP connection URL |

**Local:**
```env
RABBITMQ__URL=amqp://guest:guest@localhost:5672
```

**TLS:**
```env
RABBITMQ__URL=amqps://user:pass@broker.prod:5671
```

> See [RabbitMQ documentation](https://www.rabbitmq.com/docs) for more details.

---

## Concepts

### Stages

Stages define how traffic changes over time. Each stage has a **duration** and a **rate** (messages per second). Stages run sequentially. Use them to model ramp-ups, sustained load, spikes, and cooldowns.

```mermaid
---
config:
    xyChart:
        xAxis:
            label: "Time"
        yAxis:
            label: "Messages/sec"
---
xychart-beta
    title "Ramp -> Sustain -> Spike -> Cooldown"
    x-axis ["0s", "30s", "60s", "90s", "120s", "150s", "180s"]
    y-axis 0 --> 1000
    line [0, 100, 500, 500, 1000, 200, 200]
```

### Template variables

Template variables are supported in `PAYLOAD` and `ATTRIBUTES` only. They are **not** processed in queue-specific configuration fields (topic, key, partition, etc.) to avoid unexpected behavior.

| Variable | Description | Example |
|---|---|---|
| `{{uuid}}` | Unique UUID per occurrence | `f47ac10b-58cc-4372-a567-0e02b2c3d479` |
| `{{timestamp}}` | Current UTC time (RFC 3339) | `2026-03-23T14:30:00Z` |

Each `{{uuid}}` in a single message resolves to a **different** value.

### Metrics

QStorm collects metrics using [HDR Histogram](https://github.com/HdrHistogram/hdrhistogram-go) for accurate latency percentiles:

- **published / failed**: total message counts
- **errors overview**: distinct error messages with occurrence counts
- **success_rate / error_rate**: as percentages
- **publish_latency**: avg, p50, p75, p90, p99
- **duration**: total test time

## Roadmap

- [ ] Apache Pulsar support
- [ ] Apache ActiveMQ support
- [ ] Consumer lag metrics (cross-queue)
- [ ] Threshold assertions (fail if p99 > Xms or error rate > Y%)
- [ ] Result export (JSON, CSV) for CI/CD integration
- [ ] Custom template functions (`{{rand_int 1 100}}`, `{{rand_string 10}}`)

## License

[Apache License 2.0](LICENSE)
