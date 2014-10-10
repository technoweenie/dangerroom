# Danger Room

The Danger Room is a training center for web services to hone their HTTP
request skills.  It includes alien Shi'ar proxy technology, creating the ultimate
SOA simulator.

## API

The `bin/danger` server exposes a JSON API that lets you setup harnesses on
paths.

```
PUT /~danger/foo
Content-Type: application/vnd.danger-room.limiting-harness+json
{
  "target": "https://website.com/to/proxy",
  "harness": {
    "response_size_limit": 500
  }
}
```
