Reducing analyse time of large codebases


- what was tried
  - sequencial approach
  - concurrent approach
- which performed better? why?
   - concurrent approach performed worse
   - this indicates that the bottleneck is not CPU related
   - the bottleneck is likely the I/O operations since this process is I/O bound.



options:
- dependency-aware scheduler that builds a graph of package dependencies first, then processes them in topological order with parallelism only among truly independent packages.
- DFS to analyse files 



## Result

```txt
./benchmark.sh
Starting viztruct benchmark...
Running sequential benchmark...
Sequential run 1/1...
cleaning cache
Running concurrent benchmark...
Concurrent run 1/1...
cleaning cache
Running dependency-aware benchmark...
Dependency-aware run 1/1...
cleaning cache
Benchmark complete! Results saved to benchmark_results.txt
=================================
VIZTRUCT PERFORMANCE BENCHMARK
=================================

Test repository: /Users/buarki/projects/kubernetes
Runs per mode: 1

Sequential mode times:
Run 1: 84.782725000 seconds
Average sequential time: 84.782 seconds

Concurrent mode times:
Run 1: 83.864526000 seconds
Average concurrent time: 83.864 seconds

Dependency-aware mode times:
Run 1: 89.356829000 seconds
Average dependency-aware time: 89.356 seconds

=================================
RESULTS SUMMARY
=================================
Average sequential time: 84.782 seconds
Average concurrent time: 83.864 seconds
Average dependency-aware time: 89.356 seconds

Sequential vs Concurrent speedup: 1.01x
Sequential vs Dependency speedup: .94x
Concurrent vs Dependency speedup: .93x
```

The benchmark results are quite revealing. Contrary to our theoretical expectations, the dependency-aware mode is actually slower than both sequential and concurrent modes. Let's analyze why:
Dependency tracking overhead: Building and maintaining the dependency graph requires extra processing and memory. This overhead (about 6% compared to concurrent mode) outweighs any benefits from smarter scheduling.
Synchronization costs: The dependency-aware approach requires more complex synchronization with frequent checking of package states, adding overhead.
Sleep delays: Our implementation includes a 10ms sleep between scheduling rounds, which accumulates into noticeable delay over the course of execution.
Already-optimized Go package loader: Go's package loading system already handles dependencies efficiently, making our additional dependency management redundant or even counterproductive.
Limited true parallelism: Many packages in Go codebases have complex interdependencies that limit opportunities for parallel processing, regardless of our scheduling approach.
For this specific workload, the simple concurrent approach performs best, likely because:
It avoids the overhead of dependency tracking
It allows maximum parallelism with minimal synchronization
The Go package loader already handles most dependency issues internally
This is a great example of how performance optimizations can sometimes be counterproductive - the more complex solution isn't always faster. For your use case, I'd recommend staying with the concurrent mode, as it provides the best performance with the simplest implementation.