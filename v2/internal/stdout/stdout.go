package stdout

import (
	"fmt"
	"io"

	"github.com/raphaelreyna/oneshot/v2/internal/network"
)

type stdout struct {
	w                 io.Writer
	wantsJSON         bool
	jopts             string
	receivingToStdout bool
}

func (s *stdout) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

func (s *stdout) writeListeningOn(scheme, host, port string) {
	if s.wantsJSON || s.receivingToStdout {
		return
	}

	if host == "" {
		addrs, err := network.HostAddresses()
		if err != nil {
			fmt.Fprintf(s.w, "listening on: %s://localhost%s\n", scheme, port)
			return
		}

		fmt.Fprintln(s, "listening on: ")
		for _, addr := range addrs {
			fmt.Fprintf(s.w, "\t- %s://%s\n", scheme, address(addr, port))
		}
		return
	}

	fmt.Fprintf(s, "listening on: %s://%s\n", scheme, address(host, port))
}

func address(host, port string) string {
	if port != "" {
		port = ":" + port
	}

	return host + port
}
