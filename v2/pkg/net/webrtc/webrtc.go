package webrtc

import (
	"io"

	"github.com/pion/datachannel"
)

const DataChannelName = "oneshot"

const (
	DataChannelMTU             = 16384              // 16 KB
	BufferedAmountLowThreshold = 1 * DataChannelMTU // 2^0 MTU
	MaxBufferedAmount          = 8 * DataChannelMTU // 2^3 MTUs
)

type DataChannelByteReader struct {
	datachannel.ReadWriteCloser
}

func (r *DataChannelByteReader) Read(p []byte) (int, error) {
	n, isString, err := r.ReadWriteCloser.ReadDataChannel(p)
	if err == nil && isString {
		err = io.EOF
	}
	return n, err
}
