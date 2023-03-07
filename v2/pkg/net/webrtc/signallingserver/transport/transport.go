package transport

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
)

func uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

type Transport struct {
	c net.Conn
}

func NewTransport(c net.Conn) *Transport {
	return &Transport{
		c: c,
	}
}

func (t *Transport) Write(m messages.Message) error {
	e, err := newEnvelope(m)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}

	sizePayload := uint64ToBytes(uint64(len(payload)))
	_, err = t.c.Write(sizePayload)
	if err != nil {
		return fmt.Errorf("unable to write size: %w", err)
	}

	for 0 < len(payload) {
		n, err := t.c.Write(payload)
		if err != nil {
			return fmt.Errorf("unable to write payload: %w", err)
		}
		payload = payload[n:]
	}

	return nil
}

func (t *Transport) Read() (messages.Message, error) {
	sizeBuf := make([]byte, 8)
	_, err := t.c.Read(sizeBuf)
	if err != nil {
		return nil, fmt.Errorf("unable to read size: %w", err)
	}

	size := bytesToUint64(sizeBuf)
	buf := make([]byte, size)
	_, err = io.ReadFull(t.c, buf)
	if err != nil {
		return nil, fmt.Errorf("unable to read payload: %w", err)
	}

	var e envelope
	err = json.Unmarshal(buf, &e)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal envelope: %w", err)
	}

	return e.message()
}

func (t *Transport) Close() error {
	return t.c.Close()
}
