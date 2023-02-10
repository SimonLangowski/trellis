import subprocess
import sys
from threading import Thread
import numpy

mode = "add"
if (len(sys.argv) > 1) and eval(sys.argv[1]):
    mode = "del"

simulated_regions = ['us-east-1', 'us-west-2', 'eu-north-1', 'ap-northeast-1', 'eu-west-2', 'ap-southeast-2', 'sa-east-1']
# We use 40ms as the latency within a region
simulated_latencies = [
    [ 40,  64, 100, 170,  78, 220, 118],
    [ 64,  40, 150,  85, 140, 146, 170],
    [100, 150,  40, 236,  26, 280, 230],
    [170,  85, 236,  40, 224, 232, 268],
    [ 78, 140,  26, 224,  40, 254, 200],
    [220, 146, 280, 232, 254,  40, 308],
    [118, 170, 230, 268, 200, 308,  40]
]

def getSection(ip):
    return int(ip.split(".")[2])

def regionPattern(ip):
    return simulated_latencies[getSection(ip) - 1]

def runRemoteCommand(dest, cmd):
    rcmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/lkey',
                    'ec2-user@' + dest,
                    cmd
    ])
    return subprocess.run(rcmd, stdout=sys.stdout, stderr=sys.stderr, stdin=subprocess.PIPE, shell=True)

def Prioritize(delays):
    # Give the highest latency highest priority (which is represented by a smaller number)
    # This will starve slower latencies so that higher latency packets are sent first
    minPrio = 3
    inversePriorities = numpy.argsort(delays)
    priorities = [i for i in range(len(delays))]
    for pos in reversed(inversePriorities):
        priorities[pos] = minPrio
        minPrio += 1
    return priorities

device = "eth0"

# one latency for each region
def setLatencyPattern(server, pattern):
    # https://serverfault.com/questions/351835/tc-prio-qdisc-for-priorization-of-mysql-traffic
    # https://man7.org/linux/man-pages/man8/tc-prio.8.html
    # Root qdisc
    runRemoteCommand(server, f"sudo tc qdisc {mode} dev {device} handle 1:0 root prio bands 10 priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1")
    block = 1
    prio_pattern = Prioritize(pattern)
    for prio in prio_pattern:
        # filter
        filterBlock = f'172.16.{block}.0/24'
        runRemoteCommand(server, f"sudo tc filter {mode} dev {device} protocol ip parent 1:0 prio {prio} u32 match ip dst {filterBlock} flowid 1:1{block}")
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
