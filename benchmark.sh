#!/bin/bash

# This script benchmarks all execution modes for viztruct

# Path to test repository (change this to your test repo)
TEST_REPO="$HOME/projects/kubernetes"

# Number of runs for each mode
RUNS=1

# Output files
SEQUENTIAL_OUTPUT="sequential_results.txt"
CONCURRENT_OUTPUT="concurrent_results.txt"
DEPENDENCY_OUTPUT="dependency_results.txt"
BENCHMARK_RESULTS="benchmark_results.txt"

# Make sure viztruct is built
go build -o viztruct ./cmd/viztruct

echo "Starting viztruct benchmark..."
echo "=================================" > $BENCHMARK_RESULTS
echo "VIZTRUCT PERFORMANCE BENCHMARK" >> $BENCHMARK_RESULTS
echo "=================================" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS
echo "Test repository: $TEST_REPO" >> $BENCHMARK_RESULTS
echo "Runs per mode: $RUNS" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS

# Clear Go's package cache before each run
clear_cache() {
    echo "cleaning cache"
    go clean -cache >/dev/null 2>&1
}

# Benchmark sequential mode
echo "Running sequential benchmark..."
echo "Sequential mode times:" >> $BENCHMARK_RESULTS

seq_times=()
for i in $(seq 1 $RUNS); do
    echo "Sequential run $i/$RUNS..."
    clear_cache
    # Measure time for sequential mode
    start=$(date +%s.%N)
    ./viztruct --file="$TEST_REPO" --format=txt --timeout=300 --mode=sequential > $SEQUENTIAL_OUTPUT
    end=$(date +%s.%N)
    
    runtime=$(echo "$end - $start" | bc)
    seq_times+=($runtime)
    echo "Run $i: $runtime seconds" >> $BENCHMARK_RESULTS
done

# Calculate average sequential time
seq_sum=0
for t in "${seq_times[@]}"; do
    seq_sum=$(echo "$seq_sum + $t" | bc)
done
seq_avg=$(echo "scale=3; $seq_sum / $RUNS" | bc)
echo "Average sequential time: $seq_avg seconds" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS

# Benchmark concurrent mode
echo "Running concurrent benchmark..."
echo "Concurrent mode times:" >> $BENCHMARK_RESULTS

conc_times=()
for i in $(seq 1 $RUNS); do
    echo "Concurrent run $i/$RUNS..."
    clear_cache
    # Measure time for concurrent mode
    start=$(date +%s.%N)
    ./viztruct --file="$TEST_REPO" --format=txt --timeout=300 --mode=concurrent > $CONCURRENT_OUTPUT
    end=$(date +%s.%N)
    
    runtime=$(echo "$end - $start" | bc)
    conc_times+=($runtime)
    echo "Run $i: $runtime seconds" >> $BENCHMARK_RESULTS
done

# Calculate average concurrent time
conc_sum=0
for t in "${conc_times[@]}"; do
    conc_sum=$(echo "$conc_sum + $t" | bc)
done
conc_avg=$(echo "scale=3; $conc_sum / $RUNS" | bc)
echo "Average concurrent time: $conc_avg seconds" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS

# Benchmark dependency-aware mode
echo "Running dependency-aware benchmark..."
echo "Dependency-aware mode times:" >> $BENCHMARK_RESULTS

dep_times=()
for i in $(seq 1 $RUNS); do
    echo "Dependency-aware run $i/$RUNS..."
    clear_cache
    # Measure time for dependency-aware mode
    start=$(date +%s.%N)
    ./viztruct --file="$TEST_REPO" --format=txt --timeout=300 --mode=dependency > $DEPENDENCY_OUTPUT
    end=$(date +%s.%N)
    
    runtime=$(echo "$end - $start" | bc)
    dep_times+=($runtime)
    echo "Run $i: $runtime seconds" >> $BENCHMARK_RESULTS
done

# Calculate average dependency-aware time
dep_sum=0
for t in "${dep_times[@]}"; do
    dep_sum=$(echo "$dep_sum + $t" | bc)
done
dep_avg=$(echo "scale=3; $dep_sum / $RUNS" | bc)
echo "Average dependency-aware time: $dep_avg seconds" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS

# Calculate speedups
seq_conc_speedup=$(echo "scale=2; $seq_avg / $conc_avg" | bc)
seq_dep_speedup=$(echo "scale=2; $seq_avg / $dep_avg" | bc)
conc_dep_speedup=$(echo "scale=2; $conc_avg / $dep_avg" | bc)

echo "=================================" >> $BENCHMARK_RESULTS
echo "RESULTS SUMMARY" >> $BENCHMARK_RESULTS
echo "=================================" >> $BENCHMARK_RESULTS
echo "Average sequential time: $seq_avg seconds" >> $BENCHMARK_RESULTS
echo "Average concurrent time: $conc_avg seconds" >> $BENCHMARK_RESULTS
echo "Average dependency-aware time: $dep_avg seconds" >> $BENCHMARK_RESULTS
echo "" >> $BENCHMARK_RESULTS
echo "Sequential vs Concurrent speedup: ${seq_conc_speedup}x" >> $BENCHMARK_RESULTS
echo "Sequential vs Dependency speedup: ${seq_dep_speedup}x" >> $BENCHMARK_RESULTS
echo "Concurrent vs Dependency speedup: ${conc_dep_speedup}x" >> $BENCHMARK_RESULTS

echo "Benchmark complete! Results saved to $BENCHMARK_RESULTS"
cat $BENCHMARK_RESULTS 