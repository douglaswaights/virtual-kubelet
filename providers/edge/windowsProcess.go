package edge

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// The32CsSnapProcess is described in https://msdn.microsoft.com/de-de/library/windows/desktop/ms682489(v=vs.85).aspx
const The32CsSnapProcess = 0x00000002

// WindowsProcess represent a windows process
type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

func newWindowsProcess(e *syscall.ProcessEntry32) WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return WindowsProcess{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

//GetAllProcesses gets all the windows processes
func GetAllProcesses() ([]WindowsProcess, error) {
	handle, err := syscall.CreateToolhelp32Snapshot(The32CsSnapProcess, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = syscall.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = syscall.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

//FindProcessByName finds the windows process by name
func FindProcessByName(processes []WindowsProcess, name string) []WindowsProcess {
	results := make([]WindowsProcess, 0, 50)
	for _, p := range processes {
		if bytes.Contains([]byte(strings.ToUpper(p.Exe)), []byte(strings.ToUpper(name))) {
			results = append(results, p)
			//return &p
		}
	}
	return results
}

//KillProcessAndChildren windows
func KillProcessAndChildren(process *os.Process) error {
	procID := process.Pid
	all, _ := GetAllProcesses()
	for _, wp := range all {
		if wp.ParentProcessID == procID {
			proc, err := os.FindProcess(wp.ProcessID)
			if err != nil {
				panic(err)
			}
			_ = KillProcessAndChildren(proc)
		}
	}
	err := process.Kill()
	if err != nil {
		return err
	}
	return nil
}
