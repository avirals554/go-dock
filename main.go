package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
)

var image = map[string]string{
	"alpine":  "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz",
	"alpine3": "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.0-x86_64.tar.gz",
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("couldn't find home path")
	}
	basePath := home + "/.go-dock"

	if len(os.Args) < 2 {
		fmt.Println(`
go-dock - a container runtime

Usage:
  go-dock <command> [arguments]

Commands:
  run <image>    run a container with the given image

Examples:
  go-dock run alpine
`)
		return
	}

	switch os.Args[1] {
	case "run":
		imageName := os.Args[2]
		rootfsPath := basePath + "/images/" + imageName
		if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
			fmt.Println("image not found, run: go-dock pull", imageName)
			return
		}
		cmd := exec.Command("/proc/self/exe", "child", rootfsPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
		}
		if err := cmd.Start(); err != nil {
			fmt.Println("failed to start container:", err)
			return
		}
		id := cmd.Process.Pid
		pidStr := fmt.Sprintf("%d", id)
		os.WriteFile("/sys/fs/cgroup/mycontainer/cgroup.procs", []byte(pidStr), 0700)
		os.WriteFile("/sys/fs/cgroup/mycontainer/memory.max", []byte("10485760"), 0700)
		err := cmd.Wait()
		if err != nil {
			fmt.Println("container exited with error:", err)
		}

	case "child":
		rootfsPath := os.Args[2]
		if err := syscall.Chroot(rootfsPath); err != nil {
			fmt.Println("chroot failed:", err)
			return
		}
		if err := syscall.Chdir("/"); err != nil {
			fmt.Println("chdir failed:", err)
			return
		}
		os.MkdirAll("/sys/fs/cgroup", 0755)
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			fmt.Println("mount proc failed:", err)
			return
		}
		if err := syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, ""); err != nil {
			fmt.Println("mount cgroup failed:", err)
			return
		}
		cmd := exec.Command("/bin/sh")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("shell error:", err)
		}
	case "pull":
		image_url, ok := image[os.Args[2]]
		if ok {
			fmt.Println("found the image ")
		} else {
			fmt.Println("the image was not found at all ")
			return
		}
		raw_image, _ := http.Get(image_url)
		gzReader, err := gzip.NewReader(raw_image.Body)
		if err != nil {
			fmt.Println("zip extraction failed ")
		}
		defer gzReader.Close()
		tarReader := tar.NewReader(gzReader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			dest_path := basePath + "/images/" + os.Args[2] + "/" + header.Name
			switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(dest_path, 0755)
			case tar.TypeReg:
				file, _ := os.Create(dest_path)
				io.Copy(file, tarReader)
				file.Close()

			}

		}

	default:
		fmt.Println("unknown command:", os.Args[1])
	}
}
