#!/bin/bash

# set repositories to analyze with their GitHub URLs
declare -A REPOS=(
  #["github.com/drone/drone-go"]="https://github.com/drone/drone-go.git" # mid size
  ["github.com/drone/autoscaler"]="https://github.com/drone/autoscaler.git" # small size
  #["github.com/kubernetes/kubernetes"]="https://github.com/kubernetes/kubernetes.git" # large size
)

mkdir -p /benchmark/repos

# initialize a Go module for the benchmark
cd /benchmark
go mod init benchmark

# Function to run benchmark
run_benchmark() {
  local repo_path=$1
  local format=$2
  local description=$3
  local module_name=$4

  echo "===== $description: $module_name ====="
  
  # clear Go cache to simulate cold run
  if [[ "$description" == *"Cold Run"* ]]; then
    go clean -cache -modcache
    echo "Cleared Go cache"
  fi
  
  # time the execution
  START_TIME=$(date +%s.%N)
  
  if [ "$format" == "json" ]; then
    viztruct --file "$repo_path" --format=json > /dev/null 2>&1
  else
    viztruct --file "$repo_path" > /dev/null 2>&1
  fi
  
  END_TIME=$(date +%s.%N)
  ELAPSED=$(echo "$END_TIME - $START_TIME" | bc)
  
  echo "Time taken: $ELAPSED seconds"
  echo ""
}

echo "======================================="
echo "Viztruct Performance Benchmark"
echo "======================================="
echo "Date: $(date)"
echo "Go version: $(go version)"
echo "======================================="
echo ""

# clone and benchmark each repository
for module_name in "${!REPOS[@]}"; do
  repo_url=${REPOS[$module_name]}
  repo_dir="/benchmark/repos/$(basename $module_name)"
  
  echo "Cloning $module_name..."
  
  # clone the repository if it doesn't exist
  if [ ! -d "$repo_dir" ]; then
    git clone --depth 1 "$repo_url" "$repo_dir"
  fi
  
  # handle the kubernetes repo specially since we only need the controller package
  if [[ "$module_name" == *"kubernetes"* ]]; then
    # run benchmarks on just the controller package
    controller_path="$repo_dir/pkg/controller"
    if [ -d "$controller_path" ]; then
      # JSON output format
      run_benchmark "$controller_path" "json" "Cold Run (JSON)" "$module_name/pkg/controller"
      run_benchmark "$controller_path" "json" "Second Run (JSON)" "$module_name/pkg/controller"
      
      # SVG output format
      run_benchmark "$controller_path" "svg" "Cold Run (SVG)" "$module_name/pkg/controller"
      run_benchmark "$controller_path" "svg" "Second Run (SVG)" "$module_name/pkg/controller"
    else
      echo "Controller package not found at $controller_path"
    fi
  else
    # JSON output format
    run_benchmark "$repo_dir" "json" "Cold Run (JSON)" "$module_name"
    run_benchmark "$repo_dir" "json" "Second Run (JSON)" "$module_name"
    
    # SVG output format
    run_benchmark "$repo_dir" "svg" "Cold Run (SVG)" "$module_name"
    run_benchmark "$repo_dir" "svg" "Second Run (SVG)" "$module_name"
  fi
  
  echo "---------------------------------------"
done

echo "Benchmark completed!" 
