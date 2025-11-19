package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

// Build a minimal image from a simple Dockerfile-like syntax. Supported directives:
// FROM <image>
// COPY <src> <dst>
// CMD <command...>
//
// Usage: minidocker image build <context-dir> <name>
func imageBuild(contextDir, name string) (string, error) {
    if err := ensureDirs(); err != nil {
        return "", err
    }

    // find Dockerfile (MiniDockerfile or Dockerfile)
    df := filepath.Join(contextDir, "MiniDockerfile")
    if _, err := os.Stat(df); os.IsNotExist(err) {
        df = filepath.Join(contextDir, "Dockerfile")
        if _, err2 := os.Stat(df); os.IsNotExist(err2) {
            return "", fmt.Errorf("no MiniDockerfile or Dockerfile in context")
        }
    }

    f, err := os.Open(df)
    if err != nil {
        return "", err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    var base string
    var copies [][2]string
    var cmdline string

    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.Fields(line)
        if len(parts) == 0 {
            continue
        }
        switch strings.ToUpper(parts[0]) {
        case "FROM":
            if len(parts) < 2 {
                return "", fmt.Errorf("FROM requires image name")
            }
            base = parts[1]
        case "COPY":
            if len(parts) < 3 {
                return "", fmt.Errorf("COPY requires src and dst")
            }
            copies = append(copies, [2]string{parts[1], parts[2]})
        case "CMD":
            if len(parts) < 2 {
                return "", fmt.Errorf("CMD requires command")
            }
            cmdline = strings.Join(parts[1:], " ")
        default:
            return "", fmt.Errorf("unsupported directive: %s", parts[0])
        }
    }
    if err := scanner.Err(); err != nil {
        return "", err
    }

    // prepare image dir
    id := genID()
    imgDir := filepath.Join(imagesDir, id)
    if err := os.MkdirAll(imgDir, 0755); err != nil {
        return "", err
    }

    // If base provided, copy its rootfs into imgDir
    if base != "" {
        baseDir, err := findImageDirByName(base)
        if err != nil {
            return "", fmt.Errorf("base image %s not found: %w", base, err)
        }
        // try cp -al baseDir/. imgDir
        // Note: filepath.Join removes trailing dots, so we append "/." manually
        // to ensure cp copies contents, not the directory itself.
        cp := exec.Command("cp", "-al", baseDir+"/.", imgDir)
        if err := cp.Run(); err != nil {
            // fallback to rsync
            rsync := exec.Command("rsync", "-a", baseDir+"/", imgDir+"/")
            if err := rsync.Run(); err != nil {
                return "", fmt.Errorf("failed cloning base image: %v / %v", err, rsync.Run())
            }
        }
    }

    // Apply COPY commands
    for _, c := range copies {
        src := filepath.Join(contextDir, c[0])
        dst := filepath.Join(imgDir, c[1])
        // ensure parent
        if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
            return "", err
        }
        // use cp -a
        cp := exec.Command("cp", "-a", src, dst)
        cp.Stdout = os.Stdout
        cp.Stderr = os.Stderr
        if err := cp.Run(); err != nil {
            return "", fmt.Errorf("COPY failed %s -> %s: %v", src, dst, err)
        }
    }

    // write manifest
    im := Image{ID: id, Name: name, Cmd: cmdline}
    mf, err := os.Create(filepath.Join(imgDir, "manifest.json"))
    if err != nil {
        return "", err
    }
    enc := json.NewEncoder(mf)
    enc.SetIndent("", "  ")
    if err := enc.Encode(im); err != nil {
        mf.Close()
        return "", err
    }
    mf.Close()

    // register in index
    idx, _ := loadIndex()
    // if an image with the same name exists, remove it (replace behavior)
    for i := len(idx.Images) - 1; i >= 0; i-- {
        if idx.Images[i].Name == name {
            // remove old image files and index entry
            _ = removeIndexEntry(idx, i)
            break
        }
    }
    idx.Images = append(idx.Images, im)
    if err := saveIndex(idx); err != nil {
        return "", err
    }

    return id, nil
}
