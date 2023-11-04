package events

import (
	"time"
)

// File represents the file sent over by the client in the case
// of a transfer (send or receive)
type File struct {
	// Name is the name presented by the client.
	// This is not necessarily the name that the file will be save with.
	Name string `json:",omitempty"`
	// Path is the path the file was saved to.
	// This will only be set if the file was actually saved to disk.
	Path string `json:",omitempty"`
	MIME string `json:",omitempty"`
	// Size is the size of the file in bytes.
	// This may not always be set.
	Size int64 `json:",omitempty"`

	// TransferSize is the total size oneshot has read in / out.
	// For a successful file transfer, this will be equal to the size of the file.
	TransferSize      int64         `json:",omitempty"`
	TransferStartTime time.Time     `json:",omitempty"`
	TransferEndTime   time.Time     `json:",omitempty"`
	TransferDuration  time.Duration `json:",omitempty"`
	/// TransferRate is given in bytes / second
	TransferRate int64 `json:",omitempty"`

	Content any `json:",omitempty"`
}

// ComputeTransferFields handles calculating field values that could not be
// obtained until after the transfer (successful or not), such as the duration.
func (f *File) ComputeTransferFields() {
	if f == nil {
		return
	}

	f.TransferDuration = f.TransferEndTime.Sub(f.TransferStartTime)
	f.TransferRate = 1000 * 1000 * 1000 * f.TransferSize / int64(f.TransferDuration)
}

func (*File) isEvent() {}
