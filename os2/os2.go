package os2

import (
	"os"
)

// Like os.ReadDir, but without the sort.
func ReadDir(name string) ([]os.DirEntry, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}
