package redis

import "log"
import "golang.org/x/sys/unix"

func getOpenFilesLimit() uint64 {
	var rLimit unix.Rlimit

	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Fatalf("Can not identify limit of open files: %s", err)
	}
	return rLimit.Cur
}
