import sys
import os
import hashlib
import datetime
import boto3
import time

total_num_instances = int(sys.argv[1])
real_run = eval(sys.argv[2])
ip_list = sys.argv[3]


# system parameters that won't change for these experiments

product_description = 'Linux/UNIX'
instance_type = 'm5.xlarge'
# I think m5.xlarge will be in all regions
# You'll have to increase the cpu cores and then gomaxproc limit them if you need more memory

# Kind of matches tor node distribution
regions = ['us-east-2', 'us-west-2', 'eu-north-1', 'eu-central-1']
# copy the ami to each region, and put the ami-id here
ami_ids = ['ami-0225fcf40dbb405fe', 'ami-0be0185e0103b29e5', 'ami-0065de1049da79e55', 'ami-00a5422d60fd9036b']
security_groups = ['sg-0eb8f187730454a57', 'sg-0c9f4b34500ba6693', 'sg-0925df76652e8e40e', 'sg-0ad530e91ce467194']
subnet_ids = ['subnet-04e3a2c498f6b77a1', 'subnet-0a2c029b8f368768e', 'subnet-0b72a35e58a9457e1', 'subnet-07c4684fd723ff66b']
# Use import key pair to import
key_name = 'slangows'
# Create the same security group in all regions, open to the internet
security_group = 'LightningGlobalVPCGroup'

try:
    os.remove(ip_list)
except OSError:
    pass

sessions = [boto3.Session(profile_name='mit', region_name=r) for r in regions]

f = open(ip_list, 'a')
for idx, region in enumerate(regions):
    # launch many spot blocks
    ec2 = sessions[idx].resource('ec2', region_name=region)
    client = sessions[idx].client('ec2', region_name=region)
    num_instances = total_num_instances // len(regions)
    if idx < total_num_instances % len(regions):
        num_instances += 1
    for i in range(0, num_instances, 128):
        this_instances = min(num_instances-i,128)
        print(region, this_instances)
        h = hashlib.sha256()
        h.update(str(datetime.datetime.now()).encode('utf-8'))
        client_token = h.hexdigest()
        response = client.run_instances(
            ImageId=ami_ids[idx],
            InstanceType=instance_type,
            KeyName='slangows',
            MaxCount=this_instances,
            MinCount=this_instances,
            Monitoring={
            'Enabled': False
            },
            Placement={
                'AvailabilityZone': region + 'a',
            },
            SecurityGroupIds=[
                security_groups[idx],
            ],
            SubnetId=subnet_ids[idx],
            ClientToken=client_token,
            DryRun=not real_run,
            InstanceInitiatedShutdownBehavior='stop',
            TagSpecifications=[
                {
                    'ResourceType': 'instance',
                    'Tags': [
                        {
                            'Key': 'project',
                            'Value': 'Lightning'
                        },
                        {
                            'Key': 'owner',
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
