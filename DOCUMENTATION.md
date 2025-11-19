# Mini Docker: Comprehensive Project Documentation

## 1. Introduction

**Mini Docker** is a lightweight, educational container runtime built from scratch in Go. It is designed to demonstrate the fundamental operating system concepts that power modern container engines like Docker, Kubernetes, and Podman.

Unlike full-scale container engines, this project focuses on simplicity and readability. It strips away complex networking, image layering, and resource quotas to focus on the core mechanism: **Process Isolation**.

**Special Note on Environment:**
This project is specifically tailored to run inside **WSL (Windows Subsystem for Linux)**. Standard container runtimes often rely on features like `cgroups` (Control Groups) and `overlayfs` (Overlay Filesystems) which can be problematic or unsupported in basic WSL setups. Mini Docker works around these limitations to provide a functional "container-like" experience on Windows.

---

## 2. What is a Container? (The Theory)

To understand what we are building, we must understand what a container actually *is*.

A container is **not** a real physical object or a virtual machine. It is a **process** on your computer that has been tricked into believing it is the only process running on a separate machine.

We achieve this "trick" using Linux Kernel features:

### A. Namespaces (Isolation)
Namespaces determine **what a process can see**.
-   **PID Namespace:** The process sees itself as PID 1 (the first process). It cannot see other processes on the host.
-   **UTS Namespace:** The process has its own hostname (e.g., `minidocker` instead of `ubuntu`).
-   **Mount Namespace:** The process has its own list of mounted filesystems.
-   **IPC Namespace:** The process has its own shared memory segments.

### B. Chroot (Filesystem Isolation)
`chroot` (Change Root) changes the root directory (`/`) for a process.
-   Normally, your root is `/`.
-   In Mini Docker, we tell the process: "Your root is now `/home/user/project/minidocker/alpine_rootfs`".
-   The process cannot see or access anything outside that folder.

### C. Cgroups (Resource Control) - *Omitted*
Real Docker uses Control Groups to say "This process can only use 50% CPU". We have omitted this for WSL compatibility and simplicity.

---

## 3. Project Structure

The project is organized as follows:

```text
minidocker/
├── main.go             # The Source Code (The Brain)
├── setup_rootfs.sh     # The Setup Script (The Builder)
├── alpine_rootfs/      # The Root Filesystem (The Body)
├── minidocker_state/   # State Directory (The Memory)
├── .gitignore          # Git Configuration
└── README.md           # Quick Start Guide
```

### `main.go`
This is the single Go file containing all logic. It handles:
1.  **CLI Parsing:** Reading commands like `run`, `exec`, `stop`.
2.  **Namespace Creation:** Using `syscall.SysProcAttr` to create new Linux namespaces.
3.  **Container Initialization:** The "child" process that sets up the environment (hostname, chroot, proc mount).

### `setup_rootfs.sh`
A bash script that downloads a minimal Alpine Linux distribution (about 5MB). This folder acts as the "hard drive" for our containers.

### `minidocker_state/`
A directory where we store the Process IDs (PIDs) of running containers. This allows commands like `ps` and `stop` to know which processes are containers.

---

## 4. How It Works: The Lifecycle of a Container

When you run `sudo ./minidocker run alpine_rootfs /bin/sh`, the following happens:

### Step 1: The Parent (CLI)
1.  The program starts.
2.  It parses the arguments.
3.  It prepares to run **itself** again, but this time with special flags:
    -   `CLONE_NEWUTS`
    -   `CLONE_NEWPID`
    -   `CLONE_NEWNS`
4.  It calls `/proc/self/exe child ...`. This is a trick called **re-execution**.

### Step 2: The Child (Container Init)
1.  The new process starts. Because of the flags, it is now in a new "world" (Namespace).
2.  **Hostname:** It sets the hostname to `minidocker`.
3.  **Chroot:** It changes its root directory to `alpine_rootfs`.
4.  **Directory Change:** It moves into the new `/`.
5.  **Mount /proc:** It mounts the special `proc` filesystem so tools like `ps` work inside.
6.  **Exec:** Finally, it replaces itself with the user's command (e.g., `/bin/sh`).

### Step 3: The Running Container
The user is now interacting with `/bin/sh` inside the isolated environment.

---

## 5. Key Functions Explained

### `run()`
Sets up the namespaces.
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | ...
}
```

### `child()`
Configures the inside of the container.
```go
syscall.Sethostname([]byte("minidocker"))
syscall.Chroot(rootfs)
syscall.Mount("proc", "/proc", "proc", 0, "")
```

### `execCmd()`
Allows entering an existing container using `nsenter`.
`nsenter` is a Linux tool that lets a process "jump" into the namespaces of another process.
```go
// Enters the Mount, UTS, IPC, and PID namespaces of the target PID
exec.Command("nsenter", "-t", pid, "-m", "-u", "-i", "-p", command)
```

---

## 6. Future Improvements

If you wanted to expand this project into a "Real" Docker, you would add:
1.  **Cgroups:** To limit memory and CPU usage.
2.  **OverlayFS:** To allow multiple containers to share the same base image without copying files.
3.  **Network Namespaces:** To give each container its own IP address (requires creating virtual ethernet bridges).
4.  **Image Registry:** A way to pull images from Docker Hub instead of using a local folder.

---

## 7. Conclusion

Mini Docker demonstrates that containers are not magic. They are a clever combination of Linux primitives that isolate processes from one another. By building this, you have looked under the hood of the technology that powers the modern cloud.
