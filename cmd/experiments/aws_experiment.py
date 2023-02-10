import argparse
import os
import signal
import subprocess
import threading
import time

parser = argparse.ArgumentParser(description='Start remote test.')
parser.add_argument('servers', type=int,
                    help='number of servers')
parser.add_argument('clients', type=int,
                    help='number of client servers')
parser.add_argument('users', type=int,
                    help='number of users')
parser.add_argument('--kill', type=int, default=1,
                    help='kill all process first [0|1]')
parser.add_argument('--network', type=int, default=0, required=False,
                    help="Use artificially slow network")

args = parser.parse_args()
kill = args.kill == 1

ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

gopath = os.getenv('GOPATH')
src_dir = 'github.com/simonlangowski/lightning1'

# os.system('go install %s/cmd/coordinator' % (src_dir))
# os.system('go install %s/cmd/config' % (src_dir))
# os.system('go install %s/cmd/server' % (src_dir))
# os.system('go install %s/cmd/client' % (src_dir))

server_ips = []
client_ips = []
for i in range(args.servers):
    server_ips.append((ips[i],8000))
for i in range(args.clients):
    if len(ips) >= args.servers + args.clients:
        # if there are enough machines, use separate servers
        client_ips.append((ips[args.servers+i],9000))
    else:
        client_ips.append((ips[i],9000))

group_file = '~/go/bin/groups.json'
layer_file = '~/go/bin/servers.json'
client_file = '~/go/bin/clients.json'

addr = "%s:%d"

def remotehost(dest, c):
    cmd = "ssh -o StrictHostKeyChecking=no -i ~/.ssh/aws/slangows.pem ec2-user@%s '%s'" % (dest, c)
    return subprocess.Popen(cmd, stdout=None, stderr=None, stdin=subprocess.PIPE, shell=True)

def killall(ips):
    ks = []
    for ip in ips:
        t1 = threading.Thread(target=remotehost, args=(ip, 'killall server 2>/dev/null',))
        t2 = threading.Thread(target=remotehost, args=(ip, 'killall clientRunner 2>/dev/null',))
        t1.start()
        t2.start()
        ks.extend([t1, t2])

    for t in ks:
        t.join()
    print("processes killed")

if kill:
    killall(ips)

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

def simulateNetwork():
    command = "sudo tc qdisc add dev eth0 root tbf rate 100mbit latency 100ms burst 10000"
    processes = [remotehost(d[0], command) for d in server_ips]
    [p.wait() for p in processes]

if args.network > 0:
    simulateNetwork()

# launch the servers
print("launching servers")
sprocesses = []
for i in range(args.servers):
    cmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    'ec2-user@' + ips[i][0],
                    '\'~/go/bin/server ' + \
                    layer_file + ' ' + \
                    group_file + ' ' + \
                    addr % server_ips[i] +'\''])
    print('Running %s' % cmd)
    p = subprocess.Popen(cmd, stdout=None, stderr=None, stdin=subprocess.PIPE, shell=True)
    sprocesses.append(p)
time.sleep(0.5)

# launch the clients
print("launching clients")
cprocesses = []
print(args.clients)
for i in range(args.clients):
    cmd = " ".join(['ssh',
                    '-o StrictHostKeyChecking=no',
                    '-i ~/.ssh/aws/slangows.pem',
                    'ec2-user@' + client_ips[i][0],
                    '\'~/go/bin/clientRunner ' + \
                    client_file + ' ' + \
                    group_file + ' ' + \
                    addr % client_ips[i] +'\''])
    print('Running %s' % cmd)
    p = subprocess.Popen(cmd, stdout=None, stderr=None, stdin=subprocess.PIPE, shell=True)
    cprocesses.append(p)
time.sleep(1.0)

# start the coordinator
# It's failing because the security group policy doesn't allow port the tcp to port 8000 for the rpc from the coordinator
# The coordinator needs to be run on an aws machine
# or you have to enable the permission (securely?)
cmd = " ".join(['%s/bin/coordinator' % gopath,
                "--numusers %d" % args.users,
                "--groups groups.json",
                "--serverfile servers.json",
                "--clients clients.json",
                "--layers %d" % args.layers,
                "--notes network:%d" % args.network ,
                "--runtype 2"])
print('Running %s' % cmd)
try:
    subprocess.run(cmd, capture_output=True, stdin=subprocess.PIPE, shell=True, check=True)
except subprocess.CalledProcessError as e:
    print("ERROR:")
    print(e.stderr)
    print(e.stdout)
print("cleanup processes")
for p in cprocesses:
    os.kill(p.pid, signal.SIGINT)
for p in sprocesses:
    os.kill(p.pid, signal.SIGINT)
killall(ips)

# os.remove('servers.json')
# os.remove('groups.json')
# os.remove('clients.json')
# os.remove('aws.list')
# os.remove('ip.list')
