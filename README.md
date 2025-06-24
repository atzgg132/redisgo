# RedisGo

A Redis-compatible, in-memory data store written 100% in Go.

## Features

🚀 **Current Implementation (Phase 1-2)**
- ✅ TCP server on port 6379
- ✅ Complete RESP (Redis Serialization Protocol) parser
- ✅ RESP response formatter
- ✅ Concurrent client handling
- ✅ Basic commands: `PING`, `ECHO`
- ✅ Compatible with redis-cli and raw TCP clients

🔮 **Planned Features**
- String operations (GET, SET, DEL)
- List operations (LPUSH, RPUSH, LPOP, RPOP, LLEN)
- Set operations (SADD, SREM, SMEMBERS, SISMEMBER)
- Hash operations (HGET, HSET, HDEL, HGETALL)
- Sorted set operations (ZADD, ZREM, ZRANGE, ZSCORE)
- TTL support (EXPIRE, TTL)
- RDB snapshots
- AOF (Append-Only File) persistence
- Master/replica synchronization
- Docker support

## Quick Start

```bash
# Run the server
go run main.go

# Test with redis-cli (if installed)
redis-cli -p 6379 PING
redis-cli -p 6379 PING "hello"
redis-cli -p 6379 ECHO "world"

# Test with netcat
echo "PING" | nc localhost 6379
printf "*2\r\n\$4\r\nECHO\r\n\$5\r\nhello\r\n" | nc localhost 6379
```

## Architecture

- **Concurrent**: Each client connection runs in its own goroutine
- **Protocol**: Full RESP protocol implementation with fallback to inline commands
- **No Dependencies**: Core logic uses only Go standard library
- **Cross-Platform**: Develops on macOS, deploys on Linux via Docker

## Testing

```bash
# Run unit tests
go test -v

# Test concurrent clients
redis-cli -p 6379 PING "client1" &
redis-cli -p 6379 PING "client2" &
wait
```

## Development

This project is being developed in phases:
- **Phase 1**: Basic TCP server and connection handling ✅
- **Phase 2**: RESP protocol parsing and basic commands ✅  
- **Phase 3**: Core Redis data structures (strings, lists, sets, hashes)
- **Phase 4**: Advanced features (TTL, persistence, replication)
- **Phase 5**: Production readiness (Docker, optimization)

## Module

```
module github.com/atzgg132/redisgo
```

## License

MIT License - see LICENSE file for details.
