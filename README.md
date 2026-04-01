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
| chroot | Container gets its own root filesystem | `syscall.Chroot` |
| /proc mount | ps and process tools work inside | `syscall.Mount` |
| cgroups | Memory and CPU limits (needs bare metal) | `/sys/fs/cgroup` |

---

## Requirements

- **Linux** — this will not run on macOS or Windows. Needs a real Linux kernel.
- **Root / sudo** — namespaces and chroot require root privileges.
- **Go** — any recent version.
- **A root filesystem** — an Alpine Linux minirootfs tarball (see setup below).

> If you are on macOS, you need either a Linux VM, a VPS, or run inside a
> Docker container with `--privileged`. See the Environment section below.

---

## Setup

### 1. Clone the repo

```bash
git clone https://github.com/avirals554/go-dock
cd go-dock
```

### 2. Download a root filesystem

The container needs its own filesystem to chroot into.
Download a minimal Alpine Linux rootfs for your architecture:

**AMD64 (most Linux servers, Google Cloud Shell):**
```bash
mkdir rootfs
curl -L https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz | tar xz -C rootfs
```

**ARM64 (Apple Silicon Mac, Raspberry Pi):**
```bash
mkdir rootfs
curl -L https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/aarch64/alpine-minirootfs-3.19.0-aarch64.tar.gz | tar xz -C rootfs
```

### 3. Set the environment variable

```bash
export ROOTFS_PATH=/absolute/path/to/go-dock/rootfs
```

Or pass it inline when running (see Usage below).

---

## Usage

### Basic run

```bash
sudo ROOTFS_PATH=/absolute/path/to/rootfs $(which go) run main.go
```

You will get a shell inside the isolated container:

```
/ #
```

### Verify isolation

From inside the container:

```sh
# Should show only a handful of processes (not hundreds)
ps aux

# Should show only the Alpine filesystem
ls /

# Set a custom hostname - won't affect the host
hostname mycontainer
hostname
```

From the host in another terminal:

```sh
# Host hostname is unchanged
hostname
```

### Set a custom rootfs path

```bash
sudo ROOTFS_PATH=/path/to/your/rootfs $(which go) run main.go
```

---

## How It Works

The program runs itself twice using a two-stage pattern:

```
go run main.go          (no args = parent mode)
        |
        | spawns itself with "child" argument
        | sets up namespaces via Cloneflags
        v
go run main.go child    (child mode = already inside namespaces)
        |
        | chroot into rootfs
        | chdir to /
        | mount /proc
        | mount /sys/fs/cgroup
        v
/bin/sh                 (your isolated shell)
```

### Why two stages?

We need to mount `/proc` inside the new PID namespace. But we can only do that
after the namespace is created (when the child starts). The child mounts its own
`/proc` — correctly inside its namespace — then starts the shell.

### Why /proc/self/exe?

`/proc/self/exe` is a Linux symlink that always points to the currently running
binary. The parent uses it to spawn an exact copy of itself without needing to
know its own path on disk.

---

## Code Walkthrough

```
main.go
  |
  |-- if "child" argument present (CHILD MODE)
  |     |-- syscall.Chroot(rootfs)       change what / means for this process
  |     |-- syscall.Chdir("/")           move into the new root (critical!)
  |     |-- os.MkdirAll(/sys/fs/cgroup) create mount point for cgroups
  |     |-- syscall.Mount(proc)          mount /proc for this namespace
  |     |-- syscall.Mount(cgroup2)       mount cgroup fs inside container
  |     |-- exec.Command(/bin/sh)        start the isolated shell
  |
  |-- else (PARENT MODE)
        |-- exec.Command(/proc/self/exe, "child")
        |-- SysProcAttr.Cloneflags       create namespaces at birth
        |-- cmd.Start()                  start child, don't wait
        |-- cmd.Process.Pid              grab child PID immediately
        |-- os.WriteFile(cgroup.procs)   put child in cgroup
        |-- os.WriteFile(memory.max)     set memory limit
        |-- cmd.Wait()                   now wait for child to exit
```

