package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if os.Args[1] == "child" {
		if err := syscall.Chroot("/mycontainer/rootfs"); err != nil {
			fmt.Println("there was a problem with the chroot syscall ")
			return
		}
		if err := syscall.Chdir("/"); err != nil {
			fmt.Println(" there was a problem with the chdir syscall ")
			return
		}
		os.MkdirAll("/sys/fs/cgroup", 0755)

		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			fmt.Println("there was a problem with the mount syscall ")
			return
		}
		if err := syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, ""); err != nil {
			fmt.Println("there was an error with the 2nd mount syscall ")
			return
		}
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

		if err := cmd.Start(); err != nil {
			fmt.Println("there was an error with the command start thing ")
			return
		}
		id := cmd.Process.Pid
		pidstr := fmt.Sprintf("%d", id)
		os.WriteFile("/sys/fs/cgroup/mycontainer/cgroup.procs", []byte(pidstr), 0700)
		os.WriteFile("/sys/fs/cgroup/mycontainer/memory.max", []byte("10485760"), 0700)

		err := cmd.Wait()
		if err != nil {
			panic(err)
		}

		if err != nil {
			fmt.Println("there was some error ")
		}

	}
}
