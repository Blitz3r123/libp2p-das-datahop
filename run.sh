#!/bin/bash

experiment_name=$1
builder_count=$2
validator_count=$3
non_validator_count=$4
login=$5
builder_ip=$6
exp_duration=$7
ip=$8
echo "Experiment name: $experiment_name"
echo "Builder count: $builder_count"
echo "Validator count: $validator_count"
echo "Non validator count: $non_validator_count"
echo "Login: $login"
echo "Builder IP: $builder_ip"
echo "Parcel size: $parcel_size"
echo "Experiment duration: $exp_duration"

result_dir="/home/${login}/results"
finish_time=$(date +%d-%m-%y-%H-%M)
result_dir="/home/mapigaglio/log"

rm -rf oar*
rm -rf OAR*

if [ ! -d "$result_dir" ]; then
    mkdir -p "$result_dir"
fi

#Install go
echo "Installing go"
cd /tmp
wget "https://go.dev/dl/go1.21.6.linux-amd64.tar.gz"
tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

echo "Installing libp2p-das-datahop"
#Build and run experiment
cp -r /home/mapigaglio/libp2p-das-datahop .
cd libp2p-das-datahop

if [ ! -f "go.mod" ]; then
    echo "go.mod file does not exist - creating one"
    go get -u
    go mod tidy
    go build .
fi

apt install sysstat -y
systemctl start sysstat
sar -A -o sar_logs 1 $exp_duration >/dev/null 2>&1 &
sleep 1

echo "Number of files: $(ls -l | grep -v ^d | wc -l)"

# Run builders
for ((i=0; i<$builder_count-1; i++))
do
    echo "[BACKGROUND] Running builder $i"
    go run . -seed 1234 -port 61960 -nodeType builder -parcelSize 512 -duration $exp_duration -ip $ip -log $result_dir/ >> /home/$login/log/${ip}_builder_$i.txt 2>&1 &
    sleep 1
done
if [ $(($builder_count)) -ne 0 ]; then
    if [ $(($non_validator_count)) -eq 0 ] && [ $(($validator_count)) -eq 0 ]; then
        echo "[FOREGROUND] Running builder [0]"

        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize 512 -duration $exp_duration -ip $ip -log $result_dir/  >> /home/$login/log/${ip}_builder_$i.txt 2>&1
        sleep 1
    else
        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize 512 -duration $exp_duration -ip $ip -log $result_dir/  >> /home/$login/log/${ip}_builder_$i.txt 2>&1 &
        sleep 1
    fi;
fi;

# Run validators
for ((i=0; i<$validator_count - 1; i++))
do
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize 512 -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $result_dir/  >> /home/$login/log/${ip}_validator_$i.txt 2>&1 &
done

if [ $(($non_validator_count)) -eq 0 ]
then
    if [ $(($validator_count)) -ne 0 ]; then
        echo "[FOREGROUND] Running validator $i"
        go run . -nodeType validator -parcelSize 512 -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $result_dir/   >> /home/$login/log/${ip}_validator_$i.txt 2>&1
        sleep 1
    fi;
else
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize 512 -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $result_dir/   >> /home/$login/log/${ip}_validator_$i.txt 2>&1 &
fi

# Run non validators
for ((i=0; i<$non_validator_count - 1; i++))
do
    echo "[BACKGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize 512 -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $result_dir/   >> /home/$login/log/${ip}_nonvalidator_$i.txt 2>&1 &
done

if [ $(($non_validator_count)) -ne 0 ]; then
    echo "[FOREGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize 512 -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $result_dir/   >> /home/$login/log/${ip}_nonvalidator_$i.txt 2>&1
    sleep 1
fi;

echo "End of go commands"
sleep 1

echo "Number of files: $(ls -l | grep -v ^d | wc -l)"
echo "Number of csv files: $(ls -l *.csv | grep -v ^d | wc -l)"

cp *.csv "$result_dir"

if [ -f "sar_logs" ]; then
    echo "Copying sar_logs file"
    cp sar_logs "$result_dir"
else
    echo "sar_logs file does not exist"
fi

sleep 1
