# Mini Docker: Comprehensive Project Documentation

## 1. Introduction

**Mini Docker** is a lightweight, educational container runtime built from scratch in Go. It is designed to demonstrate the fundamental operating system concepts that power modern container engines like Docker, Kubernetes, and Podman.

Unlike full-scale container engines, this project focuses on simplicity and readability. It strips away complex networking and resource quotas to focus on the core mechanisms: **Process Isolation** and **Image Management**.

**Special Note on Environment:**
This project is specifically tailored to run inside **WSL (Windows Subsystem for Linux)**. Standard container runtimes often rely on features like `cgroups` (Control Groups) and `overlayfs` (Overlay Filesystems) which can be problematic or unsupported in basic WSL setups. Mini Docker works around these limitations to provide a functional "container-like" experience on Windows using standard Linux primitives.

---

## 2. Core Concepts (The Theory)

To understand what we are building, we must understand what a container actually *is*. It is **not** a virtual machine. It is a **process** on your computer that has been tricked into believing it is the only process running on a separate machine.

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
-   In Mini Docker, we tell the process: "Your root is now `/home/user/project/minidocker/containers/<id>`".
-   The process cannot see or access anything outside that folder.

### C. Image Layering (Hardlinks)
Real Docker uses `overlayfs` to layer images. Since WSL support for overlayfs can be tricky, we use **Hardlinks** (`cp -al`).
-   **Images** are stored as read-only templates in `./images/`.
-   **Containers** are created by recursively hardlinking the image files to `./containers/<id>/`.
-   This is fast and saves space (files share the same disk inode).
-   If a container modifies a file, the hardlink is broken (Copy-on-Write behavior is emulated via file replacement or explicit copy).

---

## 3. Project Structure

The project is organized as follows:

```text
minidocker/
├── main.go             # Entry point and CLI dispatch
├── images.go           # Image management logic (load, save, tag, inspect)
├── build.go            # MiniDockerfile parser and builder
├── setup_rootfs.sh     # Script to download base Alpine rootfs
├── images/             # Storage for image templates (read-only)
├── containers/         # Runtime storage for active containers (read-write)
├── minidocker_state/   # Stores PIDs of running containers
├── pushpa-app/         # Example user application
│   ├── MiniDockerfile  # Build definition
│   └── hello.sh        # Application script
└── DOCUMENTATION.md    # This file
```

---

## 4. Command Reference

All commands must be run with `sudo` as they require root privileges for namespaces and chroot.

### Container Lifecycle

| Command | Usage | Description |
|---------|-------|-------------|
| **run** | `minidocker run --image <name> [cmd]` | Creates and starts a new container from an image. |
| **exec** | `minidocker exec <pid> <cmd>` | Runs a command inside an *existing* running container. |
| **ps** | `minidocker ps` | Lists all currently running containers and their PIDs. |
| **stop** | `minidocker stop <pid>` | Sends SIGTERM to stop a running container. |

**Examples:**
```bash
# Run a container from the 'alpine' image
sudo ./minidocker run --image alpine /bin/sh

# Run a custom image
sudo ./minidocker run --image myhello:latest
```

### Image Management

| Command | Usage | Description |
|---------|-------|-------------|
| **build** | `minidocker image build <dir> <name>` | Builds an image from a `MiniDockerfile`. |
| **ls** | `minidocker image ls` | Lists all local images. |
| **tag** | `minidocker image tag <id> <name>` | Assigns a name (tag) to an image ID. |
| **inspect** | `minidocker image inspect <name>` | Shows image metadata and checks for validity. |
| **save** | `minidocker image save <name> <file>` | Exports an image to a `.tar` archive. |
| **load** | `minidocker image load <file>` | Imports an image from a `.tar` archive. |

**Examples:**
```bash
# Build an image
sudo ./minidocker image build ./pushpa-app my-app:v1

# Save an image to share it
sudo ./minidocker image save my-app:v1 my-app.tar

# Load an image on another machine
sudo ./minidocker image load my-app.tar
```

---

## 5. MiniDockerfile Reference

Mini Docker supports a simplified Dockerfile syntax called `MiniDockerfile`.

| Instruction | Description | Example |
|-------------|-------------|---------|
| **FROM** | The base image to start from. Must be the first line. | `FROM alpine` |
| **COPY** | Copies files from the host (build context) to the image. | `COPY hello.sh /bin/hello.sh` |
| **CMD** | The default command to run if none is specified at runtime. | `CMD /bin/hello.sh` |

**Example `MiniDockerfile`:**
```dockerfile
FROM alpine
COPY hello.sh /hello.sh
CMD /hello.sh
```

---

## 6. Technical Implementation Details

### The Build Process
1.  **Parse**: The `MiniDockerfile` is read line-by-line.
2.  **Base**: If `FROM alpine` is specified, the system looks for an image named `alpine` in `./images/`.
3.  **Clone**: The base image is copied (via hardlinks) to a new temporary image directory.
4.  **Copy**: Files specified in `COPY` are copied from the host into the new image directory.
5.  **Commit**: A `manifest.json` is written with the new ID and `CMD`. The image is registered in `images/index.json`.

### The Run Process
1.  **Clone**: The specified image is cloned from `./images/<id>` to `./containers/<new_id>` using hardlinks. This creates the container's "Root Filesystem".
2.  **Isolate**: The process re-executes itself with `CLONE_NEWPID`, `CLONE_NEWUTS`, etc.
3.  **Setup**:
    *   Hostname is set to `minidocker`.
    *   `chroot` is called to lock the process into `./containers/<new_id>`.
    *   `/proc` is mounted.
4.  **Execute**: The user's command (or the image's `CMD`) replaces the init process.

---

## 7. Tech Stack

*   **Language**: Go (Golang) 1.21+
    *   Chosen for its strong system programming capabilities and direct access to syscalls.
*   **Operating System**: Linux (Ubuntu via WSL2)
    *   Relies on Linux-specific kernel features (Namespaces, Chroot).
*   **Data Format**: JSON
    *   Used for image manifests and the image index.
*   **Archive Format**: Tar
    *   Used for saving and loading images (compatible with standard tools).

---

## 8. Future Improvements

To make this a production-grade container engine, we would need:
1.  **Cgroups**: To limit memory and CPU usage (e.g., "only use 512MB RAM").
2.  **OverlayFS**: To allow true copy-on-write layering instead of hardlinks.
3.  **Network Namespaces**: To give each container its own IP address and virtual network interface.
4.  **Registry Client**: To pull images directly from Docker Hub.

---

## 9. Conclusion

Mini Docker demonstrates that containers are not magic. They are a clever combination of Linux primitives that isolate processes from one another. By building this, you have looked under the hood of the technology that powers the modern cloud.
