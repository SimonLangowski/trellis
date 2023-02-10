package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gonum/stat"
)

type Message struct {
	sender      int
	destination int
	round       int
	layer       int
	step        string
}

func main() {
	fn := os.Args[1]
	logs := parseFile(fn)
	sender, numServers := readParams(logs)
	histData := make([]time.Duration, 0)
	for round := 0; round < 5; round++ {
		for layer := 0; layer < 91; layer++ {
			t := sendingTime(logs, round, layer, sender, numServers)
			// t := layerTime(logs, round, layer, sender, numServers)
			histData = append(histData, t)
		}
	}
	log.Printf("%v", histData)
	floatData := make([]float64, len(histData))
	for i, e := range histData {
		floatData[i] = float64(e)
	}
	m := stat.Mean(floatData, nil)
	log.Printf("Avg: %v", time.Duration(m))
	// plotHist(histData)
}

const layout = "15:04:05.99999999"

var reg, _ = regexp.Compile("[^A-Za-z0-9]+")

func processLine(line string) (Message, time.Time) {
	parts := strings.Split(line, " ")
	t, err := time.Parse(layout, parts[1])
	if err != nil {
		panic(err)
	}
	return Message{
		step:        parts[2],
		round:       numPart(parts[6]),
		layer:       numPart(parts[7]),
		sender:      numPart(parts[8]),
		destination: numPart(parts[10]),
	}, t
}

func numPart(s string) int {
	parts := strings.Split(s, ":")
	parts[1] = reg.ReplaceAllString(parts[1], "")
	i, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}
	return i
}

func parseFile(fn string) map[Message]time.Time {
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	logs := make(map[Message]time.Time)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		m, t := processLine(scanner.Text())
		logs[m] = t
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return logs
}

func readParams(logs map[Message]time.Time) (int, int) {
	sender := 0
	numServers := 0
	for k := range logs {
		if k.destination > numServers {
			numServers = k.destination
		}
		sender = k.sender
	}
	return sender, numServers + 1
}

func sendingTime(logs map[Message]time.Time, round, layer, sender, numServers int) time.Duration {
	start := getOccurenceOf(logs, round, layer, sender, numServers, "Connect", true)
	end := getOccurenceOf(logs, round, layer, sender, numServers, "Waiting", false)
	return end.Sub(start)
}

func layerTime(logs map[Message]time.Time, round, layer, sender, numServers int) time.Duration {
	start := getOccurenceOf(logs, round, layer, sender, numServers, "Connect", true)
	end := getOccurenceOf2(logs, round, layer, 1, numServers, "Processed", false)
	return end.Sub(start)
}

func getOccurenceOf(logs map[Message]time.Time, round, layer, sender, numServers int, step string, first bool) time.Time {
	var ex time.Time
	b := true
	for dest := 0; dest < numServers; dest++ {
		m := Message{
			round:       round,
			layer:       layer,
			sender:      sender,
			destination: dest,
			step:        step,
		}
		t := logs[m]
		if t.IsZero() {
			continue
		}
		if b {
			ex = t
			b = false
		} else if first && t.Before(ex) {
			ex = t
		} else if !first && t.After(ex) {
			ex = t
		}
	}
	return ex
}

func getOccurenceOf2(logs map[Message]time.Time, round, layer, numChunks, numServers int, step string, first bool) time.Time {
	var ex time.Time
	b := true
	for s := 0; s < numServers; s++ {
		m := Message{
			round:       round,
			layer:       layer,
			sender:      s,
			destination: numChunks,
			step:        step,
		}
		t := logs[m]
		if b {
			ex = t
			b = false
		} else if first && t.Before(ex) {
			ex = t
		} else if !first && t.After(ex) {
			ex = t
		}
	}
	return ex
}
