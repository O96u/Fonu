//go:build !linux

package nginx

func workerConfirmationSupported()bool{return false}
func activeWorkers(_ int)map[string]bool{return nil}
