package cmd

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/alphasoc/flightsim/simulator"
	"github.com/alphasoc/flightsim/utils"
	"github.com/alphasoc/flightsim/version"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var (
		fast           bool
		ifaceName      string
		simulatorNames = []string{"c2-dns", "dga", "scan", "tunnel"}
	)
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("run [%s]", strings.Join(simulatorNames, "|")),
		Short: "Run all simulators (default) or a particular test",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if !utils.StringsContains(simulatorNames, arg) {
					return fmt.Errorf("simulator %s not recognized", arg)
				}
			}

			if len(args) > 0 {
				simulatorNames = args
			}

			extIP, err := utils.ExternalIP(ifaceName)
			if err != nil {
				return err
			}

			simulators := selectSimulators(simulatorNames)
			interval := 2 * time.Second
			if fast {
				interval = 0
				for i := range simulators {
					simulators[i].interval = 0
				}
			}
			run(simulators, extIP, interval)
			return nil
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "run simulator fast without sleep intervals")
	cmd.Flags().StringVarP(&ifaceName, "interface", "i", "", "network interface to use")
	return cmd
}

func selectSimulators(names []string) []simulatorInfo {
	var simulators []simulatorInfo
	for _, s := range allsimualtors {
		if utils.StringsContains(names, s.name) {
			simulators = append(simulators, s)
		}
	}
	return simulators
}

type simulatorInfo struct {
	name        string
	infoHeaders []string
	infoRun     string
	s           simulator.Simulator
	interval    time.Duration
}

var allsimualtors = []simulatorInfo{
	{
		"c2-dns",
		[]string{"Preparing random sample of current C2 domains"},
		"Resolving %s",
		simulator.NewC2DNS(),
		500 * time.Millisecond,
	},
	{
		"dga",
		[]string{"Generating list of DGA domains"},
		"Resolving %s",
		simulator.NewDGA(),
		500 * time.Millisecond,
	},
	{
		"scan",
		[]string{
			"Preparing random sample of RFC 1918 destinations",
			"Preparing random sample of common TCP destination ports",
		},
		"Port scanning %s",
		simulator.NewPortScan(),
		0,
	},
	{
		"tunnel",
		[]string{"Preparing DNS tunnel hostnames"},
		"Resolving %s",
		simulator.NewTunnel(),
		500 * time.Millisecond,
	},
}

func run(simulators []simulatorInfo, extIP net.IP, interval time.Duration) error {
	printWelcome(extIP.String())
	printHeader()
	for _, s := range simulators {
		printMsg(s.name, "Starting")
		printMsg(s.name, s.infoHeaders...)
		time.Sleep(interval)

		hosts, err := s.s.Hosts()
		if err != nil {
			printMsg(s.name, color.RedString("failed ")+err.Error())
		}

		var prevHostname string
		for _, host := range hosts {
			hostname, _, err := net.SplitHostPort(host)
			if err != nil {
				hostname = host
			}

			// only print hostname when it has changed
			if prevHostname != hostname {
				printMsg(s.name, fmt.Sprintf(s.infoRun, hostname))
			}
			s.s.Simulate(extIP, host)
			time.Sleep(s.interval)
			prevHostname = hostname
		}
		printMsg(s.name, "Finished")
	}
	printGoodbay()
	return nil
}

func printHeader() {
	fmt.Println("Time      Module   Description")
	fmt.Println("--------------------------------------------------------------------------------")
}

func printMsg(module string, msg ...string) {
	for i := range msg {
		fmt.Printf("%s  %-7s  %s\n", time.Now().Format("15:04:05"), module, msg[i])
	}
}

func printWelcome(ip string) {
	fmt.Printf(`
AlphaSOC Network Flight Simulator™ %s (https://github.com/alphasoc/flightsim)
The IP address of the network interface is %s
The current time is %s

`, version.Version, ip, time.Now().Format("02-Jan-06 15:04:05"))
}

func printGoodbay() {
	fmt.Printf("\nAll done! Check your SIEM for alerts using the timestamps and details above.\n")
}
