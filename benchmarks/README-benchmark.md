# Viztruct Performance Benchmark

This directory contains tools to benchmark the performance of Viztruct when analyzing different codebases.

## Running the Benchmark

To run the benchmark, follow these steps:

1. Build the Docker image:
   ```
   docker build -t viztruct-benchmark .
   ```

2. Run the benchmark:
   ```
   docker run --rm viztruct-benchmark
   ```

The benchmark will:
- Analyze several Go repositories of varying sizes
- Measure both cold and warm run performance
- Compare JSON and SVG output formats
- Clear the Go cache between cold runs to simulate a fresh environment

## Interpreting Results

The benchmark will output timing information for each repository and scenario. The key metrics to look for:

- **Cold Run Time**: The time taken when running with a clean Go cache, simulating the first run on a system
- **Warm Run Time**: The time taken for subsequent runs, which benefits from Go's build cache
- **SVG vs JSON Performance**: Comparison of rendering times for different output formats

## Adding More Test Repositories

To benchmark with additional repositories, edit the `benchmark.sh` file and add your repositories to the `REPOS` array.

## Optimizing Performance

Based on the benchmark results, you may want to consider:

1. Implementing a caching mechanism in Viztruct to improve cold start times
2. Optimizing SVG generation for large structs
3. Adding parallel processing for analyzing multiple packages simultaneously
4. Pre-compiling templates or other expensive operations

## Expected Results

Typically, you should expect to see:

1. The first (cold) run taking significantly longer due to dependency fetching and module verification
2. Subsequent (warm) runs being much faster
3. Larger codebases taking proportionally longer to analyze
4. SVG generation potentially being more expensive than JSON generation for large structs 