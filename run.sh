#!/bin/bash

# Script parameters: 
# $1 = duration
# $2 = experiment_name
# $3 = builder count
# $4 = validator count
# $5 = non validator count

duration=$1
experiment_name=$2
builder_count=$3
validator_count=$4
non_validator_count=$5

result_dir="/home/kpeeroo/result"

#Install go
echo "Installing go"
cd /tmp
wget "https://go.dev/dl/go1.20.4.linux-amd64.tar.gz"
sudo-g5k tar -C /usr/local -xzf go1.20.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

echo "Installing libp2p-das-datahop"
#Build and run experiment
git clone https://github.com/Blitz3r123/libp2p-das-datahop.git
cd libp2p-das-datahop

echo "Running builders"
# Run builders
for ((i=1; i<=$builder_count; i++))
    do
        echo "Running builder $i"
        go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $duration &
        sleep 0.5
    done

echo "Running validators"
# Run validators
for ((i=1; i<=$validator_count; i++))
    do
        echo "Running validator $i"
        go run . -debug -duration $duration -nodeType validator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
        sleep 0.5
    done

echo "Running non-validators"
# Run non validators
for ((i=1; i<=$non_validator_count; i++))
    do
        echo "Running non validator $i"
        go run . -debug -duration $duration -nodeType nonvalidator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
        sleep 0.5
    done

if [!-d $result_dir]; then
    mkdir -p $result_dir;
fi;
mkdir $result_dir/$experiment_name_$(date +%d-%m-%y-%H-%M)
cp *.csv $result_dir/$experiment_name_$(date +%d-%m-%y-%H-%M)