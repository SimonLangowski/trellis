package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/coordinator"
	"github.com/simonlangowski/lightning1/server/prepareMessages"
)

// run the coordinator, who starts and measures the experiment

var args struct {
	RunType     int     `default:"0"`
	F           float64 `default:"0"`
	NumUsers    int     `default:"0"`
	NumServers  int     `default:"0"`
	MessageSize int     `default:"1024"`

	NumGroups int `default:"0"`
	GroupSize int `default:"0"`
	BinSize   int `default:"0"`
	LimitSize int `default:"0"`
	NumLayers int `default:"0"`
	Overflow  int `default:"32"`

	NumClientServers int    `default:"0"`
	SkipPathGen      bool   `default:"False"`
	NoCheck          bool   `default:"False"`
	LoadMessages     bool   `default:"False"`
	StartIdx         int    `default:"0"`
	Interval         int    `default:"0"`
	ServerFile       string `default:"servers.json"`
	GroupFile        string `default:"groups.json"`
	ClientFile       string `default:"clients.json"`
	KeyFile          string `default:"keys.json"`
	MessageFile      string `default:"messages.json"`
	Ips              string `default:"../experiments/ip.list"`
	Notes            string `default:""`
	OutFile          string `default:"res.json"`
	NoDummies        bool   `default:"True"`

	Latency   int `default:"0"`
	Bandwidth int `default:"0"`
}

