#!/bin/bash

echo "====================================="
echo " Redis Benchmark: Real vs Go Clone"
echo "====================================="

echo ""
echo "Note: This script requires 'redis-benchmark' to be installed."
echo ""

echo ">>> Benchmarking Real Redis (Port 6379) <<<"
redis-benchmark -p 6379 -t set,get -n 100000 -q

echo ""
echo ">>> Benchmarking Go Redis Clone (Port 6380) <<<"
redis-benchmark -p 6380 -t set,get -n 100000 -q
