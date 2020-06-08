#!/bin/bash
for i in $(seq 1 8); do
OUT_FILE="temp_out_miner$i.txt"
cat miner_input.txt | nc localhost 9000 1> $OUT_FILE & 
done