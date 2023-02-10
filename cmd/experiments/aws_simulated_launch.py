from ipaddress import ip_address
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
ami_id = 'ami-0225fcf40dbb405fe'
product_description = 'Linux/UNIX'
instance_type = 'm5.xlarge'
# I think m5.xlarge will be in all regions
# You'll have to increase the cpu cores and then gomaxproc limit them if you need more memory

# Kind of matches tor node distribution
simulated_regions = ['us-east-2', 'us-west-2', 'eu-north-1', 'eu-central-1']

# Use import key pair to import
key_name = 'slangows'
# Create the same security group in all regions, open to the internet
security_group = 'sg-00322848c00193a51'
subnet_ids = ['subnet-06d332add7a8bc585', 'subnet-0a019cca1a888e57f', 'subnet-0a58c73c0ab29821a', 'subnet-0a73c2cc62888ebce']
try:
    os.remove(ip_list)
except OSError:
    pass

baseIp = "172.16.%d.%d"
vpc_id = 'vpc-0f37f063473264ccc'

session = boto3.Session(profile_name='mit', region_name='us-east-2')
ec2 = session.resource('ec2')
client = session.client('ec2')

f = open(ip_list, 'a')
for idx, _ in enumerate(simulated_regions):
    num_instances = total_num_instances // len(simulated_regions)
    if idx < total_num_instances % len(simulated_regions):
        num_instances += 1
    # Seems like AWS is slow to calculate quota and throws error if you request too fast
    if idx > 0:
        time.sleep(5)
    for i in range(0, num_instances, 128):
        this_instances = min(num_instances-i,128)
        print(this_instances)
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
                'AvailabilityZone': 'us-east-2a',
            },
            SecurityGroupIds=[
                security_group,
            ],
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
            ],
            SubnetId=subnet_ids[idx],
        )

        ips = []
        for inst in response['Instances']:
            ips.append(inst['NetworkInterfaces'][0]['PrivateIpAddress'])

        ips = sorted(ips)
        for ip in ips:
            f.write(ip + '\n')
f.close()
