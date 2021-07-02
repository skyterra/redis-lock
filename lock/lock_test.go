package lock_test

import (
	"strconv"
	"sync"

	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/skyterra/redis-lock/lock"
)

var _ = Describe("Lock", func() {
	Context("Dial Redis", func() {
		It("should be succeed", func() {
			p, err := lock.DialRedis("127.0.0.1", "", 6379, 0)
			Expect(err).Should(Succeed())

			conn := p.Get()
			reply, err := redis.String(conn.Do("set", "redis-lock", "this is a redis lock"))
			Expect(err).Should(Succeed())
			Expect(reply == "OK").Should(BeTrue())

			reply, err = redis.String(conn.Do("get", "redis-lock"))
			Expect(err).Should(Succeed())
			Expect(reply == "this is a redis lock").Should(BeTrue())
		})
	})

	Context("Acquire Lock", func() {
		It("should be succeed", func() {
			p, err := lock.DialRedis("127.0.0.1", "", 6379, 0)
			Expect(err).Should(Succeed())

			lockName := "doc:xxx:123:xxx"
			conn := p.Get()
			conn.Do("set", lockName, 1)

			total := 10
			wg := sync.WaitGroup{}
			wg.Add(total)

			for i := 0; i < total; i++ {
				go func() {
					conn := p.Get()
					id, err := lock.AcquireLock(conn, lockName, 100, 10*1000)
					Expect(err).Should(Succeed())

					v, err := redis.String(conn.Do("get", lockName))
					number, _ := strconv.Atoi(v)
					conn.Do("set", lockName, number+1)

					lock.ReleaseLock(conn, lockName, id)
					wg.Done()
				}()
			}

			wg.Wait()

			reply, _ := redis.String(conn.Do("get", lockName))
			number, _ := strconv.Atoi(reply)
			Expect(number == 11).Should(BeTrue())
		})
	})

})
