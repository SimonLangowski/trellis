import boto3
import json

version = ' 5'
source_region='us-east-2'
ami_id = 'ami-0225fcf40dbb405fe'
dest_regions = ['us-east-2', 'us-west-2', 'eu-north-1', 'eu-central-1']
sessions = [boto3.Session(profile_name='mit', region_name=r) for r in dest_regions]
f = open("amis.json", 'a')
amis = []

for r in dest_regions:
    if r == source_region:
        amis.append(ami_id)
        continue
    session1 = boto3.client('ec2',region_name=r)

    response = session1.copy_image(
    Name='Lightning' + version,
    Description='Lightning ami ' + version + ' copy',
    SourceImageId=ami_id,
    SourceRegion=source_region
    )
    amis.append(response['ImageId'])

print(amis)
json.dump(amis, f)

vpc_format = "172.%d.0.0/16"
vpc_base_start = 16
vpcs = ['vpc-0d7413351ec329ac7', 'vpc-01d9735b8fb3ee771', 'vpc-0b94bdb9615def91b', 'vpc-05fe9632b48381667']
# for idx, r in enumerate(dest_regions):
#     ec2 = sessions[idx].resource('ec2')
#     cidrBlock = vpc_format % vpc_base_start
#     resp = ec2.create_vpc(CidrBlock=cidrBlock)
#     vpcs.append(resp)
#     vpc_base_start += 1

# print(vpcs)

# for idx, r in enumerate(dest_regions):
#     ec2 = sessions[idx].resource('ec2')
#     for idx2, r2 in enumerate(dest_regions):
#         if idx2 > idx:
#             ec2.create_vpc_peering_connection(PeerVpcId=vpcs[idx2],VpcId=vpcs[idx], PeerRegion=r2)

routing_tables = ['rtb-0b15ceca2613f9e5d', 'rtb-0cfdf0f918307d209', 'rtb-02d621fbc0f81c586', 'rtb-05603ce953d640e50']

# for idx, r in enumerate(dest_regions):
#     ec2 = sessions[idx].resource('ec2')
#     vpc = ec2.Vpc(vpcs[idx])
#     # add rules to route table
#     route_table = ec2.RouteTable(routing_tables[idx])
    # for peering in vpc.requested_vpc_peering_connections.all():
    #     peering_id = peering.id
    #     vpc1 = peering.accepter_vpc
    #     vpc2 = peering.requester_vpc
    #     vpc1_idx = vpcs.index(vpc1.id)
    #     vpc2_idx = vpcs.index(vpc2.id)
    #     other_idx = vpc2_idx
    #     if vpc2_idx == idx:
    #         other_idx = vpc1_idx
    #     cidrBlock = vpc_format % (vpc_base_start + other_idx)
    #     try:
    #         route_table.create_route(VpcPeeringConnectionId=peering_id, DestinationCidrBlock=cidrBlock)
    #     except:
    #         print("err")
    # for peering in vpc.accepted_vpc_peering_connections.all():
    #     peering_id = peering.id
    #     vpc2 = peering.accepter_vpc
    #     vpc1 = peering.requester_vpc
    #     vpc1_idx = vpcs.index(vpc1.id)
    #     vpc2_idx = vpcs.index(vpc2.id)
    #     other_idx = vpc2_idx
    #     if vpc2_idx == idx:
    #         other_idx = vpc1_idx
    #     cidrBlock = vpc_format % (vpc_base_start + other_idx)
    #     try:
    #         route_table.create_route(VpcPeeringConnectionId=peering_id, DestinationCidrBlock=cidrBlock)
    #     except:
    #         print("err")

group_name = "LightningGlobalVPCGroup"
security_groups = ['sg-0eb8f187730454a57', 'sg-0c9f4b34500ba6693', 'sg-0925df76652e8e40e', 'sg-0ad530e91ce467194']
# for idx, r in enumerate(dest_regions):
#     ec2 = sessions[idx].resource('ec2')
# #   ec2.import_key_pair(KeyName="slangows", PublicKeyMaterial=pk)
#     security_group = ec2.create_security_group(GroupName=group_name, VpcId=vpcs[idx], Description=group_name)
#     security_groups.append(security_group)
#     security_group.authorize_ingress(
#         CidrIp="13.58.87.34/32",
#         FromPort=22,
#         ToPort=10000,
#         IpProtocol="tcp",
#     )
#     security_group.authorize_ingress(
#         CidrIp="172.16.0.0/12",
#         FromPort=22,
#         ToPort=10000,
#         IpProtocol="tcp",
#     )
#     security_group.authorize_ingress(
#         CidrIp="18.0.0.0/11",
#         FromPort=22,
#         ToPort=10000,
#         IpProtocol="tcp",
#     )
#     security_group.authorize_ingress(
#         CidrIp="128.30.0.0/15",
#         FromPort=22,
#         ToPort=10000,
#         IpProtocol="tcp",
#     )
# print(security_groups)

subnets = ['subnet-04e3a2c498f6b77a1', 'subnet-0a2c029b8f368768e', 'subnet-0b72a35e58a9457e1', 'subnet-07c4684fd723ff66b']

# for idx, r in enumerate(dest_regions):
#     ec2 = sessions[idx].resource('ec2')
#     vpc = ec2.Vpc(vpcs[idx])
#     cidrBlock = vpc_format % (vpc_base_start + idx)
#     s = vpc.create_subnet(AvailabilityZone=r+"a",CidrBlock=cidrBlock)
#     subnets.append(s)
# print(subnets)