package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/vishvananda/netlink"
)

func networking(id int) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: "veth0",
		},
		PeerName: "veth1",
	}
	netlink.LinkAdd(veth)                   // create pair first
	veth0, _ := netlink.LinkByName("veth0") // then get veth0
	veth1, _ := netlink.LinkByName("veth1") // and veth1
	netlink.LinkSetNsPid(veth1, id)         // move veth1 into container
	netlink.LinkSetUp(veth0)                // bring veth0 up
	addr, _ := netlink.ParseAddr("192.168.1.1/24")
	netlink.AddrAdd(veth0, addr) // assign IP to veth0

}
func kill(basePath string) {
	updateprocess(basePath + "/containers/" + os.Args[2] + "/config.json")
	var c containers
	data, _ := os.ReadFile(basePath + "/containers/" + os.Args[2] + "/config.json")
	json.Unmarshal(data, &c)
	if err := syscall.Kill(c.PID, syscall.SIGKILL); err != nil {
		fmt.Println("kill failed :", err)
		return
	}
	fmt.Println("kill sucessful:", os.Args[2])
}
func ps(basePath string) {
	fmt.Printf("%-20s %-10s %-10s %s\n", "ID", "IMAGE", "STATUS", "STARTED")
	entries, _ := os.ReadDir(basePath + "/containers/")
	for _, entry := range entries {
		// entry.Name() gives the folder name
		data, _ := os.ReadFile(basePath + "/containers/" + entry.Name() + "/config.json")
		var c containers
		json.Unmarshal(data, &c)
		fmt.Printf("%-20s %-10s %-10s %s\n", c.ID, c.ImageName, c.Status, c.StartTime)

	}

}
func run(imageName string, basePath string) {

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
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET,
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("failed to start container:", err)
		return
	}
	id := cmd.Process.Pid
	pidStr := fmt.Sprintf("%d", id)
	os.WriteFile("/sys/fs/cgroup/mycontainer/cgroup.procs", []byte(pidStr), 0700)
	os.WriteFile("/sys/fs/cgroup/mycontainer/memory.max", []byte("10485760"), 0700)
	containerID := createcontainer(imageName, id, basePath)
	networking(id)
	err := cmd.Wait()
	updateprocess(basePath + "/containers/" + containerID + "/config.json")
	if err != nil {
		fmt.Println("container exited with error:", err)
	}

}
