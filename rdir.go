package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Wrap FileInfo so that Size() is recursive.
type rdir struct {
	fi     fs.FileInfo
	path   string
	sz     int64
	szOnce sync.Once
}

func (r rdir) Name() string       { return r.fi.Name() }
func (r rdir) Mode() os.FileMode  { return r.fi.Mode() }
func (r rdir) ModTime() time.Time { return r.fi.ModTime() }
func (r rdir) IsDir() bool        { return r.fi.IsDir() }
func (r rdir) Sys() any           { return r.fi.Sys() }
func (r *rdir) Size() int64 {
	r.szOnce.Do(func() {
		err := filepath.WalkDir(r.path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			st, err := d.Info()
			if err != nil {
				return err
			}
			r.sz += st.Size()
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "elles: dirsize for %q: %s\n", r.fi.Name(), err)
		}
	})
	return r.sz
}
