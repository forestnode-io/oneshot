package os

import (
	"fmt"
	"os"
	"sort"
)

// ReadDirSorted reads the directory named by dirname and returns a sorted list of directory entries.
// If dirsOnly is true, only directories are returned.
// The entries are sorted by modification time, in ascending order.
func ReadDirSorted(name string, dirsOnly bool) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	info := make(map[os.DirEntry]os.FileInfo, len(entries))
	for _, entry := range entries {
		ei, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to get file info: %w", err)
		}
		if !ei.IsDir() && dirsOnly {
			continue
		}
		info[entry] = ei
	}

	sde := sortableDirEntries{
		entries: entries,
		info:    info,
	}

	sort.Sort(&sde)

	return sde.entries, nil
}

type sortableDirEntries struct {
	entries []os.DirEntry
	info    map[os.DirEntry]os.FileInfo
}

func (s *sortableDirEntries) Len() int {
	return len(s.entries)
}

func (s *sortableDirEntries) Less(i, j int) bool {
	info_i := s.info[s.entries[i]]
	info_j := s.info[s.entries[j]]

	return info_i.ModTime().Before(info_j.ModTime())
}

func (s *sortableDirEntries) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
}
