set -e

# python3 aws_latency.py 0 2 0 0
python3 aws_bandwidth.py 0 500 0 0

../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.2 --runtype 3
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 93 --binsize 14 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 150 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 250000 --numlayers 97 --binsize 43 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 150 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 500000 --numlayers 98 --binsize 68 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 150 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 750000 --numlayers 98 --binsize 90 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 150 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 1000000 --numlayers 99 --binsize 111 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 150 --messagesize 10000 --outfile messagesNet.json

../coordinator/coordinator --numservers 32 --numclientservers 32 --numusers 10000 --numlayers 92 --binsize 31 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 100 --latency 150 --messagesize 10000 --outfile messagesNet.json

../coordinator/coordinator --numservers 32 --numclientservers 32 --f 0.2 --runtype 3 --numusers 10000
../coordinator/coordinator --numservers 32 --numclientservers 32 --numusers 10000 --numlayers 92 --binsize 31 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 100 --latency 0 --messagesize 10000 --outfile messagesNet.json

python3 aws_bandwidth.py 0 500 0 0
../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.2 --runtype 3
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 93 --binsize 14 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 250000 --numlayers 97 --binsize 43 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 500000 --numlayers 98 --binsize 68 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 750000 --numlayers 98 --binsize 90 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 1000000 --numlayers 99 --binsize 111 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 500 --latency 0 --messagesize 10000 --outfile messagesMet.json


python3 aws_bandwidth.py 0 1000 0 0
../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.2 --runtype 3
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 93 --binsize 14 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 1000 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 250000 --numlayers 97 --binsize 43 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 1000 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 500000 --numlayers 98 --binsize 68 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 1000 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 750000 --numlayers 98 --binsize 90 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 1000 --latency 0 --messagesize 10000 --outfile messagesNet.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 1000000 --numlayers 99 --binsize 111 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 1000 --latency 0 --messagesize 10000 --outfile messagesMet.json
