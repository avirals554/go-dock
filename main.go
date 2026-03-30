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
		syscall.Chdir("/")
		os.MkdirAll("/sys/fs/cgroup", 0755)

		syscall.Mount("proc", "/proc", "proc", 0, "")
		syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, "")
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

		err := cmd.Start()
		id := cmd.Process.Pid
		pidstr := fmt.Sprintf("%d", id)
		os.WriteFile("/sys/fs/cgroup/mycontainer/cgroup.procs", []byte(pidstr), 0700)
		os.WriteFile("/sys/fs/cgroup/mycontainer/memory.max", []byte("10485760"), 0700)

		err = cmd.Wait()
		if err != nil {
			panic(err)
		}

		if err != nil {
			fmt.Println("there was some error ")
		}

	}
}
