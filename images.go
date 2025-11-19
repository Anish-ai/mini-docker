package main

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

const imagesDir = "./images"
const containersDir = "./containers"
const indexFile = "./images/index.json"

type Image struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Cmd  string `json:"cmd"`
}

type imageIndex struct {
    Images []Image `json:"images"`
}

func ensureDirs() error {
    if err := os.MkdirAll(imagesDir, 0755); err != nil {
        return err
    }
    if err := os.MkdirAll(containersDir, 0755); err != nil {
        return err
    }
    return nil
}

func genID() string {
    b := make([]byte, 8)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

func imageLoad(tarfile string) (string, error) {
    if err := ensureDirs(); err != nil {
        return "", err
    }

    id := genID()
    dest := filepath.Join(imagesDir, id)
    if err := os.MkdirAll(dest, 0755); err != nil {
        return "", err
    }

    // Extract tar to dest
    // tar -xvf <tarfile> -C <dest>
    cmd := exec.Command("tar", "-xvf", tarfile, "-C", dest)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("tar extract failed: %w", err)
    }

    // look for manifest.json or manifest in extracted root
    manifestPath := filepath.Join(dest, "manifest.json")
    var im Image
    if _, err := os.Stat(manifestPath); err == nil {
        f, err := os.Open(manifestPath)
        if err == nil {
            defer f.Close()
            dec := json.NewDecoder(f)
            _ = dec.Decode(&im)
        }
    }

    if im.ID == "" {
        im.ID = id
    }
    if im.Name == "" {
        im.Name = im.ID
    }

    // save manifest into images/<id>/manifest.json
    mf, _ := os.Create(filepath.Join(dest, "manifest.json"))
    defer mf.Close()
    enc := json.NewEncoder(mf)
    enc.SetIndent("", "  ")
    _ = enc.Encode(im)

    // register in index
    idx, _ := loadIndex()
    idx.Images = append(idx.Images, im)
    saveIndex(idx)

    return im.ID, nil
}

