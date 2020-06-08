############ RUN PRIORITY BASED CONSENSUS IN GO #################

1. Install golang and export GOPATH and GOBIN.
2. Navigate to project folder and run ./run_go.sh to start the server. The output will be overwritten to out.txt
3. Open another terminal and run ./run_CB.sh to simulate 8 miners.

-- Each miner's output is shown in temp_out<miner_number>.txt
-- Number of miners can be changed in ./run_CB.sh in the for loop
-- Number of blocks to be mined and sleep time per block can be changed in main.go global variables.
