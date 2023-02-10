#!/bin/bash
set -e

./coordinator --runtype 3 --numservers $1 --numclientservers $1 --numusers $2 --f $3
./coordinator --runtype 2 --numservers $1 --numclientservers $1 --numusers $2 --f $3 --outfile res$1$4.json --nocheck
