//go:build !unix

package source

import "io/fs"

func stateFromInfo(info fs.FileInfo) FileState {
	return FileState{
		Size:            info.Size(),
		Mode:            info.Mode(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
	}
}
