package server

import (
	"net"
	"time"
)

type listener struct {
	net.Listener
	timer *time.Timer
}

func withTimeout(duration time.Duration, done chan<- struct{}, l net.Listener) net.Listener {
	var ll listener
	ll.Listener = l
	ll.timer = time.AfterFunc(duration, func() {
		done <- struct{}{}
	})
	return &ll
}

func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()

	if l.timer != nil {
		if !l.timer.Stop() {
			<-l.timer.C
		}
		l.timer = nil
	}

	return conn, err
}

func (l *listener) Close() error {
	return l.Listener.Close()
}

func (l *listener) Addr() net.Addr {
	return l.Listener.Addr()
}
