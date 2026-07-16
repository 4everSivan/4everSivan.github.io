//go:build !darwin && !linux

package snapshot

import "errors"

func atomicExchangeDirectories(_, _ string) error {
	return errors.New("atomic directory exchange is unsupported on this platform")
}
