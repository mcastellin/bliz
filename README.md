# bliz: Lightning-Fast HTTP Fuzzer

[![Go Report Card](https://goreportcard.com/badge/github.com/mcastellin/bliz)](https://goreportcard.com/report/github.com/mcastellin/bliz)

**bliz** is an HTTP fuzzer that leverages HTTP/1.1 pipelining to achieve request rates **600% faster** than round-trip HTTP interactions. Designed for cybersecurity professionals, bliz excels at identifying race conditions, stress-testing endpoints, and discovering hidden vulnerabilities through high-velocity request flooding.

## Key Features

⚡ **HTTP/1.1 Pipelining** - Send batches of requests using a single TCP connection<br/>
🚀 **600% faster** than standard HTTP request handling<br/>
🎯 Precision response filtering<br/>
🔧 Flexible payload generation (wordlists/numeric ranges)<br/>
📝 Request templating for complex scenarios and malformed requests<br/>
🧵 Multi-threaded concurrent processing<br/>

## Installation

### Precompiled Binaries
Download the latest release for your OS from the [Releases page](https://github.com/mcastellin/bliz/releases).

### Build from Source
1. Install [Go](https://golang.org/dl/)
2. Clone repository:
   ```bash
   git clone https://github.com/mcastellin/bliz.git
   cd bliz
   ```
3. Build and install:
   ```bash
   go install ./...
   ```

## Usage

```
A fast and flexible http fuzzer built with love

Usage:
  bliz [flags]

Flags:
      --batch-size int             Batch size for pipelined requests (default: 100)
      --gn start:end:step:format   Numeric generator (e.g., '0:100:1:%03d')
  -w, --gw filename                Wordlist generator
  -h, --help                       Show help
      --mc httpStatus              Match status codes (default: 200,204,301,302,307,401,403)
  -X, --method string              HTTP method (default: GET)
      --request filename           Request template file ('-' for STDIN)
      --request-scheme scheme      Protocol scheme for templates (default: https)
  -t, --threads int               Processing threads (default: 25)
      --timeout int               Connection timeout in seconds (default: 10)
  -u, --url string                Target URL
```

## Examples

### Basic Wordlist Fuzzing
```bash
bliz -u "https://api.example.com/v1/users/FUZZ" -w wordlist.txt
```

### Numeric Range Fuzzing
Test ID range 1-999 with 3-digit formatting:
```bash
bliz -u "https://api.example.com/orders/%s" --gn "1:999:1:%03d"
```

### Complex Request Template
Test ID range 10000-50000 with custom HTTP request template from STDIN:
```bash
bliz --gn "10000:50000" --request - <<EOF
POST /api/transfer HTTP/1.1
Host: vulnerable-bank.com
Content-Type: application/json
Authorization: Bearer {VALID_TOKEN}

{
  "from": "attacker",
  "to": "victim",
  "amount": FUZZ
}


EOF
```

### Status Code Filtering
Find endpoints returning 500 errors:
```bash
bliz -u "https://example.com/api/FUZZ" -w paths.txt --mc 500
```

## Local testing
This project includes a simple HTTP that can be used for benchmarking:

```bash
# start application stack
docker compose up -d

# introduce some simulated latency
curl -XDELETE localhost:8474/proxies/webserver/toxics/latency
curl -XPOST -H 'Content-Type: application/json' -d '{"name": "latency", "type": "latency", "stream": "upstream", "toxicity": 1.0, "attributes": {"latency": 100, "jitter": 20}}' localhost:8474/proxies/webserver/toxics

# ATTACK!
bliz -u http://localhost:52124/magic/FUZZ.html --gn '0:99999:1:%05d' --mc 200-299
```

## Roadmap
- [ ] Fuzzing modes: clusterbomb/pitchfork
- [ ] Malformed request testing
- [ ] Rate-limiter circumvention
- [ ] Automated response diffing

## License
MIT License - See [LICENSE](LICENSE) for details.