func imageSave(name, outfile string) error {
    // find image by name
    idx, err := loadIndex()
    if err != nil {
        return err
    }
    var id string
    for _, im := range idx.Images {
        if im.Name == name || strings.HasPrefix(im.Name, name) || im.ID == name {
            id = im.ID
            break
        }
    }
    if id == "" {
        return fmt.Errorf("image not found: %s", name)
    }
    src := filepath.Join(imagesDir, id)
    // Create tar: tar -cvf <outfile> -C <src> .
    cmd := exec.Command("tar", "-cvf", outfile, "-C", src, ".")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func loadIndex() (*imageIndex, error) {
    idx := &imageIndex{}
    if _, err := os.Stat(indexFile); os.IsNotExist(err) {
        return idx, nil
    }
    f, err := os.Open(indexFile)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    dec := json.NewDecoder(f)
    if err := dec.Decode(idx); err != nil && err != io.EOF {
        return nil, err
    }
    return idx, nil
}

func saveIndex(idx *imageIndex) error {
    f, err := os.Create(indexFile)
    if err != nil {
        return err
    }
    defer f.Close()
    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    return enc.Encode(idx)
}

func listImages() ([]Image, error) {
    idx, err := loadIndex()
    if err != nil {
        return nil, err
    }
    return idx.Images, nil
}

func findImageDirByName(name string) (string, error) {
    idx, err := loadIndex()
    if err != nil {
        return "", err
    }
    // prefer newest images (search in reverse)
    for i := len(idx.Images) - 1; i >= 0; i-- {
        im := idx.Images[i]
        if im.Name == name || im.ID == name || strings.HasPrefix(im.Name, name) {
            return filepath.Join(imagesDir, im.ID), nil
        }
    }
    return "", fmt.Errorf("image not found: %s", name)
}

func loadImageByName(name string) (*Image, error) {
    idx, err := loadIndex()
    if err != nil {
        return nil, err
    }
    // prefer newest images
    for i := len(idx.Images) - 1; i >= 0; i-- {
        im := idx.Images[i]
        if im.Name == name || im.ID == name || strings.HasPrefix(im.Name, name) {
            return &im, nil
        }
    }
    return nil, fmt.Errorf("image not found: %s", name)
}

// removeIndexEntry removes an image entry by index position and deletes its directory
func removeIndexEntry(idx *imageIndex, pos int) error {
    if pos < 0 || pos >= len(idx.Images) {
        return nil
    }
    id := idx.Images[pos].ID
    // remove directory
    _ = os.RemoveAll(filepath.Join(imagesDir, id))
    // remove from slice
    idx.Images = append(idx.Images[:pos], idx.Images[pos+1:]...)
    return nil
}

// imageTag adds or updates a human-friendly name for an image
func imageTag(idOrName, newName string) error {
    idx, err := loadIndex()
    if err != nil {
        return err
    }
    for i, im := range idx.Images {
        if im.ID == idOrName || im.Name == idOrName || strings.HasPrefix(im.ID, idOrName) {
            idx.Images[i].Name = newName
            // update manifest file
            manifestPath := filepath.Join(imagesDir, im.ID, "manifest.json")
            f, err := os.Create(manifestPath)
            if err == nil {
                enc := json.NewEncoder(f)
                enc.SetIndent("", "  ")
                _ = enc.Encode(idx.Images[i])
                f.Close()
            }
            return saveIndex(idx)
        }
    }
    return fmt.Errorf("image not found: %s", idOrName)
}

// imageInspect returns the manifest for an image and checks common files
func imageInspect(idOrName string) (*Image, error) {
    im, err := loadImageByName(idOrName)
    if err != nil {
        return nil, err
    }
    // check common runtime files (/bin/sh and /proc dir)
    imgDir := filepath.Join(imagesDir, im.ID)
    shPath := filepath.Join(imgDir, "bin", "sh")
    procPath := filepath.Join(imgDir, "proc")
    // annotate command if missing
    info := *im
    if _, err := os.Stat(shPath); os.IsNotExist(err) {
        info.Cmd = info.Cmd + " (warning: /bin/sh missing)"
    }
    if _, err := os.Stat(procPath); os.IsNotExist(err) {
        info.Cmd = info.Cmd + " (warning: /proc missing)"
    }
    return &info, nil
}

// cloneImageRootfs clones image rootfs into a new container directory using hardlinks when possible
func cloneImageRootfs(name string) (string, error) {
    if err := ensureDirs(); err != nil {
        return "", err
    }
    srcDir, err := findImageDirByName(name)
    if err != nil {
        return "", err
    }
    // image rootfs usually inside image dir at ./ (we extracted directly there)
    srcRoot := srcDir
    // create dest
    cid := genID()
    dest := filepath.Join(containersDir, cid)
    if err := os.MkdirAll(dest, 0755); err != nil {
        return "", err
    }

    // try cp -al srcRoot/. dest/
    // Note: filepath.Join removes trailing dots, so we append "/." manually
    cp := exec.Command("cp", "-al", srcRoot+"/.", dest)
    if err := cp.Run(); err == nil {
        return dest, nil
    }

    // fallback to rsync -a
    rsync := exec.Command("rsync", "-a", srcRoot+"/", dest+"/")
    if err := rsync.Run(); err == nil {
        return dest, nil
    }

    // last fallback: tar pipe
    tar1 := exec.Command("tar", "-cf", "-", "-C", srcRoot, ".")
    tar2 := exec.Command("tar", "-xf", "-", "-C", dest)
    r, w := io.Pipe()
    tar1.Stdout = w
    tar2.Stdin = r
    tar1.Stderr = os.Stderr
    tar2.Stderr = os.Stderr
    if err := tar1.Start(); err != nil {
        return "", err
    }
    if err := tar2.Start(); err != nil {
        return "", err
    }
    tar1.Wait()
    w.Close()
    tar2.Wait()
    return dest, nil
}
