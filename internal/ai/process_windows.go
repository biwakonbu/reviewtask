//go:build windows

package ai

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
)

// JobObjectExtendedLimitInformation extends JOBOBJECT_BASIC_LIMIT_INFORMATION
// with additional I/O and memory counters.
type JobObjectExtendedLimitInformation struct {
	BasicLimitInformation JobObjectBasicLimitInformation
	IoInfo                IoCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// JobObjectBasicLimitInformation contains basic job object limit information.
type JobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// IoCounters contains I/O accounting information.
type IoCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

const (
	// JobObjectExtendedLimitInformationClass is the information class for extended limits
	JobObjectExtendedLimitInformationClass = 9
	// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE ensures all processes are terminated when the job handle is closed
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x00002000
)

// processJobInfo stores job information for each command.
// The jobHandle is automatically cleaned up via runtime finalizer or explicit Close().
type processJobInfo struct {
	jobHandle windows.Handle
	cmd       *exec.Cmd
	mu        sync.Mutex
}

// Close releases the job handle explicitly.
// This method is idempotent and safe to call multiple times.
// It's automatically called via finalizer as a safety fallback, but explicit
// calls via killProcessGroup or defer are preferred for deterministic cleanup.
func (ji *processJobInfo) Close() error {
	if ji == nil {
		return nil
	}
	ji.mu.Lock()
	defer ji.mu.Unlock()
	if ji.jobHandle != 0 {
		windows.CloseHandle(ji.jobHandle)
		ji.jobHandle = 0
	}
	return nil
}

var (
	// processJobs maps command pointers to their job info
	processJobs   = make(map[*exec.Cmd]*processJobInfo)
	processJobsMu sync.RWMutex
)

// createJobObject creates and configures a new Windows Job Object
func createJobObject() (windows.Handle, error) {
	// Create a new Job Object
	handle, _, err := procCreateJobObjectW.Call(0, 0)
	if handle == 0 {
		return 0, fmt.Errorf("failed to create job object: %w", err)
	}

	jobHandle := windows.Handle(handle)

	// Configure the job to kill all processes when the job handle is closed
	var info JobObjectExtendedLimitInformation
	info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	ret, _, err := procSetInformationJobObject.Call(
		uintptr(jobHandle),
		JobObjectExtendedLimitInformationClass,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)

	if ret == 0 {
		windows.CloseHandle(jobHandle)
		return 0, fmt.Errorf("failed to configure job object: %w", err)
	}

	return jobHandle, nil
}

// setProcessGroup sets up a Windows Job Object for the process
// This ensures all child processes are terminated when the parent is killed
//
// Note: Unlike Unix where process groups are set up before the process starts,
// Windows requires assigning processes to Job Objects after they start.
// We handle this in killProcessGroup by assigning the process just-in-time.
func setProcessGroup(cmd *exec.Cmd) {
	// Create a new Job Object
	jobHandle, err := createJobObject()
	if err != nil {
		// Failed to create job object, continue without it
		// The process will still work, just without automatic child cleanup
		return
	}

	// Store job info for this command
	jobInfo := &processJobInfo{
		jobHandle: jobHandle,
		cmd:       cmd,
	}

	// Register finalizer for best-effort cleanup in case killProcessGroup is not called.
	// This is a defensive safety measure - explicit cleanup via killProcessGroup is preferred.
	runtime.SetFinalizer(jobInfo, func(ji *processJobInfo) {
		ji.Close()
	})

	processJobsMu.Lock()
	processJobs[cmd] = jobInfo
	processJobsMu.Unlock()

	// Try to assign ASAP after Start() to minimize race with child creation.
	go func(ji *processJobInfo) {
		// Poll briefly for Process to be set by cmd.Start().
		for i := 0; i < 100; i++ { // ~1s total
			p := ji.cmd.Process
			if p != nil {
				h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
				if err == nil {
					ret, _, _ := procAssignProcessToJobObject.Call(uintptr(ji.jobHandle), uintptr(h))
					windows.CloseHandle(h)
					if ret != 0 {
						return
					}
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}(jobInfo)
}

// killProcessGroup kills the entire job object (parent + all children)
func killProcessGroup(process *exec.Cmd) error {
	if process == nil {
		return nil
	}

	// Look up the job info for this command
	processJobsMu.Lock()
	jobInfo := processJobs[process]
	delete(processJobs, process)
	processJobsMu.Unlock()

	// If we have a job handle, try to assign the process and terminate
	if jobInfo != nil && jobInfo.jobHandle != 0 {
		// Clear the finalizer since we're doing explicit cleanup
		runtime.SetFinalizer(jobInfo, nil)

		jobInfo.mu.Lock()
		defer jobInfo.mu.Unlock()

		// If the process has started, try to assign it to the job before terminating
		// This handles the case where the process started but wasn't assigned yet
		if process.Process != nil {
			// Use minimal required rights instead of PROCESS_ALL_ACCESS
			processHandle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Process.Pid))
			if err == nil {
				// Try to assign - ignore "already assigned" failures
				procAssignProcessToJobObject.Call(uintptr(jobInfo.jobHandle), uintptr(processHandle))
				windows.CloseHandle(processHandle)
			}
		}

		// Terminate all processes in the job with exit code 1
		ret, _, termErr := procTerminateJobObject.Call(uintptr(jobInfo.jobHandle), 1)

		// Explicitly close the job handle
		jobInfo.Close()

		// Best-effort tree kill in case children weren't in the job yet.
		if process.Process != nil {
			_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(process.Process.Pid)).Run()
		}
		if ret == 0 && termErr != nil {
			return fmt.Errorf("TerminateJobObject failed: %w", termErr)
		}
		return nil
	}

	// Fallback: just kill the main process if we have one
	if process.Process != nil {
		return process.Process.Kill()
	}

	return nil
}
