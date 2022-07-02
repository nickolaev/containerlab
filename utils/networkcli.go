package utils

import (
	"github.com/scrapli/scrapligo/util"
	"strings"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	log "github.com/sirupsen/logrus"
)

var (
	// map of commands per platform which start a CLI app
	NetworkOSCLICmd = map[string]string{
		"arista_eos":    "Cli",
		"nokia_srlinux": "sr_cli",
	}

	// map of the cli exec command and its argument per runtime
	// which is used to spawn CLI session
	CLIExecCommand = map[string]map[string]string{
		"docker": {
			"exec": "docker",
			"open": "exec -it",
		},
		"containerd": {
			"exec": "ctr",
			"open": "-n clab task exec -t --exec-id clab",
		},
	}
)

// SpawnCLIviaExec spawns a CLI session over container runtime exec function
// end ensures the CLI is available to be used for sending commands over
func SpawnCLIviaExec(platformName, contName, runtime string) (*network.Driver, error) {
	var d *network.Driver
	var err error

	opts := []util.Option{
		options.WithAuthBypass(),
		options.WithSystemTransportOpenBin(CLIExecCommand[runtime]["exec"]),
		options.WithSystemTransportOpenArgsOverride(
			append(
				strings.Split(CLIExecCommand[runtime]["open"], " "),
				contName,
				NetworkOSCLICmd[platformName],
			),
		),
	}

	// scrapligo v1.0.0 changed to just "nokia_srl" rather than "nokia_srlinux"
	if strings.HasPrefix(platformName, "nokia_srl") {
		// jack up TermWidth, since we use `docker exec` to enter certificate and key strings
		// and these are lengthy
		opts = append(opts, options.WithTermWidth(5000))
	}

	p, err := platform.NewPlatform(
		platformName,
		contName,
		options.WithAuthBypass(),
	)
	if err != nil {
		log.Errorf("failed to fetch platform instance for device %s; error: %+v\n", err, contName)
		return nil, err
	}

	d, err = p.GetNetworkDriver()
	if err != nil {
		log.Errorf("failed to create driver for device %s; error: %+v\n", err, contName)
		return nil, err
	}

	transportReady := false
	for !transportReady {
		if err = d.Open(); err != nil {
			log.Debugf("%s - Cli not ready (%s) - waiting.", contName, err)
			time.Sleep(time.Second * 2)
			continue
		}
		transportReady = true
		log.Debugf("%s - Cli ready.", contName)
	}

	return d, err
}
