package config

import (
	"testing"
)

func TestGroups(t *testing.T) {
	n := 128
	for f := 0.01; f < 1; f *= 2 {
		groupSize, nGroups := CalcFewGroups(f, n)
		t.Logf("%f: %d %d\n", f, groupSize, nGroups)
		// hmm but if f = 0.01, doesn't pigeonhole mean groupSize = 2 should work?
	}
	for f := 0.01; f < 1; f *= 2 {
		groupSize, nGroups := CalcFewGroups2(f, n)
		t.Logf("%f: %d %d\n", f, groupSize, nGroups)
		// hmm but if f = 0.01, doesn't pigeonhole mean groupSize = 2 should work?
	}
}

func TestGroupSelection(t *testing.T) {
	n := 10
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(i)
	}
	for groupSize := 3; groupSize < 10; groupSize++ {
		for numGroups := 2; numGroups < 5; numGroups++ {
			groups := CreateSeparateGroupsWithSize(numGroups, groupSize, ids)
			groupList := make([][]int64, 0)
			for _, v := range groups {
				groupList = append(groupList, v.Servers)
			}
			t.Log(groupList)
		}
	}
}

const LinkOverflowProbability = -32

func TestBinSize(t *testing.T) {
	numTransitions := 70
	numServers := 100
	t.Logf("Target probability %v", Target(numTransitions*numServers*numServers, LinkOverflowProbability))
	for numMessages := 10; numMessages <= 10000000; numMessages *= 10 {
		size := BinSize2(numTransitions, numServers, numMessages, LinkOverflowProbability)
		size2 := BinSize3(numTransitions, numServers, numMessages, LinkOverflowProbability)
		t.Logf("%d messages, Expectation: %f", numMessages, float64(numMessages)/float64(numServers*numServers))
		t.Logf("Bin size: %d %d", size, size2)
	}
}

func TestBinSizeFewServers(t *testing.T) {
	numServers := 2
	numMessages := 100
	for numLayers := 0; numLayers < 10; numLayers++ {
		size := BinSize2(numLayers, numServers, numMessages, LinkOverflowProbability)
		t.Logf("%d layers, bin size: %d", numLayers, size)
	}
}

func TestBinSizeMoreServers(t *testing.T) {
	numServers := 10
	numMessages := 10000
	for numLayers := 0; numLayers < 10; numLayers++ {
		size := BinSize2(numLayers, numServers, numMessages, LinkOverflowProbability)
		t.Logf("%d layers, bin size: %d", numLayers, size)
	}
}

func TestBinSizeManyMessagesFewServers(t *testing.T) {
	numTransitions := 70
	numServers := 10
	t.Logf("Target probability %v", Target(numTransitions*numServers*numServers, LinkOverflowProbability))
	for numMessages := 10; numMessages <= 10000000; numMessages *= 10 {
		size := BinSize2(numTransitions, numServers, numMessages, LinkOverflowProbability)
		t.Logf("%d messages, Expectation: %f", numMessages, float64(numMessages)/float64(numServers*numServers))
		t.Logf("Bin size: %d", size)
	}
}

func TestBinSizeFewMessagesManyServers(t *testing.T) {
	numTransitions := 70
	for numServers := 10; numServers <= 1000; numServers *= 10 {
		numMessages := 100
		t.Logf("Target probability %v", Target(numTransitions*numServers*numServers, LinkOverflowProbability))
		size := BinSize2(numTransitions, numServers, numMessages, LinkOverflowProbability)
		t.Logf("%d messages, Expectation: %f", numMessages, float64(numMessages)/float64(numServers*numServers))
		t.Logf("Bin size: %d", size)
	}
}

func TestGroupSize(t *testing.T) {
	fs := []float64{0.01, 0.1, 0.2, 0.25, 0.4, 0.5}
	for _, f := range fs {
		t.Logf("%f: %d", f, GroupSizeWithReplacement(1, f))
	}
	for _, f := range fs {
		t.Logf("%f: %d", f, GroupSizeWithReplacement(10, f))
	}
}

// func TestNumLayers2(t *testing.T) {
// 	fs := []float64{0.01, 0.1, 0.2, 0.25, 0.4, 0.5}
// 	for _, f := range fs {
// 		t.Logf("%f: %d", f, NumLayersMethod2(1000, f))
// 	}
// 	for _, f := range fs {
// 		t.Logf("%f: %d", f, NumLayersMethod2(1000000, f))
// 	}
// }

func TestNumLayers(t *testing.T) {
	fs := []float64{0.01, 0.1, 0.2, 0.25, 0.4, 0.5}
	for _, f := range fs {
		t.Logf("%f: %d", f, NumLayers(1000, f))
	}
	for _, f := range fs {
		t.Logf("%f: %d", f, NumLayers(1000000, f))
	}
}

func TestOverflowProbability(t *testing.T) {
	numTransitions := 60
	numServers := 128
	numMessages := 1000000
	for overflow := -1; overflow >= -64; overflow *= 2 {
		size := BinSize2(numTransitions, numServers, numMessages, overflow)
		size2 := BinSize3(numTransitions, numServers, numMessages, overflow)
		t.Logf("%d messages, Expectation: %f", numMessages, float64(numMessages)/float64(numServers*numServers))
		t.Logf("Bin size: %d %d", size, size2)
	}
}
