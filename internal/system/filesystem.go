package system

import "os"

// fs defines the filesystem operations we NEED
// This is similar to io/fs.FS but with the specific operations we need
type fs interface {
	// Getwd wraps os.Getwd
	Getwd() (string, error)
	// Stat wraps os.Stat
	Stat(name string) (os.FileInfo, error)
}

// osFs implements fs using real OS calls
type osFs struct{}

func (osFs) Getwd() (string, error) {
	return os.Getwd()
}

func (osFs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
