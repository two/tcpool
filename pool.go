package tcpool

import (
	"github.com/silenceper/pool"
	"net"
	"sync"
	"time"
)

var IdleTimeout = 15 * time.Second
var InitCap = 5
var MaxCap = 30

type Key struct {
	Proxy, Schema, Addr string
}

var Alive time.Duration = 5 * time.Minute

var mapPool sync.Map

var close = func(v interface{}) error { return v.(net.Conn).Close() }

func Get(k Key) (conn net.Conn, err error) {
	v, ok := mapPool.Load(k)
	if !ok {
		v, err = newPool(k)
		if err != nil {
			return nil, err
		}
		mapPool.Store(k, v)
		go destroy(k)
	}
	iconn, err := v.(pool.Pool).Get()
	if err == nil {
		conn = iconn.(net.Conn)
	}
	return
}
func Put(k Key, conn net.Conn) error {
	var err error
	v, ok := mapPool.Load(k)
	if !ok {
		v, err = newPool(k)
		if err != nil {
			return err
		}
		go destroy(k)
	}
	return v.(pool.Pool).Put(conn)
}

func destroy(k Key) {
	select {
	case <-time.After(Alive):
		mapPool.Delete(k)
	}
}

func newPool(k Key) (pool.Pool, error) {
	factory := func() (interface{}, error) { return net.Dial(k.Schema, k.Addr) }
	poolConfig := &pool.PoolConfig{
		InitialCap:  InitCap,
		MaxCap:      MaxCap,
		Factory:     factory,
		Close:       close,
		IdleTimeout: IdleTimeout,
	}
	return pool.NewChannelPool(poolConfig)
}
