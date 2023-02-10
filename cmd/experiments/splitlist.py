
ips = []
with open('ip.list') as f:
    lines = f.readlines()
    for line in lines:
        ips.append(line.rstrip('\n'))

left = open('ip.list1', 'x')
right = open('ip.list2', 'x')

for idx, ip in enumerate(ips):
    if idx % 2 == 0:
        left.write(ip + '\n')
    else:
        right.write(ip + '\n')

left.close()
right.close()