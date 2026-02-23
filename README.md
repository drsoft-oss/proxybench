# proxybench

> Proxy checker and health monitor CLI — validate and benchmark HTTP, SOCKS5, and Shadowsocks proxies.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)

## Features

- **Protocol support**: HTTP, HTTPS, SOCKS5, Shadowsocks (ss://)
- **Auto-detection**: bare `host:port` is probed automatically
- **Speed benchmarks**: latency min/avg/p50/p95/max + loss rate
- **Throughput measurement**: optional large-file download speed test
- **Geo-location**: IP → country via embedded local database (no API)
- **Output formats**: human table, JSON, CSV
- **No external runtime dependencies** — single static binary

---

## Installation

```bash
go install github.com/anonymous-proxies-net/proxybench@latest
```

Or build from source:

```bash
git clone https://github.com/anonymous-proxies-net/proxybench
cd proxybench
go build -o proxybench .
```

---

## Usage

### Check proxies

```bash
# Single proxy
proxybench check http://1.2.3.4:8080

# Multiple proxies
proxybench check socks5://10.0.0.1:1080 http://10.0.0.2:3128

# From a file (one proxy per line)
cat proxies.txt | proxybench check

# JSON output
proxybench check socks5://host:1080 --format json

# CSV output (for spreadsheets / pipelines)
proxybench check socks5://host:1080 --format csv

# Custom test URL and timeout
proxybench check http://host:8080 --test-url http://ifconfig.me --timeout 5
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format`, `-f` | `table` | Output format: `table`, `json`, `csv` |
| `--timeout`, `-t` | `10` | Per-proxy timeout (seconds) |
| `--test-url` | `http://www.google.com` | URL for forward-check requests |
| `--concurrency`, `-c` | `10` | Max parallel checks |
| `--geo` | `true` | Show country info |
| `--db` | auto | Path to `ip2country.csv` |

---

### Benchmark proxies

```bash
proxybench bench http://1.2.3.4:8080
proxybench bench socks5://10.0.0.1:1080 --samples 10 --format json

# With throughput measurement
proxybench bench http://host:8080 --payload-url http://ipv4.download.thinkbroadband.com/10MB.zip
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format`, `-f` | `table` | Output format: `table`, `json`, `csv` |
| `--timeout`, `-t` | `15` | Per-request timeout (seconds) |
| `--samples`, `-n` | `5` | Requests per proxy |
| `--test-url` | `http://www.google.com` | Latency measurement URL |
| `--payload-url` | _(none)_ | Large file URL for speed test |
| `--concurrency`, `-c` | `5` | Max parallel proxies |

---

### Geo database management

The `check` command uses a local IP-to-country CSV database for geo lookups.
A minimal seed file is bundled; download the full database with:

```bash
proxybench db update
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `proxybench db update` | Download latest database from db-ip.com |
| `proxybench db info` | Show current database path, size, and entry count |

**Update flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--dest`, `-d` | auto | Destination path for the database file |
| `--timeout`, `-t` | `120` | Download timeout (seconds) |

The database is sourced from [db-ip.com](https://db-ip.com) (CC BY 4.0, free tier) and updated monthly. No API key required.

---

## Output examples

### Table (default)

```
ADDRESS                                        PROTO    ALIVE   LAT(ms)  COUNTRY          ERROR
──────────────────────────────────────────────────────────────────────────────────────────────
http://1.2.3.4:8080                            http     ✓           243  US United States
socks5://5.6.7.8:1080                          socks5   ✗             0                   dial tcp: connection refused
```

### JSON

```json
[
  {
    "address": "http://1.2.3.4:8080",
    "protocol": "http",
    "alive": true,
    "latency_ms": 243,
    "country": "US United States"
  }
]
```

### CSV

```
address,protocol,alive,latency_ms,country,error
http://1.2.3.4:8080,http,true,243,US United States,
socks5://5.6.7.8:1080,socks5,false,0,,dial tcp: connection refused
```

---

## Architecture

```
proxybench/
├── cmd/            # Cobra CLI commands (check, bench, db)
├── internal/
│   ├── checker/    # Liveness checks (HTTP, SOCKS5, Shadowsocks)
│   ├── bench/      # Latency + throughput benchmarks
│   ├── geo/        # IP→country lookup + DB update
│   └── output/     # JSON / CSV / table formatters
├── data/
│   └── ip2country.csv   # Bundled seed database
└── main.go
```

---

## License

MIT
