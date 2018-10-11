package tcpool

type clientPool struct {
	idleConn map[connKey][]*pconn
}
