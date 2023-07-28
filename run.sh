#!/bin/bash

# Script parameters: 
# $1=duration
# $2=experiment_name
# $3=builder count
# $4=validator count
# $5=non validator count

exp_duration=$1
experiment_name=$2
builder_count=$3
validator_count=$4
non_validator_count=$5
login=$6

result_dir="/home/${login}/results"
finish_time=$(date +%d-%m-%y-%H-%M)
result_dir="${result_dir}/${experiment_name}_${finish_time}"

rm -rf oar*
rm -rf OAR*

if [ ! -d "$result_dir" ]; then
    mkdir -p "$result_dir"
fi

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

sudo-g5k systemctl start sysstat
sar -A -o sar_logs 1 $exp_duration >/dev/null 2>&1 &
sleep 1

# Run builders
for ((i=0; i<$builder_count-1; i++))
do
    echo "[BACKGROUND] Running builder $i"
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $exp_duration &
    sleep 0.01
done

if [ $(($builder_count)) -ne 0 ] && [ $(($non_validator_count)) -eq 0 ] && [ $(($validator_count)) -eq 0 ]; then
    echo "[FOREGROUND] Running builder [0]"
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $exp_duration
    sleep 0.01
fi;

# Run validators
for ((i=0; i<$validator_count - 1; i++))
do
    echo "[BACKGROUND] Running validator $i"
    go run . -debug -duration $exp_duration -nodeType validator -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    sleep 0.01
done

if [ $(($non_validator_count)) -eq 0 ]
then
    if [ $(($validator_count)) -ne 0 ]; then
        echo "[FOREGROUND] Running validator $i"
        go run . -debug -duration $exp_duration -nodeType validator -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
    fi;
else
    echo "[BACKGROUND] Running validator $i"
    go run . -debug -duration $exp_duration -nodeType validator -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
fi

# Run non validators
for ((i=0; i<$non_validator_count - 1; i++))
do
    echo "[BACKGROUND] Running non validator $i"
    go run . -debug -duration $exp_duration -nodeType nonvalidator -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    sleep 0.01
done

if [ $(($non_validator_count)) -ne 0 ]; then
    echo "[FOREGROUND] Running non validator $i"
    go run . -debug -duration $exp_duration -nodeType nonvalidator -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
fi;

cp *.csv "$result_dir"
cp sar_logs "$result_dir"
sleep 1