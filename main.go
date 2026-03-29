package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if os.Args[1] == "child" {
		syscall.Chroot("/mycontainer/rootfs")
		syscall.Mount("proc", "/proc", "proc", 0, "")
		cmd := exec.Command("/bin/sh")

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Println("there was some error ")
		}
	} else {
		cmd := exec.Command("/proc/self/exe", "child")

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
		}

		err := cmd.Run()
		if err != nil {
			fmt.Println("there was some error ")
		}

	}
}
