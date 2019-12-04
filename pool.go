package tcpool

import (
	"errors"
	"sync"
	"time"

	"github.com/two/pool"
)

// Pool is a struct contains connection pool
// which every host have a pool
type Pool struct {
	mu         sync.Mutex
	mapPool    sync.Map
	CloseMap   sync.Map
	FactoryMap sync.Map

	IdleTimeOut time.Duration
	Alive       time.Duration
	InitCap     int
	MaxCap      int
}

// Func is a struct contains factory function
// and close function
type Func struct {
	Factory func() (interface{}, error)
	Close   func(v interface{}) error
}

// Key is a struct as host
type Key struct {
	Proxy, Schema, Addr string
}

const (
	idleTimeout               = 15 * time.Second
	initCap                   = 0
	maxCap                    = 30
	alive       time.Duration = 5 * time.Minute
)

// Get will return a connection which host is k
// if there is no k exist, will create a new pool
// and at the same time only on pool will be saved
// to map with key k, the other pool will be destroy
func (p *Pool) Get(k Key) (interface{}, error) {
	v, ok := p.mapPool.Load(k)
	if !ok {
		v, ok = p.mapPool.Load(k)
		// 下面这段不能放到锁里，因为当新建连接时间过长
		// 会导致整个获取连接的时间过长，并发情况下后面的
		// 请求都会等待解锁，导致等待时间过长
		if !ok {
			nv, err := p.newPool(k)
			if err != nil {
				return nil, err
			}
			v, ok = p.mapPool.LoadOrStore(k, nv)
			if ok {
				// 已经存在，则当前的 pool 要及时销毁
				// 否则会出现连接泄露的情况
				go nv.(pool.Pool).Release()
			} else {
				// 如果存储的是当前的，需要定时销毁
				go p.destroy(k)
			}
		}
	}
	return v.(pool.Pool).Get()
}

// Put will have a connection put into a pool
// if no pool of map with key k, return error
func (p *Pool) Put(k Key, conn interface{}) error {
	v, ok := p.mapPool.Load(k)
	if !ok {
		return errors.New("connection pool not found")
	}
	return v.(pool.Pool).Put(conn)
}

func (p *Pool) destroy(k Key) {
	select {
	case <-time.After(p.alive()):
		v, ok := p.mapPool.Load(k)
		if ok {
			v.(pool.Pool).Release()
		}
		p.mapPool.Delete(k)
	}
}

func (p *Pool) newPool(k Key) (pool.Pool, error) {
	fm, ok := p.FactoryMap.Load(k)
	if !ok {
		return nil, errors.New("load factory map failed")
	}
	cm, ok := p.CloseMap.Load(k)
	if !ok {
		return nil, errors.New("load close map failed")
	}

	poolConfig := &pool.PoolConfig{
		InitialCap:  p.initCap(),
		MaxCap:      p.maxCap(),
		Factory:     fm.(func() (interface{}, error)),
		Close:       cm.(func(v interface{}) error),
		IdleTimeout: p.idleTimeOut(),
	}
	return pool.NewChannelPool(poolConfig)
}

func (p *Pool) idleTimeOut() time.Duration {
	if p.IdleTimeOut.Nanoseconds() > 0 {
		return p.IdleTimeOut
	}
	return idleTimeout
}

func (p *Pool) alive() time.Duration {
	if p.Alive.Nanoseconds() > 0 {
		return p.Alive
	}
	return alive
}

func (p *Pool) initCap() int {
	if p.InitCap > 0 {
		return p.InitCap
	}
	return initCap
}

func (p *Pool) maxCap() int {
	if p.MaxCap > 0 {
		return p.MaxCap
	}
	return maxCap
}

// SetFunc will put factory function
// and close function to the pool
// the next time call newPool will
// use these function to create new
// a connection and close a connection
func (p *Pool) SetFunc(k Key, c Func) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.FactoryMap.Store(k, c.Factory)
	p.CloseMap.Store(k, c.Close)
}
