#!/bin/bash

# Setup script for Mini Docker Rootfs
# This script downloads a minimal Alpine Linux rootfs and extracts it.

set -e

ROOTFS_DIR="alpine_rootfs"
ALPINE_VERSION="3.18.4"
ALPINE_ARCH="x86_64"
DOWNLOAD_URL="https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/${ALPINE_ARCH}/alpine-minirootfs-${ALPINE_VERSION}-${ALPINE_ARCH}.tar.gz"
TARBALL="alpine-minirootfs.tar.gz"

if [ -d "$ROOTFS_DIR" ]; then
    echo "Directory $ROOTFS_DIR already exists. Skipping setup."
    exit 0
fi

echo "Creating rootfs directory..."
mkdir -p "$ROOTFS_DIR"

echo "Downloading Alpine Linux minimal rootfs..."
if command -v curl >/dev/null 2>&1; then
    curl -L -o "$TARBALL" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$TARBALL" "$DOWNLOAD_URL"
else
    echo "Error: Neither curl nor wget found. Please install one of them."
    exit 1
fi

echo "Extracting rootfs..."
tar -xvf "$TARBALL" -C "$ROOTFS_DIR" > /dev/null

echo "Cleaning up..."
rm "$TARBALL"

echo "Success! Rootfs created at ./$ROOTFS_DIR"
echo "You can now run: sudo ./minidocker run $ROOTFS_DIR /bin/sh"
