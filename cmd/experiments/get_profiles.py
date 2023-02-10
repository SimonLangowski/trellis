import subprocess
import sys

ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

def retrieveLog(dest):
    cmd = " ".join(['scp',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/lkey',
                    'ec2-user@' + dest + ":~/*.pprof",
                    '.'
    ])
    return subprocess.Popen(cmd, stdout=sys.stdout, stderr=sys.stderr, stdin=subprocess.PIPE, shell=True)

processes = [retrieveLog(d) for d in ips]
exit_codes = [p.wait() for p in processes]
