/*
Adding Liz Rice's Container from scratch to a new network namespace
and configuring the network via a modified CNI-plugin.

containers-from-scratch: https://github.com/lizrice/containers-from-scratch
CNI-Plugins: https://github.com/containernetworking/plugins
*/
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nleiva/cni-plugin/invoke"
)

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child("/home/nleiva/rootfs")
	default:
		panic("help")
	}
}

func run() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	// Starts the specified command but does not wait for it to complete.
	must(cmd.Start())

	os.Setenv("CNI_COMMAND", "ADD")
	err := network(cmd.Process.Pid)
	check(err)

	// Waits for the command to exit and waits for any copying to stdin or
	// copying from stdout or stderr to complete.
	must(cmd.Wait())

	// Too late to execute CNI DEL, Network namespace is gone already.
	// os.Setenv("CNI_COMMAND", "DEL")
	// err = network(cmd.Process.Pid)
	// check(err)

	// must(cmd.Run())
}

func child(rootfs string) {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	must(syscall.Chroot(rootfs))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	// must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))
	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	// must(syscall.Unmount("thing", 0))
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "liz"), 0755)
	must(ioutil.WriteFile(filepath.Join(pids, "liz/pids.max"), []byte("20"), 0700))
	// Removes the new cgroup in place after the container exits
	must(ioutil.WriteFile(filepath.Join(pids, "liz/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func network(pid int) (err error) {
	// In a single-threaded process, the thread ID is equal to the process ID.
	// In a multithreaded process, all threads have the same PID, but each one has
	// a unique TID.
	tid := pid
	netns := fmt.Sprintf("/proc/%d/task/%d/ns/net", pid, tid)
	containerid := fmt.Sprintf("cni-%d", pid)
	err = invoke.Exec(netns, containerid)
	return
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func check(err error) {
	if err != nil {
		log.Println(err)
	}
}
