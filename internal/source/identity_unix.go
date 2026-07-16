//go:build unix

package source

import (
	"io/fs"
	"syscall"
)

func stateFromInfo(info fs.FileInfo) FileState {
	state := FileState{
		Size:            info.Size(),
		Mode:            info.Mode(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		state.Device = uint64(stat.Dev)
		state.Inode = uint64(stat.Ino)
	}
	return state
}
