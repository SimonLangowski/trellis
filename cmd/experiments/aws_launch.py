import sys
import os
import hashlib
import datetime
import boto3
import time

num_instances = int(sys.argv[1])
real_run = eval(sys.argv[2])
ip_list = sys.argv[3]

# launch many spot blocks
ec2 = boto3.resource('ec2')
client = boto3.client('ec2')


# system parameters that won't change for these experiments
ami_id = 'ami-0862c4e5a27a40f3b'
product_description = 'Linux/UNIX'
instance_type = 'm5.xlarge'
# Run 10^7 messages and f > .2 on m5.xlarge, 10^7 messages, n=128, f=.3 on r6i.xlarge
# I think c5.xlarge and m5.xlarge will be in all regions, but definitely not r6i.large
# You'll have to increase the cpu cores and then gomaxproc limit them if you need more memory

region = 'us-east-2'
security_group = 'launch-wizard-8'
key_name = 'slangows'

try:
    os.remove(ip_list)
except OSError:
    pass

f = open(ip_list, 'a')
for i in range(0, num_instances, 128):
    this_instances = min(num_instances-i,128)
    print(this_instances)
    h = hashlib.sha256()
    h.update(str(datetime.datetime.now()).encode('utf-8'))
    client_token = h.hexdigest()
    response = client.run_instances(
        ImageId=ami_id,
        InstanceType=instance_type,
        KeyName='slangows',
        MaxCount=this_instances,
        MinCount=this_instances,
        Monitoring={
        'Enabled': False
        },
        Placement={
            'AvailabilityZone': region + 'c',
        },
        SecurityGroupIds=[
            security_group,
        ],
        ClientToken=client_token,
        DryRun=not real_run,
        InstanceInitiatedShutdownBehavior='stop',
        TagSpecifications=[
            {
                'ResourceType': 'instance',
                'Tags': [
                    {
                        'Key': 'Project',
                        'Value': 'Lightning'
                    },
                    {
                        'Key': 'User',
                        'Value': 'slangows',
                    }
                ]
            },
        ]
    )

    ips = []
    for inst in response['Instances']:
        ips.append(inst['NetworkInterfaces'][0]['PrivateIpAddress'])

    ips = sorted(ips)
    for ip in ips:
        f.write(ip + '\n')
f.close()
