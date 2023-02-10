import subprocess
import sys
from threading import Thread

def printHelp():
    print("Argument 1:")
    print("0 - Apply change")
    print("1 - Remove changes")
    print("Argument 2:")
    print("Bandwidth in mbit")
    exit()

if (len(sys.argv) < 2):
    printHelp()

mode = "add"
if sys.argv[1] == "1":
    mode = "del"
elif sys.argv[1] != "0":
    printHelp()

b = sys.argv[2]

simulated_regions = ['us-east-1', 'us-west-2', 'eu-north-1', 'ap-northeast-1', 'eu-west-2', 'ap-southeast-2', 'sa-east-1']
tor_regions = ['us-east-1', 'us-west-2', 'eu-north-1', 'eu-central-1']
bandwidth = b

def getSection(ip):
    return int(ip.split(".")[2])

def regionRate(ip):
    return bandwidth

def runRemoteCommand(dest, cmd):
    rcmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/lkey',
                    'ec2-user@' + dest,
                    cmd
    ])
    return subprocess.run(rcmd, stdout=sys.stdout, stderr=sys.stderr, stdin=subprocess.PIPE, shell=True)

device = "lo"

# one latency for each region
def setBandwidthPattern(server, rate):
    # https://wiki.linuxfoundation.org/networking/netem
    # https://serverfault.com/questions/916457/tc-netem-filter-explenation
    # http://tcn.hypert.net/tcmanual.pdf
    # Root qdisc
    runRemoteCommand(server, f"sudo tc qdisc {mode} dev {device} handle 2: root htb default 1")
    # Root class
    runRemoteCommand(server, f"sudo tc class {mode} dev {device} parent 2: classid 2:1 htb rate {rate}mbit")
    # Also should set the tcp receive and send buffer sizes larger

ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

threads = [Thread(target=setBandwidthPattern, args=(i, regionRate(i))) for i in ips]
for t in threads:
    t.start()
for t in threads:
    t.join()
