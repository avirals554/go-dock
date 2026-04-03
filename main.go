package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type containers struct {
	ID        string
	ImageName string
	PID       int
	StartTime string
	Status    string
}

var image = map[string]string{
	"alpine":  "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz",
	"alpine3": "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.0-x86_64.tar.gz",
}

func updateprocess(path string) {
	data, _ := os.ReadFile(path)
	var c containers
	json.Unmarshal(data, &c)
	c.Status = "DEAD"
	newData, _ := json.Marshal(c)
	os.WriteFile(path, newData, 0644)
}
func createcontainer(image string, pid int, path string) string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	os.MkdirAll(path+"/containers/"+id, 0755)
	process_status := " "

	process, _ := os.FindProcess(pid)
	err := process.Signal(syscall.Signal(0))
	if err != nil {
		process_status = "DEAD"
	}
	if err == nil {
		process_status = "ALIVE"
	}
	container := containers{ID: id, ImageName: image, PID: pid, StartTime: time.Now().String(), Status: process_status}
	container_data, _ := json.Marshal(container)
	os.WriteFile(path+"/containers/"+id+"/config.json", container_data, 0644)
	return id

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
		createcontainer(os.Args[2], id, basePath)
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
				os.Chmod(dest_path, header.FileInfo().Mode())
				file.Close()
			case tar.TypeSymlink:
				os.Symlink(header.Linkname, dest_path)

			}
		}
	case "ps":
		fmt.Printf("%-20s %-10s %-10s %s\n", "ID", "IMAGE", "STATUS", "STARTED")
		entries, _ := os.ReadDir(basePath + "/containers/")
		for _, entry := range entries {
			// entry.Name() gives the folder name
			data, _ := os.ReadFile(basePath + "/containers/" + entry.Name() + "/config.json")
			var c containers
			json.Unmarshal(data, &c)
			fmt.Printf("%-20s %-10s %-10s %s\n", c.ID[:12], c.ImageName, c.Status, c.StartTime[:19])

		}
	case "kill":
		var c containers
		data, _ := os.ReadFile(basePath + "/containers/" + os.Args[2] + "/config.json")
		json.Unmarshal(data, &c)
		if err := syscall.Kill(c.PID, syscall.SIGKILL); err != nil {
			fmt.Println("kill failed :", err)
			return
		}

		updateprocess(os.Args[2], basePath+"/containers/"+os.Args[2]+"/config.json")
		fmt.Println("kill sucessful:", os.Args[2])

	default:
		fmt.Println("unknown command:", os.Args[1])
	}
}
