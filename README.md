# go-dock

A container runtime built from scratch in Go using raw Linux kernel primitives.
No Docker. No libraries. Just syscalls.

---

## What This Is

This project implements the core of what Docker does under the hood — taking a normal
Linux process and isolating it using namespaces, chroot, and cgroups. Every line exists
because of a specific Linux kernel feature.

Built as a learning project to understand containers from first principles.

---

## What It Does

When you run `go-dock`, it creates an isolated environment with:

| Feature | What It Does | Linux Primitive |
|---|---|---|
| PID namespace | Container sees only its own processes | `CLONE_NEWPID` |
| Mount namespace | Container has its own filesystem view | `CLONE_NEWNS` |
| UTS namespace | Container has its own hostname | `CLONE_NEWUTS` |
| Network namespace | Container has its own network stack | `CLONE_NEWNET` |
| chroot | Container gets its own root filesystem | `syscall.Chroot` |
| /proc mount | ps and process tools work inside | `syscall.Mount` |
| veth pair | virtual cable connecting container to host | `netlink` |
| NAT | container can reach the internet | `iptables` |
| DNS | domain name resolution works inside | `/etc/resolv.conf` |
| cgroups | Memory and CPU limits (needs bare metal) | `/sys/fs/cgroup` |

---

## Requirements

- **Linux** — this will not run on macOS or Windows. Needs a real Linux kernel.
- **Root / sudo** — namespaces and chroot require root privileges.
- **Go** — any recent version.

> If you are on macOS, you need either a Linux VM, a VPS, or run inside a
> Docker container with `--privileged`. See the Environment section below.

---

## Installation

```bash
git clone https://github.com/avirals554/go-dock
cd go-dock
chmod +x install.sh
./install.sh
```

Or manually:

```bash
go build -o go-dock .
sudo mv go-dock /usr/local/bin/
mkdir -p ~/.go-dock/images
mkdir -p ~/.go-dock/containers
```

---

## Usage

### Pull an image

Downloads a root filesystem and stores it locally:

```bash
sudo go-dock pull alpine
```

Supported images:
- `alpine` — Alpine Linux 3.19 (AMD64)
- `alpine3` — Alpine Linux 3.18 (AMD64)

### Run a container

Starts an isolated shell inside the container:

```bash
sudo go-dock run alpine
```

You will get a shell inside the isolated container:

```
/ #
```

### List containers

```bash
sudo go-dock ps
```

Output:

```
ID                   IMAGE      STATUS     STARTED
1775215631084        alpine     ALIVE      2026-04-03 11:27:11
1775214667931        alpine     DEAD       2026-04-03 11:11:07
```

### Kill a container

```bash
sudo go-dock kill <full-container-ID>
```

### Verify isolation

From inside the container:

```sh
# Only shows a handful of processes — not the host's hundreds
ps aux

# Only shows the Alpine filesystem — not the host's files
ls /

# Set a custom hostname — won't affect the host
hostname mycontainer
hostname

# Container has its own network interface
ip addr

# Can reach the internet
ping 8.8.8.8
ping google.com
```

From the host in another terminal:

```sh
# Host hostname is completely unchanged
hostname
```

---

## How It Works

The program runs itself twice using a two-stage pattern:

```
go-dock run alpine      (parent mode)
        |
        | builds rootfs path from ~/.go-dock/images/alpine
        | sets up namespaces via Cloneflags
        | spawns /proc/self/exe child <rootfspath>
        | creates veth pair, moves veth1 into container
        | enables IP forwarding + NAT
        | writes PID to cgroup
        | waits for child to exit
        v
/proc/self/exe child    (child mode — already inside namespaces)
        |
        | chroot into rootfs
        | chdir to /
        | mount /proc
        | mount /sys/fs/cgroup
        | bring veth1 up, assign IP 192.168.1.2
        | add default route via 192.168.1.1
        | write /etc/resolv.conf for DNS
        v
/bin/sh                 (your isolated shell with internet access)
```

