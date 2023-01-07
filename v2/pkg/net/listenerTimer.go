package network

import (
	"net"
	"time"
)

type ListenerTimer struct {
	net.Listener
	timer *time.Timer
	C     <-chan time.Time
}

func NewListenerTimer(l net.Listener, d time.Duration) *ListenerTimer {
	var ll ListenerTimer
	ll.Listener = l
	ll.timer = time.NewTimer(d)
	ll.C = ll.timer.C
	return &ll
}

func (l *ListenerTimer) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()

	if l.timer != nil {
		if !l.timer.Stop() {
			<-l.timer.C
		}
		l.timer = nil
	}

	return conn, err
}

func (l *ListenerTimer) Close() error {
	return l.Listener.Close()
}

func (l *ListenerTimer) Addr() net.Addr {
	return l.Listener.Addr()
}
