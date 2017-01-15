// +build !linux

package redis

// This file is used on other operating systems then linux. In this case
// getOpenFilesLimit returns always 500.
// TODO: Find out how to get the open file limit on windows and darvin(mac)
func getOpenFilesLimit() uint64 {
	return 500
}
