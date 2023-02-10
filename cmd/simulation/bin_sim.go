package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/alexflint/go-arg"
	"github.com/gonum/stat"
	"github.com/simonlangowski/lightning1/config"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

var args struct {
	NumServers int `default:"0"`
	NumUsers   int `default:"0"`
	NumLayers  int `default:"0"`
	NumTrials  int `default:"100"`
	Type       int `default:"0"`
	Param      int `default:"0"`
}

type output struct {
	Args         interface{}
	Results      []int
	Mean         float64
	Stddev       float64
	Quantiles    []int
	EmpiricSizes []int
	EmpiricProbs []float64
	TheorySizes  []int
	TheoryProbs  []float64
}

var quantiles = []float64{0.95, 0.99, 0.999}

func main() {
	f, err := os.OpenFile("res.txt", os.O_RDWR|os.O_APPEND, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	arg.MustParse(&args)
	results := BinSimMany(args.NumServers, args.NumUsers, args.NumLayers, args.NumTrials, args.Type, args.Param)
	var values plotter.Values
	max := 0
	for _, r := range results {
		values = append(values, float64(r))
		if r > max {
			max = r
		}
	}
	m, s := stat.MeanStdDev(values, nil)
	log.Printf("Mean: %v, stddev: %v", m, s)
	log.Printf("Max is %v, %v messages per server", max, max*args.NumServers)
	quants := make([]int, len(quantiles))
	sort.Float64s(values)
	for i, q := range quantiles {
		quants[i] = int(stat.Quantile(q, stat.Empirical, values, nil))
		totalDummies := quants[i]*args.NumServers*args.NumServers - args.NumUsers
		log.Printf("%v: %v with %v messages per server, %v total", q, quants[i], quants[i]*args.NumServers, totalDummies)
	}
	empiricSizes := make([]int, 0)
	empiricProbs := make([]float64, 0)
	empirics := make(plotter.XYs, 0)
	for binSize := int(math.Floor(m)); binSize <= max; binSize++ {
		failProb := 1 - stat.CDF(float64(binSize), stat.Empirical, values, nil)
		empiricSizes = append(empiricSizes, binSize)
		empiricProbs = append(empiricProbs, failProb)
		empirics = append(empirics, plotter.XY{X: failProb, Y: float64(binSize)})
	}
	theorySizes := make([]int, 0)
	theoryProbs := make([]float64, 0)
	theorys := make(plotter.XYs, 0)
	for overflowProbaility := 1; overflowProbaility <= 16; overflowProbaility++ {
		requiredSize := config.BinSize2(args.NumLayers, args.NumServers, args.NumUsers, -overflowProbaility)
		theoryProb := math.Exp2(-float64(overflowProbaility))
		theorySizes = append(theorySizes, requiredSize)
		theoryProbs = append(theoryProbs, theoryProb)
		theorys = append(theorys, plotter.XY{X: theoryProb, Y: float64(requiredSize)})
	}
	log.Printf("%v %v %v %v", empiricSizes, empiricProbs, theorySizes, theoryProbs)

	o := output{
		Args:         args,
		Results:      results,
		Mean:         m,
		Stddev:       s,
		Quantiles:    quants,
		EmpiricSizes: empiricSizes,
		EmpiricProbs: empiricProbs,
		TheorySizes:  theorySizes,
		TheoryProbs:  theoryProbs,
	}

	b, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}
	f.WriteString("\n")

	p := plot.New()
	p.X.Label.Text = "Failure probability"
	p.Y.Label.Text = "Required size"
	err = plotutil.AddLinePoints(p, "Empirical", empirics, "Theory", theorys)
	if err != nil {
		panic(err)
	}
	// Save the plot to a PNG file.
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "points.png"); err != nil {
		panic(err)
	}
	histPlot(values, args.NumTrials)
}

func histPlot(values plotter.Values, numBins int) {
	p := plot.New()
	p.Title.Text = "histogram plot"
	hist, err := plotter.NewHist(values, numBins)
	if err != nil {
		panic(err)
	}
	p.Add(hist)

	if err := p.Save(3*vg.Inch, 3*vg.Inch, "hist.png"); err != nil {
		panic(err)
	}
}

// return a histogram of observed bin sizes
func BinSimMany(numServers, numMessages, numLayers, numTrials, t, param int) []int {
	results := make([]int, numTrials)
	jobs := make(chan int)
	wg := sync.WaitGroup{}
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			for {
				j, ok := <-jobs
				if !ok {
					break
				}
				// random initial configuration
				initialConfig := make([]int, numServers)
				r := config.NewPRGShuffler(rand.Reader)
				for i := 0; i < numMessages; i++ {
					initialConfig[r.Intn(numServers)]++
				}

				max := 0
				for l := 0; l < numLayers; l++ {
					var maxBinInOneLayer int
					if t == 0 {
						maxBinInOneLayer, initialConfig = PlainShuffle(initialConfig) //BinSimBase(numServers, numMessages)
					} else if t == 1 {
						cols := numServers / param
						maxBinInOneLayer, initialConfig = Dim2Shuffle(param, cols, initialConfig)
					} else if t == 2 {
						maxBinInOneLayer, initialConfig = ApplyButterfly(param, initialConfig)
					} else {
						panic("Unrecognized type")
					}
					if maxBinInOneLayer > max {
						max = maxBinInOneLayer
					}
				}
				results[j] = max
			}
			wg.Done()
		}()
	}
	for i := 0; i < numTrials; i++ {
		jobs <- i
		if i%10 == 0 {
			log.Print(i)
		}
	}
	close(jobs)
	wg.Wait()
	return results
}

