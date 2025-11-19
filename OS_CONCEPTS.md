# Mini Docker: Operating Systems Coursework Connection

This document maps the technical implementation of **Mini Docker** to core concepts taught in Operating Systems (OS) courses. Use this guide to structure your presentation and demonstrate how this project applies theoretical OS knowledge in practice.

---

## 1. Processes and Process Management

**Course Topic:** Process Lifecycle, PCB (Process Control Block), Context Switching.

*   **Project Application:**
    *   **Process Creation:** The project uses the `clone()` system call (via Go's `exec.Command` and `SysProcAttr`) to create new processes. This is a more granular version of the standard `fork()` + `exec()` model.
    *   **PID (Process ID):** We explicitly manipulate PIDs. The containerized process believes it is **PID 1** (the `init` process) due to the *PID Namespace*, while the host OS sees it as a regular process with a high PID (e.g., 10455).
    *   **Process Termination:** The `stop` command demonstrates sending **Signals** (`SIGTERM`) to a specific PID to request termination, a fundamental IPC mechanism.
    *   **Zombie Processes:** The `cmd.Wait()` function in `main.go` ensures the parent waits for the child to finish, preventing zombie processes (entries in the process table that have finished execution but haven't been collected).

## 2. System Calls (The Kernel Interface)

**Course Topic:** User Mode vs. Kernel Mode, The System Call Interface.

*   **Project Application:**
    *   Mini Docker does not use high-level libraries to create containers; it talks directly to the Linux Kernel using **System Calls**.
    *   **Key Syscalls Used:**
        *   `syscall.SysProcAttr`: Configures attributes for the new process (specifically `Cloneflags`).
        *   `syscall.Sethostname`: Changes the system hostname for the isolated process.
        *   `syscall.Chroot`: Changes the root directory path resolution.
        *   `syscall.Mount`: Attaches the `/proc` pseudo-filesystem.
        *   `syscall.Exec`: Replaces the current process image with a new program (e.g., `/bin/sh`).

## 3. Filesystems and I/O

**Course Topic:** Directory Structure, Inodes, VFS (Virtual File System), Mounting.

*   **Project Application:**
    *   **Root Filesystem (rootfs):** We demonstrate that "root" (`/`) is just a pointer. By changing this pointer using `chroot`, we restrict the process's file access to a specific subdirectory (`./containers/<id>`).
    *   **Inodes & Hardlinks:** Our image layering strategy uses `cp -al`. This creates **Hardlinks**, meaning the file in the image and the file in the container point to the exact same **Inode** on the disk. This saves space and demonstrates how the OS manages file metadata separate from file content.
    *   **Pseudo-Filesystems:** We mount `/proc`. This is not a real disk filesystem but a kernel interface exposed as files. Reading `/proc/1/status` reads kernel memory structures about process 1. This demonstrates the "Everything is a File" philosophy of Unix/Linux.

## 4. Virtualization and Isolation

**Course Topic:** Virtual Machines vs. Containers, Resource Isolation.

*   **Project Application:**
    *   **Namespaces:** This is the modern OS approach to virtualization. Instead of simulating hardware (like a VM), we partition kernel resources.
        *   **PID Namespace:** Virtualizes the Process ID counter.
        *   **UTS Namespace:** Virtualizes system identifiers (Hostname).
        *   **Mount Namespace:** Virtualizes the mount table (what disks/folders are visible).
    *   **Comparison:** You can explain how this is lighter than a VM because there is no Hypervisor and no Guest OS kernel. It's just a standard process with "blinders" on.

---

## Key Terms for Your Presentation

| Term | OS Definition | Usage in Project |
|------|---------------|------------------|
| **Namespace** | A kernel feature that partitions kernel resources such that one set of processes sees one set of resources while another set of processes sees a different set. | Used to give the container its own PID 1, Hostname, and Mounts. |
| **Chroot** | An operation that changes the apparent root directory for the current running process and its children. | Used to trap the container inside the `alpine_rootfs` folder. |
| **Syscall** | The programmatic way in which a computer program requests a service from the kernel of the operating system. | `syscall.Mount`, `syscall.Exec`, etc., are the building blocks of the runtime. |
| **PID 1 (Init)** | The first process started by the kernel during booting. It is responsible for starting other processes. | Inside our container, the shell (`/bin/sh`) thinks it is PID 1. |
| **Hardlink** | A directory entry that associates a name with a file on a file system. Multiple hardlinks point to the same Inode. | Used to "copy" images to containers instantly without duplicating data. |
| **Signal** | A limited form of inter-process communication used in Unix-like operating systems. | `SIGTERM` is used by the `stop` command to tell the container to exit. |

---

## Presentation Slide Outline Ideas

### Slide 1: Project Overview
*   **Title:** Mini Docker: A Linux Container Runtime from Scratch.
*   **Goal:** Demystifying "Container Magic" using OS Primitives.
*   **Tech Stack:** Go (Golang), Linux Kernel API.

### Slide 2: The "What is a Container?" Question
*   It's not a Virtual Machine.
*   It's a **Process** with restricted views.
*   Relies on **Namespaces** (Visibility) and **Chroot** (Access).

### Slide 3: OS Concept - Process Isolation (Namespaces)
*   **Theory:** Processes usually see all other processes.
*   **Implementation:** `CLONE_NEWPID` flag.
*   **Result:** Container sees itself as PID 1. Host sees it as PID 12345.

### Slide 4: OS Concept - Filesystem Isolation (Chroot)
*   **Theory:** The Root Directory (`/`) is the top of the tree.
*   **Implementation:** `syscall.Chroot("./containers/my-container")`.
*   **Result:** The process cannot `cd ..` above its new root. It is "jailed".

### Slide 5: OS Concept - Efficient Storage (Inodes)
*   **Problem:** Copying 100MB images for every container is slow.
*   **OS Solution:** **Hardlinks**.
*   **Implementation:** `cp -al`. Files share the same **Inode**.
*   **Benefit:** Instant container creation, near-zero disk usage overhead.

### Slide 6: Demo & Conclusion
*   Show `minidocker run`.
*   Show `ps` inside vs outside (PID differences).
*   **Takeaway:** Containers are just a clever application of standard Operating System features.
