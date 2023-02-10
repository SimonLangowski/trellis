import subprocess
import sys
from threading import Thread


def runRemoteCommand(dest, cmd):
    rcmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/lkey',
                    'ec2-user@' + dest,
                    cmd
    ])
    return subprocess.run(rcmd, stdout=sys.stdout, stderr=sys.stderr, stdin=subprocess.PIPE, shell=True)

def pullAndBuild(ip):
    runRemoteCommand(ip, "cd Lightning1; git pull")
    runRemoteCommand(ip, "cd Lightning1/cmd/server; go install && go build")
    runRemoteCommand(ip, "cd Lightning1/cmd/client; go install && go build")

ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

threads = [Thread(target=pullAndBuild, args=(i,)) for i in ips]
for t in threads:
    t.start()
for t in threads:
    t.join()
