package config

import (
	"log"
	"math"
)

func NumLayers(numusers int, f float64) int {
	// this calculates the dominant term
	r := math.Sqrt(-3*f*f + 4*f)
	l := (ShuffleSecurityFactor*math.Log(2) - math.Log(float64(numusers)) - math.Log(2*r) - math.Log(-f+r+2)) / (math.Log(f/2 + r/2))
	return int(math.Ceil(l + 1))
}

func Chernoff(mu, delta float64) float64 {
	/*
		numerator := math.Exp(delta * mu)
		denominatorExponent := math.Log(1+delta) * mu * (1 + delta)
		denominator := math.Exp(denominatorExponent)
		return numerator / denominator
	*/
	// Use chernoff aproximation to avoid stability issues
	return math.Exp(-delta * delta * mu / (2 + delta))
}

// func BinSize1(numTransitions, numServers, numMessages int) int {
// 	return binsize(numTransitions, numServers*numServers, numMessages)
// }

// // this is just for load balancing - we don't send dummies to groups because they whp have an adversary
// func GroupBinSize(numGroups, numServers, numMessages int) int {
// 	return binsize(1, numGroups*numServers, numMessages)
// }

// func binsize(numTransitions, numChoices, numMessages int) int {
// 	// Think of n^2 * l independent binomial events where numMessages are sampled each with probability 1/n^2
// 	target := Target(numTransitions*numChoices, LinkOverflowProbability)
// 	// log.Printf("Target %v", target)
// 	// We could compute the binomial exactly, but to avoid numerical issues
// 	// We just use a chernoff bound
// 	expectation := float64(numMessages) / float64(numChoices)
// 	// Chernoff bound is monotonic so binary search
// 	size := sort.Search(numMessages, func(numSelected int) bool {
// 		// Chernoff bound assumes delta > 0
// 		// The is equivalent to that the bin size must be larger than the expectation
// 		// which by pigeonhole, it must
// 		delta := (float64(numSelected) / expectation) - 1
// 		if delta <= 0 {
// 			return false
// 		}
// 		return Chernoff(expectation, delta) < target
// 	})
// 	// log.Printf("exp: %v, delta: %v, Probability %v", expectation, (float64(size)/expectation)-1, Chernoff(expectation, (float64(size)/expectation)-1))
// 	return size
// }

func KLDiv(a, p float64) float64 {
	return a*math.Log(a/p) + (1-a)*math.Log((1-a)/(1-p))
}

// probability that Binom(n trials, p success) has less than or equal to k success
func BinomialChernoff(k, n, p float64) float64 {
	// https://en.wikipedia.org/wiki/Binomial_distribution#Tail_bounds
	// https://link.springer.com/content/pdf/10.1007/BF02458840.pdf
	return math.Exp(-n * KLDiv(k/n, p))
}

// probability that Binom(n trials, p success) has greater than or equal to k success
func UpperBinomialChernoff(k, n, p float64) float64 {
	return BinomialChernoff(n-k, n, 1-p)
}

func BinSize2(numTransitions, numServers, numMessages, overflowProbability int) int {
	// Think of n^2 * l independent binomial events where numMessages are sampled each with probability 1/n^2
	// this does a union bound
	target := Target(numTransitions*numServers*numServers, overflowProbability)
	// each link selected with p = 1/n^2
	p := 1 / float64(numServers*numServers)
	for binSize := numMessages; binSize > 0; binSize-- {
		// Find probability that Binom(n = number messages, p) has k greater than binSize
		// stop when the probability exceeds target
		if UpperBinomialChernoff(float64(binSize), float64(numMessages), p) > target {
			// we went too far!
			return binSize + 1
		}
	}
	return 1
}

func BinSize3(numTransitions, numServers, numMessages, overflowProbability int) int {
	// union bound probability all links fail
	// P(fail) < 2^{-lambda} / (n^2*l)
	// log P(fail) = -lambda * ln2 - ln (n^2 * l)
	failProb := -(float64(-overflowProbability)*math.Ln2 + math.Log(float64(numServers*numServers*numTransitions)))
	for binSize := (numMessages / (numServers * numServers)); binSize <= numMessages; binSize++ {
		// Find probability that Binom(n = number messages, p) fails with low probability
		prob := ChernoffHoldingBinomial(binSize, numServers, numMessages)
		if prob < failProb {
			log.Printf("%v %v", prob, failProb)
			// we went too far!
			return binSize
		}
	}
	return numMessages
}

// https://en.wikipedia.org/wiki/Chernoff_bound#Additive_form_(absolute_error)
func ChernoffHoldingBinomial(binsize, numservers, nummessages int) float64 {
	// n in chernoff holding is the m messages
	// 1/numServers^2 links probability of being chosen
	p := 1 / float64(numservers*numservers)
	// Given the bin size, calculate epsilon
	epsilon := float64(binsize)/float64(nummessages) - p
	// this is the exponent e^-D(p+e||p)*n
	return -KLDiv(p+epsilon, p) * float64(nummessages)
}
