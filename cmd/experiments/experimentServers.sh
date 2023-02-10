set -e

# python3 aws_latency.py 0 2 0 0
# python3 aws_bandwidth.py 0 200 0 0

../coordinator/coordinator --numservers 256 --numclientservers 256 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 256 --numclientservers 256 --numusers 100000 --numlayers 64 --binsize 13 --f 0.1 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 256 --numclientservers 256 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 256 --numclientservers 256 --numusers 100000 --numlayers 95 --binsize 13 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 256 --numclientservers 256 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 256 --numclientservers 256 --numusers 100000 --numlayers 135 --binsize 13 --f 0.3 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json

../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 64 --binsize 26 --f 0.1 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 95 --binsize 26 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 128 --numclientservers 128 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 128 --numclientservers 128 --numusers 100000 --numlayers 135 --binsize 26 --f 0.3 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json

../coordinator/coordinator --numservers 64 --numclientservers 64 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 64 --numclientservers 64 --numusers 100000 --numlayers 64 --binsize 58 --f 0.1 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 64 --numclientservers 64 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 64 --numclientservers 64 --numusers 100000 --numlayers 95 --binsize 58 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 64 --numclientservers 64 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 64 --numclientservers 64 --numusers 100000 --numlayers 135 --binsize 58 --f 0.3 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json

../coordinator/coordinator --numservers 32 --numclientservers 32 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 32 --numclientservers 32 --numusers 100000 --numlayers 64 --binsize 157 --f 0.1 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 32 --numclientservers 32 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 32 --numclientservers 32 --numusers 100000 --numlayers 95 --binsize 157 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 32 --numclientservers 32 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 32 --numclientservers 32 --numusers 100000 --numlayers 135 --binsize 157 --f 0.3 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json

../coordinator/coordinator --numservers 16 --numclientservers 16 --f 0.1 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 16 --numclientservers 16 --numusers 100000 --numlayers 64 --binsize 499 --f 0.1 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 16 --numclientservers 16 --f 0.2 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 16 --numclientservers 16 --numusers 100000 --numlayers 95 --binsize 499 --f 0.2 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json
../coordinator/coordinator --numservers 16 --numclientservers 16 --f 0.3 --runtype 3 --numusers 100000
../coordinator/coordinator --numservers 16 --numclientservers 16 --numusers 100000 --numlayers 135 --binsize 499 --f 0.3 --runtype 2 --skippathgen --nocheck --bandwidth 200 --latency 150 --messagesize 10000 --outfile messagesServers.json

