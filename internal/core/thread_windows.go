//go:build windows

package core

import "github.com/shirou/gopsutil/v3/process"

func getThread(pid int) int {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0
	}
	count, err := p.NumThreads()
	if err != nil {
		return 0
	}
	return int(count)
}
