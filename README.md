# TurboIntruder

This is my attempt to replicate the features provided by TurboIntruder in Go.

* Use HTTP 1.x pipelining to batch-up HTTP requests write/reads.
* Multithreading

```bash
curl -XDELETE localhost:8474/proxies/webserver/toxics/latency
curl -XPOST -H 'Content-Type: application/json' -d '{"name": "latency", "type": "latency", "stream": "upstream", "toxicity": 1.0, "attributes": {"latency": 100, "jitter": 20}}' localhost:8474/proxies/webserver/toxics
```

## Debugging

As this application aggressively uses goroutines, it will come the time where you need to debug hanging routines. To show all running goroutines on `panic()` set the following environment variable:

```
GOTRACEBACK=all
```
