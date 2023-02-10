import subprocess
import sys
from threading import Thread

def printHelp():
    print("Argument 1:")
    print("0 - Apply change")
    print("1 - Remove changes")
    print("Argument 2:")
    print("0 - no latency addition")
    print("1 - Latency on exactly 1 link")
    print("2 - Latency matching real world")
    print("3 - High latency all links")
    exit()

if (len(sys.argv) < 4):
    printHelp()

mode = "add"
if sys.argv[1] == "1":
    mode = "del"
elif sys.argv[1] != "0":
    printHelp()

simulated_regions = ['us-east-1', 'us-west-2', 'eu-north-1', 'eu-central-1']
simulated_latencies = [[2 for _ in range(len(simulated_regions))] for _ in range(len(simulated_regions))]
if sys.argv[2] == "1":
    simulated_latencies[0][1] = 300
    simulated_latencies[1][0] = 300
elif sys.argv[2] == "2":
    simulated_latencies = [
        [ 32,  64, 100, 78],
        [ 64,  32, 150, 140],
        [100, 150,  13, 26],
        [ 78, 140,  26, 13],
    ]
elif sys.argv[2] == "3":
    simulated_latencies = [[300 for _ in range(len(simulated_regions))] for _ in range(len(simulated_regions))]
elif sys.argv[2] == "4":
    l = len(simulated_latencies) - 1
    simulated_latencies[l-1][l] = 300
    simulated_latencies[l][l-1] = 300
elif sys.argv[2] == "5":
    # Every event should happen at a unique time - minimize congestion - well not if you assign 2 servers to the same region
    # There are 7 independent sets (7 choose 2, /3) that can run in parallel (e.g the parallel lines in the graph)
    s = [37.5, 75, 112.5, 150, 187.5, 225, 262.5, 300]
    independent_sets = [
        [(0,1),(1,2),(2,3),(3,4),(4,5),(5,6),(6,0)],
        [(2,6),(3,0),(4,1),(5,2),(6,3),(0,4),(1,5)],
        [(3,5),(4,6),(5,0),(6,1),(0,2),(1,3),(2,4)]
    ]
    # Set main diagonal
    for i in range(7):
        simulated_latencies[i][i] = s[0]
    for i_set in independent_sets:
        for idx, edge in enumerate(i_set):
            simulated_latencies[edge[0]][edge[1]] = s[idx+1]
            # Symmetry
            simulated_latencies[edge[1]][edge[0]] = s[idx+1]
elif sys.argv[2] == "6":
    # One row and column are slow - one server far from the rest
    l = len(simulated_latencies) - 1
    simulated_latencies[l] = [300 for _ in range(len(simulated_regions))]
    for i in range(len(simulated_latencies)):
        simulated_latencies[i][l] = 300
elif sys.argv[2] != "0":
    printHelp()

print(simulated_latencies)

def getSection(ip):
    return int(ip.split(".")[2])

def regionPattern(ip):
    s = getSection(ip)
    if s > 10:
        s = s//16
    return simulated_latencies[s - 1]

def runRemoteCommand(dest, cmd):
    rcmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/lkey',
                    'ec2-user@' + dest,
                    cmd
    ])
    return subprocess.run(rcmd, stdout=sys.stdout, stderr=sys.stderr, stdin=subprocess.PIPE, shell=True)

input_device = "lo"
device = "ifb1"

# one latency for each region
def setLatencyPattern(server, pattern):

    # https://wiki.linuxfoundation.org/networking/netem
    # Due to mechanisms like TSQ (TCP Small Queues), for TCP performance test results to be realistic, netem must be placed on the ingress of the receiver host (see “How can I use netem on incoming traffic?”, below).
    # When you run TCP over large Bandwidth Delay Product links, you need to do some TCP tuning to increase the maximum possible buffer space. 
    runRemoteCommand(server, "sudo modprobe ifb")
    runRemoteCommand(server, f"sudo ip link set dev {device} up")
    runRemoteCommand(server, f"sudo tc qdisc add dev {input_device} ingress")
    runRemoteCommand(server, f"sudo tc filter add dev {input_device} parent ffff: protocol ip u32 match u32 0 0 flowid 1:1 action mirred egress redirect dev {device}")
    # https://serverfault.com/questions/916457/tc-netem-filter-explenation
    # http://tcn.hypert.net/tcmanual.pdf
    # Root qdisc
    runRemoteCommand(server, f"sudo tc qdisc {mode} dev {device} handle 1:0 root htb")
    # Root class
    runRemoteCommand(server, f"sudo tc class {mode} dev {device} parent 1:0 classid 1:1 htb rate 10000Mbps")
    # We assign each region to  a contiguous ip block 172.16.X.0/24, starting with X=1
    block = 1
    for delay in pattern:
        # Leaf class
        runRemoteCommand(server, f"sudo tc class {mode} dev {device} parent 1:0 classid 1:1{block} htb rate 10000Mbps")
        # It is possible to add more complicated distributions, like normal distribution, packet loss, etc. here
        # We add half of the rtt ping to each side
        # We also need a large limit since we have a large latency and bandwidth
        # Pareto distribution adds the rare but large latency values observed
        # Variance determined by comparing with experiment on real network
        # Observed packet loss (communicating between datacenters)
        loss = 0.00001
        delay_pattern = f'netem delay {delay/2}ms {delay/8}ms distribution pareto loss {loss*100}% limit 100000'
        # Leaf qdisc
        runRemoteCommand(server, f"sudo tc qdisc {mode} dev {device} parent 1:1{block} handle {block}0: {delay_pattern}")
        # filter
        filterBlock = f'172.16.{block*16}.0/20'
        runRemoteCommand(server, f"sudo tc filter {mode} dev {device} parent 1:0 protocol ip prio {block} u32 match ip src {filterBlock} flowid 1:1{block}")
        block += 1

ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

threads = [Thread(target=setLatencyPattern, args=(i, regionPattern(i))) for i in ips]
for t in threads:
    t.start()
for t in threads:
    t.join()
