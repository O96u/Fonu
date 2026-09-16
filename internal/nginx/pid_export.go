package nginx

// ReadPID reads a PID from the given pid file.
func ReadPID(path string) (int, error) {
	return readPIDFile(path)
}

// IsPIDAlive reports whether the process with the given PID is running.
func IsPIDAlive(pid int) bool {
	return isPIDAlive(pid)
}