func main() {
	var net *coordinator.CoordinatorNetwork
	p := arg.MustParse(&args)
	if args.NumServers == 0 || args.NumUsers == 0 {
		log.Printf("Set numservers and numusers")
		p.WriteHelp(os.Stdout)
		return
	}
	if args.GroupSize == 0 {
		if args.F != 0 {
			if args.NumGroups != 0 {
				args.GroupSize = config.CalcGroupSize(args.NumServers, args.NumGroups, args.F)
			} else {
				args.GroupSize, args.NumGroups = config.CalcFewGroups2(args.F, args.NumServers)
			}
		} else {
			log.Printf("Set groupsize or f")
			p.WriteHelp(os.Stdout)
			return
		}
	}
	if args.NumGroups == 0 {
		// Integer division takes floor
		args.NumGroups = args.NumServers / args.GroupSize
	}
	if args.NumLayers == 0 {
		if args.F != 0 {
			args.NumLayers = config.NumLayers(args.NumUsers, args.F)
		} else {
			log.Printf("Set numlayers or f")
			p.WriteHelp(os.Stdout)
			return
		}
	}
	if args.BinSize == 0 {
		args.BinSize = config.BinSize2(args.NumLayers, args.NumServers, args.NumUsers, -args.Overflow)
	}
	if args.LimitSize == 0 {
		// the limit for how many servers needs to check the boomerang message
		// the size of an anytrust group ensures one honest server
		// we need replacement because servers can be selected multiple times
		args.LimitSize = config.GroupSizeWithReplacement(args.NumGroups, args.F)
	}
	if args.LoadMessages {
		args.NumClientServers = 0
	}

	log.Printf("%+v", args)
	if args.RunType == 0 {
		// run in the same process
		net = coordinator.NewInProcessNetwork(args.NumServers, args.NumGroups, args.GroupSize)
	} else if args.RunType == 1 {
		// run in separate process on the same machine
		serverConfigs, groupConfigs, clientConfigs := coordinator.NewLocalConfig(args.NumServers, args.NumGroups, args.GroupSize, args.NumClientServers, false)
		if args.LoadMessages {
			oldServers, err := config.UnmarshalServersFromFile(args.ServerFile)
			if err != nil {
				log.Fatalf("Could not read servers file %s", args.ServerFile)
			}
			// copy old keys
			for id, s := range serverConfigs {
				old := oldServers[id]
				s.PrivateKey = old.PrivateKey
				s.PublicKey = old.PublicKey
			}
		}
		net = coordinator.NewLocalNetwork(serverConfigs, groupConfigs, clientConfigs)
		defer net.KillAll()
	} else if args.RunType == 2 {
		// run on remote machines
		if args.NumClientServers == 0 {
			args.ClientFile = ""
		}
		net = coordinator.NewRemoteNetwork(args.ServerFile, args.GroupFile, args.ClientFile)
		defer net.KillAll()
		net.SetKill()
	} else if args.RunType == 3 {
		// create configuration files
		ips := ReadCsv(args.Ips)
		servers := make(map[int64]*config.Server)
		clients := make(map[int64]*config.Server)
		ids := make([]int64, 0)
		for id, addr := range ips {
			ids = append(ids, int64(id))
			if len(ips) >= args.NumClientServers+args.NumServers {
				if id < args.NumServers {
					servers[int64(id)] = config.CreateServerWithExisting(addr+":8000", int64(id), servers)
				} else if id < args.NumClientServers+args.NumServers {
					clients[int64(id-args.NumServers)] = config.CreateServerWithExisting(addr+":8900", int64(id-args.NumServers), servers)
				}
			} else {
				if id < args.NumServers {
					servers[int64(id)] = config.CreateServerWithExisting(addr+":8000", int64(id), servers)
				}
				// create on same servers
				if id < args.NumClientServers {
					clients[int64(id)] = config.CreateServerWithExisting(addr+":8900", int64(id), servers)
				}
			}
		}
		ids = ids[:args.NumServers]
		groups := config.CreateSeparateGroupsWithSize(args.NumGroups, args.GroupSize, ids)
		if args.NumClientServers > 0 {
			err := config.MarshalServersToFile(args.ClientFile, clients)
			if err != nil {
				log.Fatalf("Could not write clients file %s", args.ClientFile)
			}
		}

		if args.LoadMessages {
			oldServers, err := config.UnmarshalServersFromFile(args.ServerFile)
			if err != nil {
				log.Fatalf("Could not read servers file %s", args.ServerFile)
			}
			// copy old keys
			for id, s := range servers {
				old := oldServers[id]
				s.PrivateKey = old.PrivateKey
				s.PublicKey = old.PublicKey
			}
		}

		err := config.MarshalServersToFile(args.ServerFile, servers)
		if err != nil {
			log.Fatalf("Could not write servers file %s", args.ServerFile)
		}
		err = config.MarshalGroupsToFile(args.GroupFile, groups)
		if err != nil {
			log.Fatalf("Could not write group file %s", args.GroupFile)
		}
		return
	} else if args.RunType == 4 {
		// just wanted to see computed values
		expectation := (float64(args.NumUsers) / float64(args.NumServers*args.NumServers))
		expectation = math.Ceil(expectation)
		log.Printf("Dummy overhead: %.2f%%", 100*(float64(args.BinSize)/expectation-1))
		log.Printf("Group overhead: %f", config.GroupSizeCost(args.GroupSize, args.NumGroups, args.NumServers))
		bufferSizePath := prepareMessages.PathEstablishmentLengths(args.NumLayers, 8, args.LimitSize)[0] * args.BinSize * args.NumServers
		bufferSizeLightning := prepareMessages.LightningMessageLengths(args.NumLayers, args.MessageSize)[0] * args.BinSize * args.NumServers
		numDummies := args.BinSize*args.NumServers*args.NumServers - args.NumUsers
		log.Printf("Total path buffer size: %fG, lightning size %fG, numDummies: %v", float64(bufferSizePath)/1000000000, float64(bufferSizeLightning)/1000000000, numDummies)
		// log.Printf("Simulated time: %d path, %d broadcast", )
		return
	} else if args.RunType == 5 {
		// generate and record path establishment messages
		args.GroupSize = 1
		net = coordinator.NewInProcessNetwork(args.NumServers, args.NumGroups, args.GroupSize)
		if args.LoadMessages {
			oldServers, err := config.UnmarshalServersFromFile(args.ServerFile)
			if err != nil {
				log.Fatalf("Could not read servers file %s", args.ServerFile)
			}
			// copy old keys
			for id, s := range net.ServerConfigs {
				old := oldServers[id]
				s.PrivateKey = old.PrivateKey
				s.PublicKey = old.PublicKey
			}
			// have to rebuild if we changed the keys...
			net.SetupInProcess(args.NumServers)
		}
		err := config.MarshalServersToFile(args.ServerFile, net.ServerConfigs)
		if err != nil {
			log.Fatalf("Could not write servers file %s", args.ServerFile)
		}
	}
	numLayers := args.NumLayers
	numServers := args.NumServers
	numMessages := args.NumUsers
	numLightning := 5
	c := coordinator.NewCoordinator(net)
	if args.LoadMessages {
		c.LoadKeys(args.KeyFile)
		c.LoadMessages(args.MessageFile)
	}
	l := 0
	if !args.SkipPathGen {
		for i := 0; i < numLayers; i++ {
			log.Printf("Round %v", i)
			exp := c.NewExperiment(i, numLayers, numServers, numMessages, args)
			if i == 0 {
				exp.KeyGen = !args.LoadMessages
				exp.LoadKeys = args.LoadMessages
			}
			exp.Info.PathEstablishment = true
			exp.Info.LastLayer = (i == numLayers-1)
			exp.Info.Check = !args.NoCheck
			exp.Info.Interval = int64(args.Interval)
			if args.BinSize > 0 {
				exp.Info.BinSize = int64(args.BinSize)
			} else if i == 0 {
				log.Printf("Using bin size %d", exp.Info.BinSize)
			}
			if args.LimitSize > 0 {
				exp.Info.BoomerangLimit = int64(args.LimitSize)
			} else {
				exp.Info.BoomerangLimit = int64(numLayers)
			}
			exp.Info.ReceiptLayer = 0
			if i-int(exp.Info.BoomerangLimit) > 0 {
				exp.Info.ReceiptLayer = int64(i) - exp.Info.BoomerangLimit
			}
			exp.Info.NextLayer = int64(i)
			if args.RunType == 5 {
				exp.Info.StartId = int64(args.StartIdx)
				exp.Info.EndId = exp.Info.StartId + int64(numMessages)
				c.WriteKeys(args.KeyFile)
				c.WriteMessages(args.MessageFile, exp)
				return
			}
			err := c.DoAction(exp)
			if err != nil {
				log.Print(err)
				return
			}
			log.Printf("Path round %v took %v", i, time.Since(exp.ExperimentStartTime))
			exp.RecordToFile(args.OutFile)
			RecordToCsv(args.OutFile+".csv", exp)
		}
		l = numLayers
	}
	for i := l; i < l+numLightning; i++ {
		log.Printf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, args)
		exp.Info.PathEstablishment = false
		exp.Info.MessageSize = int64(args.MessageSize)
		exp.Info.Check = !args.NoCheck
		if args.BinSize > 0 {
			exp.Info.BinSize = int64(args.BinSize)
		}
		if args.SkipPathGen && (i == 0) {
			exp.Info.SkipPathGen = true
			exp.KeyGen = true
		}
		exp.Info.Interval = int64(args.Interval)
		err := c.DoAction(exp)
		if err != nil {
			log.Print(err)
			return
		}
		log.Printf("Lightning round %v took %v", i, time.Since(exp.ExperimentStartTime))
		exp.RecordToFile(args.OutFile)
		RecordToCsv(args.OutFile+".csv", exp)
	}
	l += numLightning
}

