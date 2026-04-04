package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	// fmt.Println("the update process function is being called ")  -- no longer needed , it was made just for checking
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
		run(os.Args[2], basePath)

	case "child":
		child(os.Args[2])
	case "pull":

		Pull(os.Args[2], basePath)
	case "ps":
		ps(basePath)
	case "kill":
		kill(basePath)

	default:
		fmt.Println("unknown command:", os.Args[1])
		fmt.Println(`
go-dock - a container runtime

Usage:
  go-dock <command> [arguments]

Commands:
  run <image>    run a container with the given image

Examples:
  go-dock run alpine
`)
	}
}
