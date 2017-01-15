package redis

func getOpenFilesLimit() uint64 {
	// TODO: Find out if there is something like an open file limit on windows
	// and how to get it.
	return 500
}
