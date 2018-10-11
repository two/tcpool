package tcpool

import (
	"net"
	"sync"
)

// pconn persist conn
type pconn struct {
	mu     sync.Mutex
	conn   net.Conn
	key    connKey
	maxCap int
}

type connKey struct {
	scheme, addr string
}

func (p *pconn) Get() (interface{}, error) {
	conn, err := net.Dial(p.key.scheme, p.key.addr)
	if err != nil {
		return nil, err
	}
	return conn, nil

}

func (p *pconn) Put(interface{}) error {
	return nil
}
