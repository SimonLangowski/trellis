import sys
import boto3

ip_list2 = "ip.list"
ip_list = "aws.list"


regions = ['us-east-1', 'us-east-2', 'us-west-2', 'eu-north-1', 'ap-northeast-1', 'eu-west-2', 'ap-southeast-2', 'sa-east-1']

def tagged(instance):
    tags = instance['Tags']
    for t in tags:
        if t['Key'] == 'Name':
            return True
    return False


ips = []
ips2 = []

for r in regions:

    ec2 = boto3.client('ec2', region_name=r)

    response = ec2.describe_instances(
            DryRun=False,
    )

    for resp in response['Reservations']:
        for inst in resp['Instances']:
            if inst['State']['Name'] == 'running' and inst['KeyName'] == 'slangows':
                if not tagged(inst):  
                    ips.append(inst['PrivateIpAddress'])
                    # ips2.append(inst['PublicIpAddress'])

ips = sorted(ips)
ips2 = sorted(ips2)
f = open(ip_list, 'w')
for ip in ips:
    f.write(ip + '\n')
f.close()

f = open(ip_list2, 'w')
for ip in ips2:
    f.write(ip + '\n')
f.close()
