//go:build darwin

package snapshot

import "golang.org/x/sys/unix"

func atomicExchangeDirectories(left, right string) error {
	return unix.RenamexNp(left, right, unix.RENAME_SWAP)
}
