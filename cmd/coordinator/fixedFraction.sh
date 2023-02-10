#!/bin/bash
set -e

#./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 10000 --f 0.1
#./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 10000 --f 0.1 --outfile res$1.json --notes $2 --latency $3 --bandwidth $4
#./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 100000 --f 0.1
#./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 100000 --f 0.1 --outfile res$1.json --notes $2 --nocheck --latency $3 --bandwidth $4
./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 1000000 --f 0.1
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 1000000 --f 0.1 --outfile res$1.json --notes $2 --nocheck --latency $3 --bandwidth $4
./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 10000000 --f 0.1
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 10000000 --f 0.1 --outfile res$1.json --notes $2 --nocheck --latency $3 --bandwidth $4
