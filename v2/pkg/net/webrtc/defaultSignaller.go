package webrtc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pion/webrtc/v3"
)

func DefaultSessionSignaller() SessionSignaller {
	return &ss{
		serverSD: make(chan *webrtc.SessionDescription),
		clientSD: make(chan *webrtc.SessionDescription),
		sigChan:  make(chan int),
	}
}

type ss struct {
	serverSD, clientSD chan *webrtc.SessionDescription
	sigChan            chan int
	id                 int
	started            bool
}

func (s *ss) RegisterServer(ctx context.Context, id string) (<-chan int, error) {
	if s.started {
		return nil, fmt.Errorf("server already registered")
	}

	s.started = true

	go func() {
		for {
			if ctx.Err() != nil {
				close(s.sigChan)
				close(s.serverSD)
				close(s.clientSD)
				return
			}

			// wait for user to press enter
			var line string
			fmt.Scanln(&line)

			s.sigChan <- s.id
			s.id++

			ssd := <-s.serverSD
			ssdBytes, _ := json.Marshal(ssd)
			fmt.Printf("\n\nserver SDP: \n%s\n", string(ssdBytes))

			fmt.Println("Please paste the client SDP below and press enter:")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanLines)

			if scanner.Scan() {
				line = scanner.Text()
			}

			var sd webrtc.SessionDescription
			if err := json.Unmarshal([]byte(line), &sd); err != nil {
				return
			}

			s.clientSD <- &sd
		}
	}()

	return s.sigChan, nil
}

func (s *ss) ExchangeSD(ctx context.Context, _ int, sd *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if s.sigChan == nil {
		return nil, fmt.Errorf("server not registered")
	}

	s.serverSD <- sd
	return <-s.clientSD, nil
}