### Why two stages?

We need to mount `/proc` inside the new PID namespace — but only after the namespace
is created. The child is already inside the new namespace when it starts, so it mounts
its own `/proc` correctly, then starts the shell.

### How networking works

```
Container (192.168.1.2)
    └── veth1 <-----------> veth0 (192.168.1.1)
                                └── NAT/Masquerade
                                      └── eth0 (host IP)
                                            └── Internet
```

A veth pair is a virtual ethernet cable with two ends. One end (`veth1`) lives inside
the container's network namespace. The other (`veth0`) stays on the host. NAT translates
the container's private IP to the host's public IP for outbound traffic.

### Image storage

Images are stored as extracted root filesystems:

```
~/.go-dock/
    images/
        alpine/          <- go-dock pull alpine extracts here
            bin/
            etc/
            lib/
            ...
    containers/
        <id>/
            config.json  <- created when container starts
```

### Container tracking

When a container starts, go-dock saves a `config.json`:

```json
{
  "ID": "1775215631084251214",
  "ImageName": "alpine",
  "PID": 4261,
  "StartTime": "2026-04-03 11:27:11",
  "Status": "ALIVE"
}
```

`go-dock ps` reads all config files and displays them. Status is updated to `DEAD`
when the container exits or is killed.

---

## Code Structure

```
go-dock/
    main.go        <- CLI entry point, reads os.Args, routes to functions
    container.go   <- run, ps, kill, createcontainer, updateprocess, networking
    image.go       <- pull, image map, tar/gzip extraction
    namespace.go   <- child mode, chroot, mount, veth1 setup, DNS
    install.sh     <- installation script
    README.md
```

---

## Environment Notes

### Google Cloud Shell

Works for everything except cgroups. Cloud Shell runs inside Kubernetes which
restricts cgroup access. Namespaces, chroot, networking, pull, ps, kill all work fine.

```bash
sudo HOME=/home/<your-username> $(which go) run . pull alpine
sudo HOME=/home/<your-username> $(which go) run . run alpine
```

### Docker (--privileged)

```bash
docker run -it --privileged -v $(pwd):/mycontainer ubuntu bash
cd /mycontainer
# install go, then:
sudo go run . pull alpine
sudo go run . run alpine
```

Note: cgroups will be restricted inside Docker too. Use a real Linux VPS for
full cgroup support.

### Real Linux VPS (recommended for full features)

Any VPS with a real Linux kernel works. Recommended:
- Hetzner CX22 — cheapest, ~4 EUR/month
- Oracle Cloud Free Tier — free forever, needs credit card verification
- AWS EC2 t2.micro — free for 12 months
- Raspberry Pi — works perfectly, use ARM64 image

On a real VPS, first enable memory and cpu controllers:

```bash
echo "+memory +cpu" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
mkdir -p /sys/fs/cgroup/mycontainer
```

Then run normally.

---

## Known Limitations

- **cgroups require bare metal** — Docker-in-Docker and Cloud Shell restrict
  cgroup subtree_control. The code is correct, the environment is the wall.

- **Single container networking** — veth interfaces are hardcoded as veth0/veth1.
  Running two containers simultaneously would conflict. Fix: use unique names per container.

- **AMD64 only for now** — image URLs in the map point to x86_64 builds.
  ARM64 URLs can be added to the image map for Raspberry Pi / Apple Silicon.

- **chroot not pivot_root** — chroot is simpler but slightly less secure than
  pivot_root. Production runtimes use pivot_root.

- **Short ID not supported in kill** — must use the full container ID shown by `go-dock ps`.

- **No port mapping yet** — cannot expose container ports to the host.

---

## Progress

How far along is go-dock compared to a real container runtime?

