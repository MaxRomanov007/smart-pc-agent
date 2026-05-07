//go:build windows

package restart

import (
	"os"
	"syscall"
)

// now stops the application via stop(), then spawns a detached copy of the
// binary and exits.
//
// exec.Command is avoided because the agent is built with -H windowsgui and
// has no console — inheriting Stdin/Stdout/Stderr causes errors on some
// Windows versions. Instead syscall.CreateProcess is used directly with
// bInheritHandles=false and CREATE_NEW_PROCESS_GROUP so the child is fully
// detached from the parent's job object.
//
// The HKCU Run registry value written by the Inno Setup installer points to
// the same executable path, so autostart is unaffected.
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

	stop()

	var si syscall.StartupInfo
	var pi syscall.ProcessInformation

	// CREATE_NEW_PROCESS_GROUP — fully detach from parent job/console.
	const createNewProcessGroup = 0x00000200

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
	if err != nil {
		os.Exit(0)
	}

	syscall.CloseHandle(pi.Thread)
	syscall.CloseHandle(pi.Process)

	os.Exit(0)
}
