package events

import (
	"fmt"
	"io"
	"time"

	oneshotfmt "github.com/raphaelreyna/oneshot/v2/internal/out/fmt"
)

type File struct {
	Name string `json:",omitempty"`
	Path string `json:",omitempty"`
	MIME string `json:",omitempty"`
	Size int64  `json:",omitempty"`

	TransferSize      int64         `json:",omitempty"`
	TransferStartTime time.Time     `json:",omitempty"`
	TransferEndTime   time.Time     `json:",omitempty"`
	TransferDuration  time.Duration `json:",omitempty"`
	TransferRate      int64         `json:",omitempty"`

	Content any `json:",omitempty"`
}

func (f *File) ComputeTransferFields() {
	f.TransferDuration = f.TransferEndTime.Sub(f.TransferStartTime)
	f.TransferRate = 1000 * 1000 * 1000 * f.TransferSize / int64(f.TransferDuration)
}

func (f *File) PrettyPrint(w io.Writer) {
	const tmplt = `  Transfer Info:
    Size: %s
	Duration: %v
	Rate: %s`

	fmt.Fprintf(w, tmplt,
		oneshotfmt.PrettySize(f.TransferSize),
		oneshotfmt.RoundedDurationString(f.TransferDuration, 3),
		oneshotfmt.PrettyRate(f.TransferRate),
	)
}

func (*File) isEvent() {}
