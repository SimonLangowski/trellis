package coordinator

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
)

func memProfile(name string) {
	f, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

// very useful for debugging
// also good for profiling
func TestInprocessLightning(t *testing.T) {
	f, err := os.Create("lightning.pprof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	numServers := 10
	numGroups := 3
	groupSize := 3
	numLayers := 10
	numMessages := 100
	net := NewInProcessNetwork(numServers, numGroups, groupSize)
	c := NewCoordinator(net)
	exp := c.NewExperiment(0, numLayers, numServers, numMessages, "")
	exp.Info.SkipPathGen = true
	exp.KeyGen = true
	exp.Info.PathEstablishment = false
	err = c.DoAction(exp)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !exp.Passed {
		t.Log("Did message check?")
		t.FailNow()
	}
}

func TestInprocessPathEstablishment(t *testing.T) {
	f, err := os.Create("path.pprof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	numServers := 10
	numGroups := 3
	groupSize := 3
	numLayers := 10
	numMessages := 100
	net := NewInProcessNetwork(numServers, numGroups, groupSize)
	c := NewCoordinator(net)
	for i := 0; i < numLayers; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.KeyGen = (i == 0)
		exp.Info.PathEstablishment = true
		exp.Info.ReceiptLayer = 0
		exp.Info.NextLayer = int64(i)
		exp.Info.BoomerangLimit = int64(numLayers)
		exp.Info.LastLayer = (i == numLayers-1)
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if !exp.Passed {
			t.Log("Did message check?")
			t.FailNow()
		}
	}
}

func TestInprocessBoomerangPathEstablishment(t *testing.T) {
	f, err := os.Create("boomerangpath.pprof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	numServers := 10
	numGroups := 3
	groupSize := 3
	numLayers := 10
	numMessages := 100
	net := NewInProcessNetwork(numServers, numGroups, groupSize)
	c := NewCoordinator(net)
	for i := 0; i < numLayers; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.KeyGen = (i == 0)
		exp.Info.PathEstablishment = true
		exp.Info.BoomerangLimit = int64(numLayers) / 2
		if i-int(exp.Info.BoomerangLimit) > 0 {
			exp.Info.ReceiptLayer = int64(i) - exp.Info.BoomerangLimit
		}
		exp.Info.NextLayer = int64(i)
		exp.Info.LastLayer = (i == numLayers-1)
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if !exp.Passed {
			t.Log("Did message check?")
			t.FailNow()
		}
	}
}

func TestInprocessPathAndLightning(t *testing.T) {
	f, err := os.Create("pathLightning.pprof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	numServers := 10
	numGroups := 3
	groupSize := 3
	numLayers := 10
	numMessages := 100
	numLightning := 5
	net := NewInProcessNetwork(numServers, numGroups, groupSize)
	c := NewCoordinator(net)
	for i := 0; i < numLayers; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.KeyGen = (i == 0)
		exp.Info.PathEstablishment = true
		exp.Info.BoomerangLimit = int64(numLayers) / 2
		if i-int(exp.Info.BoomerangLimit) > 0 {
			exp.Info.ReceiptLayer = int64(i) - exp.Info.BoomerangLimit
		}
		exp.Info.NextLayer = int64(i)
		exp.Info.LastLayer = (i == numLayers-1)
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if !exp.Passed {
			t.Log("Did message check?")
			t.FailNow()
		}
	}
	for i := numLayers; i < numLayers+numLightning; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.Info.PathEstablishment = false
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if !exp.Passed {
			t.Log("Did message check?")
			t.FailNow()
		}
	}
}

func TestInprocessPathAndLightning2(t *testing.T) {
	numServers := 10
	numGroups := 3
	groupSize := 3
	numLayers := 10
	numMessages := 1000
	numLightning := 20
	net := NewInProcessNetwork(numServers, numGroups, groupSize)
	c := NewCoordinator(net)
	for i := 0; i < numLayers; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.KeyGen = (i == 0)
		exp.Info.PathEstablishment = true
		exp.Info.BoomerangLimit = int64(numLayers) / 2
		if i-int(exp.Info.BoomerangLimit) > 0 {
			exp.Info.ReceiptLayer = int64(i) - exp.Info.BoomerangLimit
		}
		exp.Info.NextLayer = int64(i)
		exp.Info.LastLayer = (i == numLayers-1)
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if i == 0 || i == numLayers-1 {
			memProfile(fmt.Sprintf("round%d.pprof", i))
		}
	}
	for i := numLayers; i < numLayers+numLightning; i++ {
		t.Logf("Round %v", i)
		exp := c.NewExperiment(i, numLayers, numServers, numMessages, "")
		exp.Info.PathEstablishment = false
		err := c.DoAction(exp)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		if i == numLayers+numLightning-1 {
			memProfile("leaks.prof")
		}
	}
}