---

## Environment Notes

### Google Cloud Shell
Works for everything except cgroups. Cloud Shell runs inside Kubernetes which
restricts cgroup access. Namespaces, chroot, /proc all work fine.

```bash
sudo ROOTFS_PATH=$(pwd)/rootfs $(which go) run main.go
```

### Docker (--privileged)
```bash
docker run -it --privileged -v $(pwd):/mycontainer ubuntu bash
cd /mycontainer
# install go, then run
sudo ROOTFS_PATH=/mycontainer/rootfs go run main.go
```

Note: cgroups will be restricted inside Docker too. Use a real Linux VPS for
full cgroup support.

### Real Linux VPS (recommended for full features)
Any VPS with a real Linux kernel works. Recommended:
- Hetzner CX22 — cheapest, ~4 EUR/month
- Oracle Cloud Free Tier — free forever, needs credit card verification
- AWS EC2 t2.micro — free for 12 months

On a real VPS, first enable memory and cpu controllers:
```bash
echo "+memory +cpu" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
mkdir -p /sys/fs/cgroup/mycontainer
```

Then run normally.

### Raspberry Pi
Works perfectly. Real Linux kernel, full root access. Just make sure to
download the ARM64 rootfs (aarch64).

---

## Known Limitations

- **cgroups require bare metal** — Docker-in-Docker and Cloud Shell restrict
  cgroup subtree_control. The code is correct, the environment is the wall.

- **Network isolation not yet implemented** — container shares host network.
  Next step is `CLONE_NEWNET` + veth pairs.

- **No image pulling** — rootfs must be downloaded manually. Next step is
  pulling from a registry or OCI image spec.

- **Single container** — no lifecycle management, no multiple containers.

- **chroot not pivot_root** — chroot is simpler but slightly less secure than
  pivot_root. Production runtimes use pivot_root.

---

## What's Next

Things to build to get closer to real Docker:

```
[ ] Network namespace    CLONE_NEWNET + veth pairs + bridge networking
[ ] User namespace       CLONE_NEWUSER + uid/gid mapping
[ ] pivot_root           More secure alternative to chroot
[ ] Image layers         Overlayfs for copy-on-write filesystems
[ ] OCI image pulling    Pull images from Docker Hub / registries
[ ] Port mapping         Forward host ports into container network
[ ] Container networking Multiple containers talking to each other
```

---

## What i Learned Building This

- Every running program is a **process** with a PID and access to shared resources
- **Namespaces** control what a process can *see* (processes, filesystem, network, hostname)
- **cgroups** control what a process can *use* (CPU, memory, disk I/O)
- **chroot** changes what a process thinks is `/` — but needs `chdir("/")` after it
- `/proc` is a virtual filesystem the kernel generates live — not real files on disk
- `cmd.Start()` starts a child and returns; `cmd.Wait()` pauses until child exits
- When a parent exits, orphaned children lose their terminal and exit too
- Syscalls are direct kernel requests — no middleman process, no file on disk
- Docker on Mac runs a hidden Linux VM because namespaces are Linux-only
- Docker-in-Docker restricts cgroup access because Docker sits above you in the hierarchy
- Every user has their own PATH — sudo uses root's PATH, not yours

---

## Resources

- [Julia Evans — What even is a container](https://jvns.ca/blog/2016/10/10/what-even-is-a-container/)
- [Liz Rice — Containers from Scratch (YouTube)](https://www.youtube.com/watch?v=8fi7uSYlOdc)
- [NGINX — What are Namespaces and cgroups](https://blog.nginx.org/blog/what-are-namespaces-cgroups-how-do-they-work)
- [Linux man pages — clone(2), chroot(2), mount(2)](https://man7.org/linux/man-pages/)
