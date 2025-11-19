# Mini Docker (Container Sandbox) for WSL

This is a simplified container runtime written in Go, designed to run inside **WSL (Windows Subsystem for Linux)**. It demonstrates core operating system concepts used in containerization, such as namespaces, chroot, and process isolation, without relying on features that are often problematic or unsupported in basic WSL environments (like cgroups or overlayfs).

## Features

- **PID Isolation:** The containerized process runs as PID 1 inside its own namespace.
- **Hostname Isolation:** The container has its own hostname (`minidocker`).
- **Filesystem Isolation:** Uses `chroot` to restrict the process to a specific directory (rootfs).
- **Mount Isolation:** Mounts a private `/proc` filesystem for the container.
- **IPC Isolation:** Isolated Inter-Process Communication resources.
- **Basic CLI:** `run`, `exec`, `ps`, `stop`.

## Prerequisites

- **WSL (Ubuntu)** installed on Windows.
- **Go** installed (`sudo apt install golang-go`).
- **Root privileges** (via `sudo`) are required to create namespaces and chroot.

## Installation

1.  Clone or create this project directory.
2.  Build the project:
    ```bash
    go build -o minidocker main.go
    ```

## Setting up a Root Filesystem (Rootfs)

Since we are not using overlayfs (which layers images), we need a directory that contains a full Linux filesystem tree. We can easily create one using `docker export` or `debootstrap`.

### Option 1: Using the provided script (Recommended)

I have provided a script `setup_rootfs.sh` that uses Docker to export a minimal Alpine Linux filesystem.

1.  Ensure you have Docker installed in WSL (or Docker Desktop connected to WSL).
2.  Run the script:
    ```bash
    chmod +x setup_rootfs.sh
    ./setup_rootfs.sh
    ```
    This will create a directory named `alpine_rootfs` in the current folder.

### Option 2: Manual Creation (using Docker)

```bash
# Pull a lightweight image
docker pull alpine

# Create a container (don't start it)
docker create --name temp_alpine alpine

# Export the filesystem to a tarball
docker export temp_alpine > alpine.tar

# Create a directory and extract it
mkdir alpine_rootfs
tar -xvf alpine.tar -C alpine_rootfs

# Cleanup
docker rm temp_alpine
rm alpine.tar
```

## Usage

**Note:** All commands must be run with `sudo` because creating namespaces requires root privileges.

### 1. Run a Container

Start a new container running `/bin/sh`.

```bash
sudo ./minidocker run alpine_rootfs /bin/sh
```

You will be dropped into a shell inside the container.
Try running `ps` or `hostname` inside to see the isolation.

### 2. List Running Containers

Open a new terminal window (in WSL) and run:

```bash
sudo ./minidocker ps
```

### 3. Execute a Command in a Running Container

If you have a container running (check `ps` for the PID), you can enter it:

```bash
# Replace <PID> with the actual PID from 'ps'
sudo ./minidocker exec <PID> /bin/sh
```

### 4. Stop a Container

```bash
sudo ./minidocker stop <PID>
```

## OS Concepts Explained

### Namespaces
Namespaces wrap a global system resource in an abstraction that makes it appear to the processes within the namespace that they have their own isolated instance of the global resource.

-   **PID Namespace (`CLONE_NEWPID`):**  Isolates the process ID number space. The process inside the container sees itself as PID 1, while on the host it has a different PID.
-   **UTS Namespace (`CLONE_NEWUTS`):**  Isolates the hostname and NIS domain name. Changing the hostname inside the container doesn't affect the host.
-   **Mount Namespace (`CLONE_NEWNS`):** Isolates the set of filesystem mount points. We use this to mount `/proc` inside the container without affecting the host's `/proc`.
-   **IPC Namespace (`CLONE_NEWIPC`):** Isolates System V IPC objects and POSIX message queues.

### Chroot
`chroot` (Change Root) changes the apparent root directory for the current running process and its children. A program that is run in such a modified environment cannot name (and therefore normally cannot access) files outside the designated directory tree. This provides the filesystem isolation.

### /proc Filesystem
The `/proc` filesystem is a pseudo-filesystem which provides an interface to kernel data structures. Tools like `ps` and `top` read from `/proc` to list processes. By mounting a new `/proc` instance inside our container (which is in a new PID namespace), `ps` will only show processes belonging to that namespace.

## Limitations (WSL Specific)

-   **No Cgroups:** We are not using Control Groups (cgroups) to limit resource usage (CPU/Memory) because cgroup v2 support in WSL can be complex to configure manually without systemd or specific kernel options.
-   **No OverlayFS:** We use a simple directory as the rootfs. Real Docker uses OverlayFS to layer images efficiently.
-   **No Network Namespace:** The container shares the host's network stack. Implementing network isolation in WSL requires complex bridge setup which is out of scope for this mini-project.
