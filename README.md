![example workflow](https://github.com/skyterra/redis-lock/actions/workflows/go.yml/badge.svg)
# redis-lock
使用redis实现分布式锁

# 使用说明

```go
package main

import (
	"github.com/skyterra/redis-lock/lock"
	"log"
)

func main() {
	lockName := "doc:abc:123:efg"

	p, err := lock.DialRedis("127.0.0.1", "", 6379, 0)
	if err != nil {
		log.Fatal(err)
	}
	
	conn := p.Get()
	id, err := lock.AcquireLock(conn, lockName, 100, 10*1000)
	// TODO your operation.


	lock.ReleaseLock(conn, lockName, id)
}
```
