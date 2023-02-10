import sys
import hashlib
import datetime
import boto3
import time

real_run = eval(sys.argv[1])
#terminate ALL instances

ec2 = boto3.client('ec2')

response = ec2.describe_instances(
        DryRun=not real_run,
)

def tagged(instance):
    tags = instance['Tags']
    for t in tags:
        if t['Key'] == 'Name':
            return True
    return False


instance_ids = []

for resp in response['Reservations']:
    for inst in resp['Instances']:
        if inst['State']['Name'] == 'running' and inst['KeyName'] == 'slangows':
            if not tagged(inst):
                instance_ids.append(inst['InstanceId'])


def chunks(l, n):
    """Yield successive n-sized chunks from l."""
    for i in range(0, len(l), n):
        yield l[i:i + n]

for ids in chunks(instance_ids, 100):
    response = ec2.stop_instances(
        InstanceIds=ids,
        DryRun=not real_run
    )
