import subprocess

ips = []
with open('aws.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

def transferFile(dest, fileName):
    cmd = " ".join(['scp',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/aws/slangows.pem',
                    fileName,
                    'ec2-user@' + dest + ":~/go/bin/" + fileName
    ])
    return subprocess.Popen(cmd, stdout=None, stderr=None, stdin=subprocess.PIPE, shell=True)

def transferToAll(dests, fileName):
    processes = [transferFile(d, fileName) for d in dests]
    exit_codes = [p.wait() for p in processes]
    for i in exit_codes:
        if i != 0:
            print("Error transfering file: " + fileName)

transferToAll(ips, "servers.json")
transferToAll(ips, "groups.json")
transferToAll(ips, "clients.json")