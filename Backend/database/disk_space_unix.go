//go:build !windows

package database

import "syscall"

func diskSpace(path string) (uint64, uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, 0, err
	}
	total := stats.Blocks * uint64(stats.Bsize)
	free := stats.Bavail * uint64(stats.Bsize)
	return total, free, nil
}
