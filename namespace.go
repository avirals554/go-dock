package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/vishvananda/netlink"
)

func child(argument string) {
	rootfsPath := argument
	//giving the path to the chroot
	if err := syscall.Chroot(rootfsPath); err != nil {
		fmt.Println("chroot failed:", err)
		return
	}
	if err := syscall.Mkdir("home", 0755); err != nil {
		fmt.Println("root error", err)
		return
	}
	if err := syscall.Mkdir("home/user", 0755); err != nil {
		fmt.Println("root error", err)
		return
	}
	if err := syscall.Chdir("/home/user"); err != nil {
		fmt.Println("chdir failed:", err)
		return
	}
	os.WriteFile("/etc/resolv.conf", []byte("nameserver 8.8.8.8\nnameserver 1.1.1.1\n"), 0644) // this is for addding the dns look up this checks the namesperver and finds the ip this has 2 nameservers one of google and the other of cloudfare
	//creating a mount for the cgroup cause alpine doesnt have it ---
	os.MkdirAll("/sys/fs/cgroup", 0755)
	//mounting a new proc for the new process namespace that is to be creates
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Println("mount proc failed:", err)
		return
	}
	// Mount cgroup2 so container processes can see their resource limit---
	if err := syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, ""); err != nil {
		fmt.Println("mount cgroup failed:", err)
		return
	}
	veth1, _ := netlink.LinkByName("veth1")
	netlink.LinkSetUp(veth1)
	addr, _ := netlink.ParseAddr("192.168.1.2/24")
	netlink.AddrAdd(veth1, addr)
	route := &netlink.Route{
		Gw: net.ParseIP("192.168.1.1"),
	}
	netlink.RouteAdd(route)
	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("shell error:", err)
	}
}

//there are a few problems with this --
// 1. what if we want to make more than one containers ?
//  that would mean that every time we would create the same pair of veth right ? and that is a problem cause which is which ?
//
