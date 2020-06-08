#!/bin/bash
fuser -k 9000/tcp
go run main.go 1>out.txt 2>&1