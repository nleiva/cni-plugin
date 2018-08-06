package invoke

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
)

var (
	command     = getenv("CNI_COMMAND")
	ifname      = getenv("CNI_IFNAME")
	cnipath     = getenv("CNI_PATH")
	config      = "cni-conf.json"
	defaultExec = &invoke.RawExec{Stderr: os.Stderr}
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Panicf("%s environment variable not set.", name)
	}
	return v
}

// Exec is a function that executes a CNI plugin
func Exec(netns, containerid string) error {
	// We use a file here.They use in general Stdin := os.Stdin
	Stdin, err := os.Open(config)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	stdinData, err := ioutil.ReadAll(Stdin)
	if err != nil {
		return fmt.Errorf("error reading from stdin: %v", err)
	}

	// Parse the config file
	cfg := new(types.NetConf)
	if err := json.Unmarshal(stdinData, cfg); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	// type (string): Refers to the filename of the CNI plugin executable.
	pluginPath := filepath.Join(cnipath, cfg.Type)

	args := invoke.Args{
		Command:     command,
		ContainerID: containerid,
		NetNS:       netns,
		IfName:      ifname,
		Path:        cnipath,
	}

	env := os.Environ()
	environ := append([]string{
		"CNI_COMMAND=" + args.Command,
		"CNI_CONTAINERID=" + args.ContainerID,
		"CNI_NETNS=" + args.NetNS,
		"CNI_IFNAME=" + args.IfName,
		"CNI_PATH=" + args.Path,
	}, env...)

	r, err := defaultExec.ExecPlugin(pluginPath, stdinData, environ)
	fmt.Printf("%s\n", string(r))
	return err
}
