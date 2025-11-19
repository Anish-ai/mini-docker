# New Features: Image Management System

This document outlines the newly added Image Management System for MiniDocker, enabling users to build, share, and manage container images similar to standard Docker.

## Overview

The new update introduces a complete lifecycle for container images:
1.  **Build** images from a `MiniDockerfile`.
2.  **Manage** images (list, tag, inspect).
3.  **Share** images (save to/load from tarballs).
4.  **Run** containers directly from named images.

## New Commands

### 1. Image Management (`minidocker image <subcommand>`)

| Command | Usage | Description |
|---------|-------|-------------|
| **build** | `minidocker image build <dir> <name>` | Builds a new image from a `MiniDockerfile` in the specified directory. |
| **ls** | `minidocker image ls` | Lists all available images, their IDs, and default commands. |
| **tag** | `minidocker image tag <id> <name>` | Assigns a human-readable name (tag) to an image ID. |
| **inspect** | `minidocker image inspect <name>` | Shows details about an image (ID, Name, CMD) and checks for validity. |
| **save** | `minidocker image save <name> <file.tar>` | Exports an image to a tarball for sharing. |
| **load** | `minidocker image load <file.tar>` | Imports an image from a tarball. |

### 2. Running Containers

You can now run containers using an image name instead of a raw rootfs path.

```bash
sudo ./minidocker run --image <image_name> [command]
```

- **Example**: `sudo ./minidocker run --image myhello:latest`
- If `[command]` is omitted, the default `CMD` from the image is executed.

## MiniDockerfile Support

You can now define images using a `MiniDockerfile`. Supported instructions:

- **`FROM <image>`**: Base image to start from (e.g., `alpine`).
- **`COPY <src> <dest>`**: Copy files from the host context to the container.
- **`CMD <command>`**: Set the default command to run.

**Example `MiniDockerfile`:**
```dockerfile
FROM alpine
COPY hello.sh /hello.sh
CMD /hello.sh
```

## Technical Implementation (How it Works)

### Image Storage
Images are stored in `./images/<image_id>/`. Each image directory contains:
- The full root filesystem (extracted).
- A `manifest.json` file containing metadata (ID, Name, Default Command).

### Layering Strategy
When building or running a container:
1.  **Build**: The base image is copied to the new image directory using `cp -al` (hardlinks). This saves disk space and is fast. New files are then copied on top.
2.  **Run**: A new container rootfs is created in `./containers/<id>/` by recursively hardlinking the image rootfs. This acts as a primitive "Copy-on-Write" mechanism.

### Portability
The `save` and `load` commands use standard `tar` archives, making it easy to move images between different MiniDocker instances.

## Use Cases

1.  **Application Packaging**: Package your application and its dependencies (scripts, binaries) into a single named entity.
2.  **Versioning**: Use tags (e.g., `v1.0`, `latest`) to manage different versions of your environment.
3.  **Distribution**: Build an image on one machine, `save` it to a file, and `load` it on another machine.
4.  **Reproducibility**: Ensure every container runs with the exact same filesystem and configuration defined in the `MiniDockerfile`.
