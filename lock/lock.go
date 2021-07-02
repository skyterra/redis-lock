package lock

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gomodule/redigo/redis"
)

const (
	DefaultAcquireTimeout int64 = 100       // 获取锁的timeout时间，默认100ms
	DefaultLockTimeout    int64 = 10 * 1000 // 锁过期时间，默认10s

	RetryInterval time.Duration = 1        // 获取锁失败后，重试间隔，默认1ms
	LockPrefix                  = "stlock" // 锁前缀
)

// 连接redis，返回连接池
func DialRedis(host, password string, port, db int) (*redis.Pool, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialDatabase(db), redis.DialPassword(password))
		},
	}

	conn := pool.Get()
	defer conn.Close()

	if conn.Err() != nil {
		return nil, conn.Err()
	}

	if r, _ := redis.String(conn.Do("PING")); r != "PONG" {
		return nil, errors.New("connect redis failed")
	}

	return pool, nil
}

// 获取分布式锁
func AcquireLock(conn redis.Conn, lockName string, acquireTimeout, lockTimeout int64) (string, error) {
	if acquireTimeout <= 0 {
		acquireTimeout = DefaultAcquireTimeout
	}

	if lockTimeout <= 0 {
		lockTimeout = DefaultLockTimeout
	}

	identifier := uuid.NewString()
	endTime := time.Now().UnixNano()/1e6 + acquireTimeout
	lockName = fmt.Sprintf("%s:%s", LockPrefix, lockName)

	for time.Now().UnixNano()/1e6 < endTime {
		reply, err := redis.String(conn.Do("set", lockName, identifier, "px", lockTimeout, "nx"))
		if err != nil && err != redis.ErrNil {
			return "", err
		}

		// 获取锁成功
		if reply == "OK" {
			return identifier, nil
		}

		// 获取锁失败，1ms后重试
		time.Sleep(RetryInterval * time.Millisecond)
	}

	return "", errors.New("acquire lock failed")
}

// 释放分布式锁
func ReleaseLock(conn redis.Conn, lockName, identifier string) error {
	lockName = fmt.Sprintf("%s:%s", LockPrefix, lockName)

	// 监控锁状态
	conn.Do("watch", lockName)
	defer conn.Do("unwatch", lockName)

	// 检查是否是当前进程添加的锁
	reply, _ := redis.String(conn.Do("get", lockName))
	if reply != identifier {
		return nil
	}

	// 删除锁
	conn.Do("multi")
	conn.Do("del", lockName)
	_, err := redis.Values(conn.Do("exec"))
	if err != nil {
		// 删除锁期间，锁状态发生变化，删除失败
		return err
	}

	return nil
}