func ReadCsv(fn string) []string {
	ifile, err := os.Open(fn)
	if err != nil {
		log.Fatalf("Could not open %s", fn)
	}
	reader := csv.NewReader(ifile)
	ips, err := reader.ReadAll()
	if err != nil {
		log.Fatal("Could not read the list of ips")
	}
	ifile.Close()
	output := make([]string, 0)
	for _, ip := range ips {
		output = append(output, ip[0])
	}
	return output
}

func RecordToCsv(fn string, e *coordinator.Experiment) {
	f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b := bufio.NewWriter(f)

	fmt.Fprintf(b, "%v, ", args.NumServers)
	fmt.Fprintf(b, "%v, ", args.NumUsers)
	fmt.Fprintf(b, "%v, ", args.F)
	fmt.Fprintf(b, "%v, ", args.MessageSize)

	fmt.Fprintf(b, "%v, ", args.NumLayers)
	fmt.Fprintf(b, "%v, ", args.BinSize)
	fmt.Fprintf(b, "%v, ", args.Bandwidth)
	fmt.Fprintf(b, "%v, ", args.Latency)

	fmt.Fprintf(b, "%v, ", e.Info.Round)
	fmt.Fprintf(b, "%v, ", e.Info.PathEstablishment)
	fmt.Fprintf(b, "%d\n", e.ServerRoundTime)

	err = b.Flush()
	if err != nil {
		panic(err)
	}

}
