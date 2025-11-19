package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// Usage:
//   minidocker run <rootfs> <command> <args...>
//   minidocker child <rootfs> <command> <args...>  (Internal use)
//   minidocker exec <container_pid> <command> <args...>
//   minidocker ps
//   minidocker stop <container_pid>

const stateDir = "./minidocker_state"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Ensure state directory exists
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		os.Mkdir(stateDir, 0755)
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	case "exec":
		execCmd()
	case "ps":
		ps()
	case "stop":
		stop()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: minidocker <command> [args...]")
	fmt.Println("Commands:")
	fmt.Println("  run <rootfs> <cmd>   Run a command in a new container")
	fmt.Println("  exec <pid> <cmd>     Run a command in an existing container")
	fmt.Println("  ps                   List running containers")
	fmt.Println("  stop <pid>           Stop a container")
}

func run() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: minidocker run <rootfs> <command> [args...]")
		os.Exit(1)
	}

	rootfs := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:]

	fmt.Printf("Starting container with rootfs: %s\n", rootfs)

	// Prepare the command to re-execute ourselves as "child"
	// We pass the same arguments to the child
	cmdArgs := append([]string{"child", rootfs, command}, args...)
	cmd := exec.Command("/proc/self/exe", cmdArgs...)

	// Set up namespaces
	// CLONE_NEWUTS: New hostname namespace
	// CLONE_NEWPID: New PID namespace (child becomes PID 1)
	// CLONE_NEWNS:  New Mount namespace
	// CLONE_NEWIPC: New IPC namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting container: %v\n", err)
		os.Exit(1)
	}

	// Save container state
	saveContainerState(cmd.Process.Pid, command)
	fmt.Printf("Container started with PID %d\n", cmd.Process.Pid)

	// Wait for the container to finish
	cmd.Wait()

	// Cleanup state
	removeContainerState(cmd.Process.Pid)
	fmt.Printf("Container %d stopped\n", cmd.Process.Pid)
}

func child() {
	if len(os.Args) < 4 {
		os.Exit(1)
	}

	rootfs := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:]

	fmt.Printf("Initializing container...\n")

	// 1. Hostname
	must(syscall.Sethostname([]byte("minidocker")))

	// 2. Chroot
	// We must chroot to the provided rootfs to isolate the filesystem
	must(syscall.Chroot(rootfs))
	must(syscall.Chdir("/"))

	// 3. Mount /proc
	// Essential for tools like 'ps' to work inside the container
	// We mount it at /proc. The directory must exist in the rootfs.
	must(syscall.Mount("proc", "/proc", "proc", 0, ""))

	// 4. Execute the command
	// We use syscall.Exec to replace the current process (PID 1 inside container)
	// with the user's command.
	
	// Look for the command in the PATH (inside the new root)
	cmdPath, err := exec.LookPath(command)
	if err != nil {
		// If not found in path, try using it directly
		cmdPath = command
	}

	if err := syscall.Exec(cmdPath, append([]string{command}, args...), os.Environ()); err != nil {
		fmt.Printf("Error executing command inside container: %v\n", err)
		os.Exit(1)
	}
}

func execCmd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: minidocker exec <pid> <command> [args...]")
		os.Exit(1)
	}

	pid := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:]

	// We use nsenter to enter the namespaces of the target PID
	// nsenter is a standard Linux tool usually available in Ubuntu
	// -t <pid>: target process
	// -m: enter mount namespace
	// -u: enter UTS namespace
	// -i: enter IPC namespace
	// -p: enter PID namespace
	
	nsArgs := []string{"-t", pid, "-m", "-u", "-i", "-p", command}
	nsArgs = append(nsArgs, args...)

	cmd := exec.Command("nsenter", nsArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command in container: %v\n", err)
		os.Exit(1)
	}
}

func ps() {
	files, err := ioutil.ReadDir(stateDir)
	if err != nil {
		fmt.Printf("Error reading state directory: %v\n", err)
		return
	}

	fmt.Println("PID\tCOMMAND")
	for _, file := range files {
		pid := file.Name()
		// Read the command from the file
		content, err := ioutil.ReadFile(filepath.Join(stateDir, pid))
		if err != nil {
			continue
		}
		fmt.Printf("%s\t%s\n", pid, string(content))
	}
}

func stop() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: minidocker stop <pid>")
		os.Exit(1)
	}

	pidStr := os.Args[2]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("Invalid PID: %v\n", err)
		os.Exit(1)
	}

	// Send SIGTERM
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		fmt.Printf("Error stopping container: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Sent SIGTERM to container %d\n", pid)
}

// Helpers

func must(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func saveContainerState(pid int, command string) {
	pidStr := strconv.Itoa(pid)
	filepath := filepath.Join(stateDir, pidStr)
	ioutil.WriteFile(filepath, []byte(command), 0644)
}

func removeContainerState(pid int) {
	pidStr := strconv.Itoa(pid)
	filepath := filepath.Join(stateDir, pidStr)
	os.Remove(filepath)
}
