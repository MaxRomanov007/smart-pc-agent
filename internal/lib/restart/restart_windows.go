//go:build windows

package restart

import (
	"os"
	"syscall"
)

// Now spawns a detached copy of the binary, then stops the application and exits.
//
// Order matters:
//  1. CreateProcess first — the new process starts before we touch anything
//  2. stop() second      — graceful shutdown of the current process
//  3. os.Exit(0)         — current process exits cleanly
//
// This guarantees the child is already running before the parent tears down,
// avoiding a race where stop() cancels context and goroutines release resources
// that the child might need (e.g. open ports, DB connections).
//
// exec.Command is avoided because the agent is built with -H windowsgui and
// has no console — inheriting Stdin/Stdout/Stderr causes errors on some
// Windows versions. syscall.CreateProcess with bInheritHandles=false and
// CREATE_NEW_PROCESS_GROUP fully detaches the child from the parent's job object.
func now(stop func()) {
	exe, err := os.Executable()
	if err != nil {
		stop()
		os.Exit(0)
	}

	argv0, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		stop()
		os.Exit(0)
	}

	var si syscall.StartupInfo
	var pi syscall.ProcessInformation

	// CREATE_NEW_PROCESS_GROUP — fully detach from parent job/console.
	const createNewProcessGroup = 0x00000200

	// 1. Spawn the new process BEFORE calling stop().
	err = syscall.CreateProcess(
		argv0, // lpApplicationName
		argv0, // lpCommandLine
		nil,   // lpProcessAttributes
		nil,   // lpThreadAttributes
		false, // bInheritHandles — do NOT inherit stdin/stdout/stderr
		createNewProcessGroup,
		nil, // lpEnvironment — inherit parent env
		nil, // lpCurrentDirectory — inherit parent cwd
		&si,
		&pi,
	)

	syscall.CloseHandle(pi.Thread)
	syscall.CloseHandle(pi.Process)

	// 2. Graceful shutdown — cancel context, let services finish.
	stop()

	if err != nil {
		// Failed to spawn — just exit, autostart will relaunch on next login.
		os.Exit(1)
	}

	// 3. Exit current process.
	os.Exit(0)
}
