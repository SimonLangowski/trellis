package config

/*
Code to create anytrust groups

*/

import (
	"log"
	"math"
	"math/big"
)

// Used to calculate the number of servers required to see the boomerang
// Since servers can be repeatedly used each is independently likely to be adversarial
func GroupSizeWithReplacement(nGroups int, f float64) int {
	target := Target(nGroups, AnytrustGroupSecurityFactor)
	size := math.Ceil(math.Log2(target) / math.Log2(f))
	return int(size)
}

// An upper bound on the individual group probability
// When multiple anytrust groups are selected and we want to bound
// the probability that all are adversarial
func Target(numGroups int, securityFactor int) float64 {
	// 1 - (1- 2^-lambda)^(1/numGroups)
	// This turns into 1 + - 2^-lambda when we take log(1+x)
	base := -math.Exp2(float64(securityFactor))
	// take natural log so we can take exponent of arbitrary base
	exponent := math.Log1p(base) / float64(numGroups)
	// we now compute e^(log(1-2^-lambda)/numGroups) - 1, and then negate the answer
	return -math.Expm1(exponent)
}

// Calculate the size of an anytrust group without replacement
// E.g if f=0.01 and N=100 then replacement would require 11 but pigeonhole principle means only size 2 is required
func GroupSizeWithoutReplacement(nServers, nGroups int, f float64) int {
	target := Target(nGroups, AnytrustGroupSecurityFactor)
	// pick servers until the probability is small enough
	advCount := int64(math.Ceil(float64(nServers) * f))
	totalCount := int64(nServers)
	// Compute the probability of picking all adversaries until less than target
	// (nf / n) * (nf - 1)/(n - 1) * (nf - 2)/(n - 2) ...
	probability := new(big.Rat).SetFrac64(advCount, totalCount)
	term := new(big.Rat)
	for groupSize := 1; groupSize < nServers; groupSize++ {
		current, _ := probability.Float64()
		if current < target {
			return groupSize
		}
		advCount--
		totalCount--
		term.SetFrac64(advCount, totalCount)
		probability = probability.Mul(probability, term)
	}
	return nServers
}

func CalcGroupSize(nServers, nGroups int, f float64) int {
	if Model == 1 {
		return GroupSizeWithReplacement(nGroups, f)
	} else if Model == 2 {
		return GroupSizeWithoutReplacement(nServers, nGroups, f)
	} else {
		return 0
	}
}

func CreateRandomGroups(nGroups int, f float64, serverIds []int64) map[int64]*Group {
	n := len(serverIds)
	size := CalcGroupSize(len(serverIds), nGroups, f)
	if size > n { // this should never happen in practice, but useful for testing..
		size = n
	}
	return CreateRandomGroupsWithSize(nGroups, size, serverIds)
}

func CreateRandomGroupsWithSize(nGroups, size int, serverIds []int64) map[int64]*Group {
	log.Printf("Warning: fix group distribution")
	n := len(serverIds)
	groups := make(map[int64]*Group)
	// seeding for determinstic tests; a public unbiased source of randomness should be used
	r := SeededShuffler()
	for i := 0; i < nGroups; i++ {
		indices := r.SelectRandom(n, size)
		servers := make([]int64, size)
		for i, j := range indices {
			servers[i] = serverIds[j]
		}
		groups[int64(i)] = &Group{Gid: int64(i), Servers: servers}
	}
	return groups
}

func CreateSeparateGroupsWithSize(nGroups, size int, serverIds []int64) map[int64]*Group {
	n := len(serverIds)
	// Select from disjoint sets of a random permutation
	s := SeededShuffler()
	permutation := s.Perm(n)
	groups := make(map[int64]*Group)
	index := 0
	for i := 0; i < nGroups; i++ {
		servers := make([]int64, size)
		used := make(map[int64]bool)
		for j := 0; j < size; j++ {
			if index >= len(permutation) {
				permutation = append(permutation, s.Perm(n)...)
			}
			servers[j] = int64(permutation[index])
			if used[servers[j]] {
				// duplicate
				// we want to reselect the permutation so that this didn't happen
				// this means randomly reshuffling the duplicate into the remaining permutation
				// choose a random value to swap with and try again
				swapLocation := int(s.UInt64()%uint64(len(permutation)-index-1)) + index + 1
				permutation[index], permutation[swapLocation] = permutation[swapLocation], permutation[index]
				j--
				continue
			} else {
				used[servers[j]] = true
			}
			index++
		}
		groups[int64(i)] = &Group{Gid: int64(i), Servers: servers}
	}
	return groups
}

// assign each server to at most 1 group
func CalcFewGroups(f float64, n int) (int, int) {
	nGroups := n // every server in its own group
	for ; nGroups >= 1; nGroups-- {
		size := CalcGroupSize(n, nGroups, f)
		actualNumGroups := n / size
		if actualNumGroups >= nGroups {
			break
		}
	}
	if nGroups < 1 {
		nGroups = 1
	}

	return CalcGroupSize(n, nGroups, f), nGroups
}

func CalcFewGroups2(f float64, n int) (int, int) {
	bestCost := float64(n) // 1 group of size n, and therfore maxGroups is 1
	bestNGroups := 1
	for nGroups := 1; nGroups <= n; nGroups++ {
		groupSize := CalcGroupSize(n, nGroups, f)
		cost := GroupSizeCost(groupSize, nGroups, n)
		// log.Printf("%d groups of %d cost %f", nGroups, groupSize, cost)
		if cost < bestCost {
			bestCost = cost
			bestNGroups = nGroups
		}
	}
	return CalcGroupSize(n, bestNGroups, f), bestNGroups
}

func GroupSizeCost(groupSize, numGroups, numServers int) float64 {
	// the latency cost of groups is
	// each group does groupSize work
	// each group does 1/numGroups of the work

	// the latency is defined by the member who is in the most groups
	// ceiling division
	maxGroups := (numGroups*groupSize + (numServers - 1)) / numServers
	return float64(maxGroups*groupSize) / float64(numGroups)
}