func BinSimBase(numServers, numMessages int) int {
	numLinks := numServers * numServers
	bins := make([]int, numLinks)
	r := config.NewPRGShuffler(rand.Reader)
	for i := 0; i < numMessages; i++ {
		bins[r.Intn(numLinks)] += 1
	}
	max := 0
	for _, count := range bins {
		if count > max {
			max = count
		}
	}
	return max
}

// An iterated butterfly network
// I don't know how many layers it would take
// simulate one butterfly iteration
// note that this is log(n) layers
// Basically this simulates a plain shuffle in log(n) layers?
func ApplyButterfly(numBits int, initialConfig []int) (int, []int) {
	numServers := len(initialConfig)
	locations := make([][]int, numServers)
	nextLocations := make([][]int, numServers)
	for i := range locations {
		locations[i] = make([]int, 2)
		locations[i][0] = initialConfig[i]
		nextLocations[i] = make([]int, 2)
	}
	maxSeen := 0
	r := config.NewPRGShuffler(rand.Reader)
	for i := 0; i < numBits; i++ {
		for s := 0; s < numServers; s++ {
			goTo := invertBit(i, s)
			myMessages := locations[s][0] + locations[s][1]
			for m := 0; m < myMessages; m++ {
				if r.Intn(2) == 0 {
					// stay
					nextLocations[s][0]++
				} else {
					// go
					nextLocations[goTo][1]++
				}
			}
		}
		locations = nextLocations
		nextLocations := make([][]int, numServers)
		for k, l := range locations {
			if l[0] > maxSeen {
				maxSeen = l[0]
			}
			if l[1] > maxSeen {
				maxSeen = l[1]
			}
			nextLocations[k] = make([]int, 2)
		}
	}
	for i := range nextLocations {
		initialConfig[i] = locations[i][0] + locations[i][1]
	}
	return maxSeen, initialConfig
}

func invertBit(bit, num int) int {
	m := (1 << bit)
	return num ^ m
}

// Each server is labeled (1,1) .. (sqrt(n), sqrt(n))
// If we first shuffle on rows, then on columns, it will take 2L layers (and this does 2 at once)
// If 3 coordinates, 3L, etc.
// Can probably reduce by intermingling shuffle directions
func Dim2Shuffle(rows, cols int, initialConfig []int) (int, []int) {
	mat := make([][]int, rows)
	for i := range mat {
		mat[i] = make([]int, cols)
	}
	numServers := len(initialConfig)
	for i := 0; i < numServers; i++ {
		sRow := i / cols
		sCol := i % cols
		mat[sRow][sCol] = initialConfig[i]
	}
	// swap rows
	max := 0
	for i, row := range mat {
		var c int
		c, mat[i] = PlainShuffle(row)
		if c > max {
			max = c
		}
	}
	// transpose
	tMat := transpose(mat)
	// swap cols
	for j, col := range tMat {
		var c int
		c, tMat[j] = PlainShuffle(col)
		if c > max {
			max = c
		}
	}
	mat = transpose(tMat)
	for i := 0; i < numServers; i++ {
		sRow := i / cols
		sCol := i % cols
		initialConfig[i] = mat[sRow][sCol]
	}
	return max, initialConfig
}

func PlainShuffle(initialConfig []int) (int, []int) {
	numServers := len(initialConfig)
	bins := make([][]int, numServers)
	for i := 0; i < numServers; i++ {
		bins[i] = make([]int, numServers)
	}
	r := config.NewPRGShuffler(rand.Reader)
	for i := 0; i < numServers; i++ {
		for j := 0; j < initialConfig[i]; j++ {
			nextServer := r.Intn(numServers)
			bins[nextServer][i] += 1
		}
	}
	max := 0
	for i := 0; i < numServers; i++ {
		t := 0
		for j := 0; j < numServers; j++ {
			l := bins[i][j]
			if l > max {
				max = l
			}
			t += l
		}
		initialConfig[i] = t
	}
	return max, initialConfig
}

func transpose(in [][]int) [][]int {
	rows := len(in)
	cols := len(in[0])
	out := make([][]int, cols)
	for j := range out {
		out[j] = make([]int, rows)
		for i := range out[j] {
			out[j][i] = in[i][j]
		}
	}
	return out
}
