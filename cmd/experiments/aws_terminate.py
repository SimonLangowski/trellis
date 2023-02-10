import sys
import hashlib
import datetime
import boto3
import time

real_run = eval(sys.argv[1])
#terminate ALL instances

def tagged(instance):
    tags = instance['Tags']
    for t in tags:
        if t['Key'] == 'Name':
            return True
    return False

def chunks(l, n):
    """Yield successive n-sized chunks from l."""
    for i in range(0, len(l), n):
        yield l[i:i + n]

regions = ['us-east-2', 'us-west-2', 'eu-north-1', 'eu-central-1']

for r in regions:
    session = boto3.Session(profile_name='mit')
    ec2 = session.client('ec2', region_name=r)

    response = ec2.describe_instances(
            DryRun=not real_run,
    )

    instance_ids = []

    for resp in response['Reservations']:
        for inst in resp['Instances']:
            if inst['State']['Name'] == 'running' and inst['KeyName'] == 'slangows':
                if not tagged(inst):
                    instance_ids.append(inst['InstanceId'])

    for ids in chunks(instance_ids, 100):
        response = ec2.terminate_instances(
            InstanceIds=ids,
            DryRun=not real_run
        )