```
Core isolation
  [x] PID namespace          container sees only its own processes
  [x] Mount namespace        container has its own filesystem view
  [x] UTS namespace          container has its own hostname
  [x] Network namespace      container has its own network stack
  [x] chroot                 container gets its own root filesystem
  [x] /proc mount            ps and process tools work inside
  [ ] User namespace         container root maps to safe host user
  [ ] pivot_root             more secure alternative to chroot

Resource limits
  [x] cgroups (code done)    memory.max and cpu limits written
  [ ] cgroups (enforced)     needs bare metal Linux to actually work

Networking
  [x] network namespace      CLONE_NEWNET — isolated network stack
  [x] veth pair              virtual cable between container and host
  [x] IP forwarding          host forwards packets for container
  [x] NAT / masquerade       container reaches internet via host IP
  [x] DNS                    domain name resolution via resolv.conf
  [x] default route          container knows to send traffic to host
  [ ] unique veth names      support multiple containers at once
  [ ] port mapping           expose container ports to host
  [ ] bridge networking      containers talk to each other

Image management
  [x] pull                   download and extract rootfs from CDN
  [x] image map              named images with download URLs
  [ ] go-dock images         list downloaded images
  [ ] ARM64 support          add ARM64 URLs to image map
  [ ] OCI registry           pull from Docker Hub / real registries
  [ ] image layers           overlayfs copy-on-write like real Docker

Container lifecycle
  [x] run                    start an isolated container
  [x] ps                     list containers with status
  [x] kill                   stop a running container
  [x] container tracking     config.json saved per container
  [x] auto status update     DEAD set on exit and kill
  [ ] go-dock rm             delete container record from disk
  [ ] short ID matching      accept first 12 chars like Docker does

CLI and distribution
  [x] named commands         go-dock run / pull / ps / kill
  [x] usage message          helpful output when no args given
  [x] refactored codebase    split into container.go image.go namespace.go
  [ ] install script         one-line install like get.docker.com
  [ ] binary releases        pre-built binaries on GitHub Releases
  [ ] CI/CD                  auto-build on every git push
```

**Overall: ~70% of a minimal container runtime**

---

## What I Learned Building This

- Every running program is a **process** with a PID and access to shared resources
- **Namespaces** control what a process can *see* (processes, filesystem, network, hostname)
- **cgroups** control what a process can *use* (CPU, memory, disk I/O)
- **chroot** changes what a process thinks is `/` — but needs `chdir("/")` after it
- `/proc` is a virtual filesystem the kernel generates live — not real files on disk
- `cmd.Start()` starts a child and returns; `cmd.Wait()` pauses until child exits
- When a parent exits without `cmd.Wait()`, orphaned children lose their terminal and exit
- Syscalls are direct kernel requests — no middleman process, no file on disk
- Docker on Mac runs a hidden Linux VM because namespaces are Linux-only
- Docker-in-Docker restricts cgroup access because Docker sits above you in the hierarchy
- Every user has their own PATH — sudo uses root's PATH, not yours
- `.tar.gz` files have two layers — gzip compression wrapping a tar archive
- File permissions must be explicitly restored when extracting tar archives
- Symlinks are a separate entry type in tar and need special handling
- Go compiles to a single self-contained binary — no runtime needed on target machine
- All files in the same Go package share functions and variables automatically
- Processes don't have IP addresses — network interfaces do, processes share them via ports
- A veth pair is a virtual ethernet cable — two ends, one per namespace
- NAT translates private container IPs to the host's public IP for internet access
- DNS is just a text file telling the OS which server to ask for domain name lookups
- IP forwarding must be explicitly enabled for the host to route container packets
- Port forwarding and NAT are opposite directions — outbound vs inbound traffic

---

## Resources

- [Julia Evans — What even is a container](https://jvns.ca/blog/2016/10/10/what-even-is-a-container/)
- [Liz Rice — Containers from Scratch (YouTube)](https://www.youtube.com/watch?v=8fi7uSYlOdc)
- [NGINX — What are Namespaces and cgroups](https://blog.nginx.org/blog/what-are-namespaces-cgroups-how-do-they-work)
- [Linux man pages — clone(2), chroot(2), mount(2)](https://man7.org/linux/man-pages/)

---

*Built from first principles. No magic.*
