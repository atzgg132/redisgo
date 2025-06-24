# RedisGo

A Redis-compatible, in-memory data store written 100% in Go.

## Features

🚀 **Current Implementation (Phase 1-3)**
- ✅ TCP server on port 6379
- ✅ Complete RESP (Redis Serialization Protocol) parser
- ✅ RESP response formatter
- ✅ Concurrent client handling with goroutines
- ✅ Thread-safe in-memory store with RWMutex
- ✅ Redis string commands: `GET`, `SET`, `DEL`
- ✅ Basic commands: `PING`, `ECHO`
- ✅ Type system with WRONGTYPE error handling
- ✅ Redis-compatible error messages and responses
- ✅ Binary-safe string handling
- ✅ Compatible with redis-cli and raw TCP clients

🔮 **Planned Features**
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

# Test basic commands
redis-cli -p 6379 PING
redis-cli -p 6379 PING "hello"
redis-cli -p 6379 ECHO "world"

# Test string commands
redis-cli -p 6379 SET mykey "Hello World"
redis-cli -p 6379 GET mykey
redis-cli -p 6379 DEL mykey

# Test with special characters
redis-cli -p 6379 SET special "hello\nworld\ttab"
redis-cli -p 6379 GET special

# Test multiple key deletion
redis-cli -p 6379 SET key1 "value1"
redis-cli -p 6379 SET key2 "value2"
redis-cli -p 6379 DEL key1 key2 nonexistent

# Test with raw protocol
echo "PING" | nc localhost 6379
printf "*3\r\n\$3\r\nSET\r\n\$3\r\nkey\r\n\$5\r\nvalue\r\n" | nc localhost 6379
```

## Architecture

- **Concurrent**: Each client connection runs in its own goroutine
- **Thread-Safe Store**: RWMutex enables concurrent reads, exclusive writes
- **Type System**: Entry struct supports multiple Redis data types with validation
- **Protocol**: Full RESP protocol implementation with fallback to inline commands
- **Error Handling**: Redis-compatible error messages and WRONGTYPE validation
- **No Dependencies**: Core logic uses only Go standard library
- **Cross-Platform**: Develops on macOS, deploys on Linux via Docker

### Store Design

```go
type Store struct {
    data map[string]*Entry  // Key-value storage
    mu   sync.RWMutex      // Read-write mutex
}

type Entry struct {
    Type      string       // Data type (string, list, set, hash, sortedset)
    Value     interface{}  // Actual data
    ExpiresAt time.Time    // TTL support (ready for future)
}
```

## Testing

```bash
# Run all unit tests
go test -v

# Test for race conditions
go test -race -v

# Test specific functionality
go test -run TestStoreBasicOperations -v
go test -run TestStoreConcurrency -v
go test -run TestStoreTypeHandling -v

# Integration testing with redis-cli
redis-cli -p 6379 SET test "value"
redis-cli -p 6379 GET test
redis-cli -p 6379 DEL test

# Test concurrent clients
redis-cli -p 6379 SET concurrent1 "value1" &
redis-cli -p 6379 SET concurrent2 "value2" &
redis-cli -p 6379 GET concurrent1 &
redis-cli -p 6379 GET concurrent2 &
wait
```

### Test Coverage

- **Unit Tests**: Store operations, RESP parsing, type handling
- **Concurrency Tests**: RWMutex behavior, race condition detection
- **Integration Tests**: redis-cli compatibility, raw protocol handling
- **Error Handling Tests**: Argument validation, type mismatches

## Development

This project is being developed in phases:
- **Phase 1**: Basic TCP server and connection handling ✅
- **Phase 2**: RESP protocol parsing and basic commands ✅  
- **Phase 3**: Redis string commands and thread-safe store ✅
- **Phase 4**: Core Redis data structures (lists, sets, hashes, sorted sets)
- **Phase 5**: Advanced features (TTL, persistence, replication)
- **Phase 6**: Production readiness (Docker, optimization)

## Module

```
module github.com/atzgg132/redisgo
```

## License

MIT License - see LICENSE file for details.
