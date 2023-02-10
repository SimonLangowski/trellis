package coordinator

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/network"
)

const ServerProcessName = "server"
const ClientProcessName = "client"
const waitTime = 10 * time.Second

func TransferFileToAllServers(servers map[int64]*config.Server, fn string) bool {
	done := make(chan bool)
	for _, s := range servers {
		go func(s *config.Server) {
			cmd := exec.Command("scp", "-i", "~/.ssh/lkey", "-o", "StrictHostKeyChecking=no", fn,
				fmt.Sprintf("ec2-user@%s:~/go/bin/%s", config.Host(s.Address), fn))
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			// log.Printf("Running %v", cmd)
			err := cmd.Run()
			if err != nil {
				log.Print(err)
				done <- false
			} else {
				done <- true
			}
		}(s)
	}
	for range servers {
		b := <-done
		if !b {
			return false
		}
	}
	return true
}

func KillRemoteServers(servers map[int64]*config.Server, processName string) {
	done := make(chan bool)
	for _, s := range servers {
		go func(s *config.Server) {
			cmd := exec.Command("ssh", "-i", "~/.ssh/lkey", "-o", "StrictHostKeyChecking=no",
				fmt.Sprintf("ec2-user@%s", config.Host(s.Address)),
				fmt.Sprintf("pkill %s", processName))
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			cmd.Run()
			// if err != nil {
			// 	log.Printf("Could not kill %s on server %s", processName, s.Address)
			// }
			done <- true
		}(s)
	}
	for range servers {
		<-done
	}
}

func StartRemoteServers(servers map[int64]*config.Server, processName, serverFile, groupFile, clientsFile string) bool {
	ch := make(chan bool)
	for _, s := range servers {
		go func(s *config.Server) {
			cmd := exec.Command("ssh", "-i", "~/.ssh/lkey", "-o", "StrictHostKeyChecking=no",
				fmt.Sprintf("ec2-user@%s", config.Host(s.Address)),
				fmt.Sprintf("~/go/bin/%s ~/go/bin/%s ~/go/bin/%s ~/go/bin/%s %s", processName, serverFile, groupFile, clientsFile, s.Address))
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			// log.Printf("Running %v", cmd)
			err := cmd.Run()
			if err != nil {
				log.Print(err)
				ch <- false
			}
		}(s)
	}
	select {
	case <-ch:
		return false
	case <-time.After(waitTime):
		return true
	}
}

func RunRemoteCommandOnEach(servers map[int64]*config.Server, remoteCmd string) bool {
	done := make(chan bool)
	for _, s := range servers {
		go func(s *config.Server) {
			cmd := exec.Command("ssh", "-i", "~/.ssh/lkey", "-o", "StrictHostKeyChecking=no",
				fmt.Sprintf("ec2-user@%s", config.Host(s.Address)),
				remoteCmd)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			// log.Printf("Running %v", cmd)
			err := cmd.Run()
			if err != nil {
				log.Print(err)
				done <- false
			} else {
				done <- true
			}
		}(s)
	}
	for range servers {
		b := <-done
		if !b {
			return false
		}
	}
	return true
}

// Latency in ms, bandwidth in Mb/s
func (c *CoordinatorNetwork) SlowNetwork(latency, bandwidth int) {
	if latency <= 0 && bandwidth <= 0 {
		return
	}
	if bandwidth <= 0 {
		// Don't set 0 bandwidth!
		bandwidth = 12500
	}
	var device = ""
	if c.serverNetType == inprocess {
		network.MockLatency = latency
		network.MockBandwidth = bandwidth
		return
	} else if c.serverNetType == remote {
		device = "eth0"
	} else if c.serverNetType == local {
		device = "lo"
	}
	// assign half of the round trip latency (ping time) to each server
	latency = latency / 2
	cmd := fmt.Sprintf("sudo tc qdisc add dev %s root handle 1:0 netem delay %dms rate %dmbit limit 100000", device, latency, bandwidth)
	if !RunRemoteCommandOnEach(c.ServerConfigs, cmd) {
		c.KillAll()
		panic("Error setting network slowdown")
	}
}
