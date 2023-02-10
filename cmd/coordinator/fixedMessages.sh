#!/bin/bash
set -e

./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.1
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.1 --outfile res$1-$2.json --notes $2 --nocheck
./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.2
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.2 --outfile res$1-$2.json --notes $2 --nocheck
./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.3
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers 2000000 --f 0.3 --outfile res$1-$2.json --notes $2 --nocheck
