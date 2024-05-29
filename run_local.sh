#!/bin/bash

experiment_name=$1
builder_count=$2
validator_count=$3
non_validator_count=$4
builder_ip=127.0.0.1
parcel_size=$5
exp_duration=$6
echo "Experiment name: $experiment_name"
echo "Builder count: $builder_count"
echo "Validator count: $validator_count"
echo "Non validator count: $non_validator_count"
echo "Builder IP: $builder_ip"
echo "Parcel size: $parcel_size"
echo "Experiment duration: $exp_duration"
ip=127.0.0.1
result_dir="./results"
finish_time=$(date +%d-%m-%y-%H-%M)
run_dir=${experiment_name}
log_dir="${result_dir}/${experiment_name}"
if [ ! -d "$result_dir" ]; then
    mkdir -p "$result_dir"
fi

if [ ! -d "$log_dir" ]; then
    mkdir -p "$log_dir"
fi

port_counter=10200
# Run builders
for ((i=0; i<$builder_count-1; i++))
do
    echo "[BACKGROUND] Running builder $i"
    go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size -duration $exp_duration -ip $ip -log $log_dir/ >> $log_dir/${ip}_builder_$i.txt 2>&1 &
    ((port_counter++))
    sleep 1
done
if [ $(($builder_count)) -ne 0 ]; then
    if [ $(($non_validator_count)) -eq 0 ] && [ $(($validator_count)) -eq 0 ]; then
        echo "[FOREGROUND] Running builder [0]"

        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size -duration $exp_duration -ip $ip -log $log_dir/  >> $log_dir/${ip}_builder_$i.txt 2>&1
        sleep 1
        ((port_counter++))
    else
        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size -duration $exp_duration -ip $ip -log $log_dir/  >> $log_dir/${ip}_builder_$i.txt 2>&1 &
        sleep 1
        ((port_counter++))
    fi;
fi;

# Run validators
for ((i=0; i<$validator_count - 1; i++))
do
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize $parcel_size -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $log_dir/  >> $log_dir/${ip}_validator_$i.txt 2>&1 &
done

if [ $(($non_validator_count)) -eq 0 ]
then
    if [ $(($validator_count)) -ne 0 ]; then
        echo "[FOREGROUND] Running validator $i"
        go run . -nodeType validator -parcelSize $parcel_size -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $log_dir/   >> $log_dir/${ip}_validator_$i.txt 2>&1
        sleep 1
    fi;
else
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize $parcel_size -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $log_dir/   >> $log_dir/${ip}_validator_$i.txt 2>&1 &
fi

# Run non validators
for ((i=0; i<$non_validator_count - 1; i++))
do
    echo "[BACKGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize $parcel_size -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $log_dir/   >> $log_dir/${ip}_nonvalidator_$i.txt 2>&1 &
done

if [ $(($non_validator_count)) -ne 0 ]; then
    echo "[FOREGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize $parcel_size -duration $exp_duration -ip $ip -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73  -log $log_dir/   >> $log_dir/${ip}_nonvalidator_$i.txt 2>&1
    sleep 1
fi;

echo "End of go commands"
sleep 1

cp *.csv "$log_dir"

sleep 1
