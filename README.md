# Trellis: Fast and Scalable Metadata Private Anonymous Broadcast

**Paper:** https://eprint.iacr.org/2022/1548.pdf (NDSS 2023)
The following instructions are for an AWS amazon linux 2 AMI

### Dependencies
Install go 1.17: https://golang.org/doc/install
```
wget https://golang.org/dl/go1.17.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.17.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```
We also need cmake > 3.8
```
wget https://github.com/Kitware/CMake/releases/download/v3.22.0-rc2/cmake-3.22.0-rc2-linux-x86_64.tar.gz
sudo tar -C /usr/local -xzf cmake-3.22.0-rc2-linux-x86_64.tar.gz
export PATH=$PATH:/usr/local/cmake-3.22.0-rc2-linux-x86_64/bin
```
Clone the repo
```
sudo yum install git
git clone git@github.com:SimonLangowski/trellis
cd trellis
```
Install and build mcl
```
sudo yum install gcc10 gcc10-c++ cmake gmp-devel openssl-devel
cd crypto/pairing/mcl/scripts
export CC=gcc10-gcc
export CXX=gcc10-c++
./install_deps.sh
export LD_LIBRARY_PATH=/usr/local/lib
```
Build go files
```
cd ../../../cmd/server
go install && go build
cd ../client
go install && go build
cd ../coordinator
go install && go build
```

### Running locally
Basic test
```
./coordinator --numusers 100 --numservers 10 --numlayers 10 --groupsize 3 --numgroups 3 --runtype 0
./coordinator --numusers 100 --numservers 10 --numlayers 10 --groupsize 3 --numgroups 3 --runtype 1
```
### Parameters
| argument | meaning |
| ---- | ----- |
| f | fraction of servers controlled by the adversary |
| numservers | total number of servers |
| numusers | number of messages |
| numlayers | (optional) number of layers |
| groupsize | (optional) size of anytrust group |
| numgroups | (optional) number of anytrust groups |
| runtype | 0: create keys, 1: run local, 2: run on servers |

Additional arguments will be computed based on the provided values, but you can provide an override for them, for example, to use a simulated number of bins.

### Helper files
Helper files (may need modification for your aws account)
| file | purpose |
| ---- | ----- |
| aws_global_setup.py | setup private vpn network |
| aws_launch.py | launch test in one aws region |
| aws_global_launch.py | launch test in multiple aws regions |
| aws_bandwidth.py | limit the bandwidth of each machine |
| aws_latency.py | add (artificial) network delay to each machine |
| aws_terminate.py | kill all the machines with the specified key |


### Other programs 
Run key exchange in ```server/keyExchange```
``` go test exchangeKey_test.go ```
Calculate the number of bins empirically (for 1/256 probability of failure)
In ```cmd/simulation```
| argument | meaning |
| ---- | ----- |
| numservers | total number of servers |
| numusers | number of messages |
| numlayers | number of layers |
| numtrials | number of trials |
Remember to then add additional layers to account for failure probability.
