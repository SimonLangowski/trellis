set -e

python3 aws_latency.py 0 2 0 0
python3 aws_bandwidth.py 0 200 0 0

../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 10000 --numlayers 92 --binsize 9 --f 0.2 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 93 --binsize 14 --f 0.2 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 50000 --numlayers 94 --binsize 18 --f 0.2 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 75000 --numlayers 95 --binsize 22 --f 0.2 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 95 --binsize 26 --f 0.2 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json

../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 10000 --numlayers 62 --binsize 9 --f 0.1 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 63 --binsize 14 --f 0.1 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 50000 --numlayers 64 --binsize 18 --f 0.1 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 75000 --numlayers 64 --binsize 22 --f 0.1 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 64 --binsize 26 --f 0.1 --runtype 2 --bandwidth 2000 --latency 150 --outfile messagesPath.json

../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 10000 --numlayers 129 --binsize 9 --f 0.3 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 25000 --numlayers 132 --binsize 14 --f 0.3 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 50000 --numlayers 133 --binsize 18 --f 0.3 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 75000 --numlayers 134 --binsize 22 --f 0.3 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 135 --binsize 26 --f 0.3 --runtype 2 --bandwidth 200 --latency 150 --outfile messagesPath.json
