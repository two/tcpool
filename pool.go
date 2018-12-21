package tcpool

import (
	"errors"
	"sync"
	"time"

	"github.com/two/pool"
)

type Pool struct {
	mapPool    sync.Map
	CloseMap   sync.Map
	FactoryMap sync.Map

	idleTimeOut time.Duration
	alive       time.Duration
	initCap     int
	maxCap      int
}

type Key struct {
	Proxy, Schema, Addr string
}

const IdleTimeout = 15 * time.Second
const InitCap = 5
const MaxCap = 30
const Alive time.Duration = 5 * time.Minute

func (p *Pool) Get(k Key) (conn interface{}, err error) {
	v, ok := p.mapPool.Load(k)
	if !ok {
		v, err = p.newPool(k)
		if err != nil {
			return nil, err
		}
		p.mapPool.Store(k, v)
		go p.destroy(k)
	}
	conn, err = v.(pool.Pool).Get()
	return
}

func (p *Pool) Put(k Key, conn interface{}) error {
	var err error
	v, ok := p.mapPool.Load(k)
	if !ok {
		v, err = p.newPool(k)
		if err != nil {
			return err
		}
		go p.destroy(k)
	}
	return v.(pool.Pool).Put(conn)
}

func (p *Pool) destroy(k Key) {
	select {
	case <-time.After(Alive):
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
		InitialCap:  p.GetInitCap(),
		MaxCap:      p.GetMaxCap(),
		Factory:     fm.(func() (interface{}, error)),
		Close:       cm.(func(v interface{}) error),
		IdleTimeout: p.GetIdleTimeOut(),
	}
	return pool.NewChannelPool(poolConfig)
}

func (p *Pool) SetIdleTimeOut(t time.Duration) {
	p.idleTimeOut = t
}

func (p *Pool) GetIdleTimeOut() time.Duration {
	if p.idleTimeOut.Nanoseconds() > 0 {
		return p.idleTimeOut
	}
	return IdleTimeout
}

func (p *Pool) SetAlive(t time.Duration) {
	p.alive = t
}

func (p *Pool) GetAlive() time.Duration {
	if p.alive.Nanoseconds() > 0 {
		return p.alive
	}
	return Alive
}

func (p *Pool) SetInitCap(i int) {
	p.initCap = i
}

func (p *Pool) GetInitCap() int {
	if p.initCap > 0 {
		return p.initCap
	}
	return InitCap
}

func (p *Pool) SetMaxCap(i int) {
	p.maxCap = i
}

func (p *Pool) GetMaxCap() int {
	if p.maxCap > 0 {
		return p.maxCap
	}
	return MaxCap
}
