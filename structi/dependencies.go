package structi

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// PackageNode represents a package in the dependency graph
type PackageNode struct {
	Package       *packages.Package
	Dependencies  map[string]bool // Packages this package depends on
	Dependents    map[string]bool // Packages that depend on this package
	Processed     bool            // Whether this package has been processed
	ProcessingErr error           // Error encountered during processing
}

// DependencyGraph represents the dependency graph of packages
type DependencyGraph struct {
	Nodes map[string]*PackageNode
	mu    sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*PackageNode),
	}
}

// AddPackage adds a package to the graph
func (g *DependencyGraph) AddPackage(pkg *packages.Package) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Nodes[pkg.PkgPath]; exists {
		return // Already added
	}

	g.Nodes[pkg.PkgPath] = &PackageNode{
		Package:      pkg,
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		Processed:    false,
	}
}

// AddDependency adds a dependency from one package to another
func (g *DependencyGraph) AddDependency(from, to string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Make sure both packages exist in the graph
	if _, exists := g.Nodes[from]; !exists {
		logf("Warning: Package %s not found in graph when adding dependency\n", from)
		return
	}

	if _, exists := g.Nodes[to]; !exists {
		logf("Warning: Package %s not found in graph when adding dependency\n", to)
		return
	}

	// Add the dependency relationship
	g.Nodes[from].Dependencies[to] = true
	g.Nodes[to].Dependents[from] = true
}

// FindRoots returns packages that don't depend on any other packages in the graph
func (g *DependencyGraph) FindRoots() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var roots []string
	for pkgPath, node := range g.Nodes {
		if len(node.Dependencies) == 0 {
			roots = append(roots, pkgPath)
		}
	}

	return roots
}

// GetPackageNode returns a package node by its path
func (g *DependencyGraph) GetPackageNode(pkgPath string) (*PackageNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.Nodes[pkgPath]
	return node, exists
}

// SetProcessed marks a package as processed
func (g *DependencyGraph) SetProcessed(pkgPath string, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if node, exists := g.Nodes[pkgPath]; exists {
		node.Processed = true
		node.ProcessingErr = err
	}
}

// AllDependenciesProcessed checks if all dependencies of a package have been processed
func (g *DependencyGraph) AllDependenciesProcessed(pkgPath string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.Nodes[pkgPath]
	if !exists {
		return false
	}

	for depPath := range node.Dependencies {
		depNode, exists := g.Nodes[depPath]
		if !exists || !depNode.Processed {
			return false
		}
	}

	return true
}

// GetReadyPackages returns packages that are ready to be processed
// A package is ready if all its dependencies have been processed
func (g *DependencyGraph) GetReadyPackages() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var ready []string
	for pkgPath, node := range g.Nodes {
		if !node.Processed {
			allDepsProcessed := true
			for depPath := range node.Dependencies {
				depNode, exists := g.Nodes[depPath]
				if !exists || !depNode.Processed {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				ready = append(ready, pkgPath)
			}
		}
	}

	return ready
}

// AllProcessed checks if all packages have been processed
func (g *DependencyGraph) AllProcessed() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, node := range g.Nodes {
		if !node.Processed {
			return false
		}
	}

	return true
}

// DetectCycles detects cycles in the dependency graph
// Returns true if a cycle is found
func (g *DependencyGraph) DetectCycles() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var isCyclic func(string) bool
	isCyclic = func(pkgPath string) bool {
		if !visited[pkgPath] {
			visited[pkgPath] = true
			recStack[pkgPath] = true

			node, exists := g.Nodes[pkgPath]
			if exists {
				for depPath := range node.Dependencies {
					if !visited[depPath] && isCyclic(depPath) {
						return true
					} else if recStack[depPath] {
						return true
					}
				}
			}
		}

		recStack[pkgPath] = false
		return false
	}

	for pkgPath := range g.Nodes {
		if !visited[pkgPath] {
			if isCyclic(pkgPath) {
				return true
			}
		}
	}

	return false
}

// GetProcessingErrors returns a map of package paths to errors encountered during processing
func (g *DependencyGraph) GetProcessingErrors() map[string]error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	errors := make(map[string]error)
	for pkgPath, node := range g.Nodes {
		if node.ProcessingErr != nil {
			errors[pkgPath] = node.ProcessingErr
		}
	}

	return errors
}

// BuildDependencyGraph builds a dependency graph from the given packages
func buildDependencyGraph(pkgs []*packages.Package) *DependencyGraph {
	graph := NewDependencyGraph()

	// First, add all packages to the graph
	for _, pkg := range pkgs {
		graph.AddPackage(pkg)
	}

	// Then, add dependencies between packages
	for _, pkg := range pkgs {
		for _, imp := range pkg.Imports {
			// Only add dependencies for packages that are in our set
			if _, exists := graph.GetPackageNode(imp.PkgPath); exists {
				graph.AddDependency(pkg.PkgPath, imp.PkgPath)
			}
		}
	}

	logf("Built dependency graph with %d packages\n", len(graph.Nodes))
	hasCycles := graph.DetectCycles()
	if hasCycles {
		logf("Warning: Dependency graph has cycles. This may affect processing order.\n")
	}

	return graph
}

// ProcessPackagesWithDependencies processes packages respecting their dependencies
func processPackagesWithDependencies(pkgs []*packages.Package, strategyNames []string) ([][]Info, error) {
	// Build the dependency graph
	logf("Building dependency graph...\n")
	graph := buildDependencyGraph(pkgs)

	// Create a map to store results
	var allInfos [][]Info
	var allInfosMutex sync.Mutex

	// Use a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, concurrency)

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Create an error channel to collect errors
	errChan := make(chan error, len(pkgs))

	// Track time for timeout
	startTime := time.Now()
	timeoutDuration := time.Duration(timeout) * time.Second

	logf("Processing packages with dependency-aware scheduler (%d workers)...\n", concurrency)

	// Process packages in dependency order
	for !graph.AllProcessed() {
		// Check if we've exceeded the timeout
		if timeout > 0 && time.Since(startTime) > timeoutDuration {
			return allInfos, fmt.Errorf("processing timed out after %d seconds", timeout)
		}

		// Get packages that are ready to be processed
		readyPkgs := graph.GetReadyPackages()

		if len(readyPkgs) == 0 && !graph.AllProcessed() {
			// If we have no ready packages but not all are processed,
			// we might have a cycle or a dependency on an unprocessed package
			// Let's just process any unprocessed package to break potential deadlocks
			logf("No ready packages, but not all are processed. Possible cycle detected.\n")
			for pkgPath, node := range graph.Nodes {
				if !node.Processed {
					readyPkgs = append(readyPkgs, pkgPath)
					break
				}
			}
		}

		// Process all ready packages in parallel
		for _, pkgPath := range readyPkgs {
			pkgNode, exists := graph.GetPackageNode(pkgPath)
			if !exists || pkgNode.Processed {
				continue
			}

			wg.Add(1)
			go func(pkgPath string, pkg *packages.Package) {
				// Acquire semaphore slot
				semaphore <- struct{}{}
				defer func() {
					// Release semaphore slot
					<-semaphore
					wg.Done()

					// Handle panics in goroutine
					if r := recover(); r != nil {
						errMsg := fmt.Sprintf("Recovered from panic in package processing goroutine (%s): %v",
							pkgPath, r)
						logf("%s\n", errMsg)
						errChan <- fmt.Errorf(errMsg)
					}
				}()

				// Skip packages with errors
				if len(pkg.Errors) > 0 {
					if skipErrors {
						logf("Skipping package %s due to errors\n", pkgPath)
						graph.SetProcessed(pkgPath, fmt.Errorf("package has errors: %v", pkg.Errors[0]))
						return
					} else {
						err := fmt.Errorf("error in package %s: %v", pkgPath, pkg.Errors[0])
						errChan <- err
						graph.SetProcessed(pkgPath, err)
						return
					}
				}

				// Skip packages with no syntax
				if len(pkg.Syntax) == 0 {
					graph.SetProcessed(pkgPath, nil)
					return
				}

				var packageInfos []Info
				var processingErr error

				// Process each file in the package
				for _, syntax := range pkg.Syntax {
					// Handle file processing errors gracefully
					func() {
						defer func() {
							if r := recover(); r != nil {
								errMsg := fmt.Sprintf("Recovered from panic processing file in package %s: %v",
									pkgPath, r)
								logf("%s\n", errMsg)
								processingErr = fmt.Errorf(errMsg)
							}
						}()

						fileInfos, err := collectStructs(syntax, pkg.TypesInfo, strategyNames)
						if err != nil {
							if skipErrors {
								logf("Error processing package %s: %v\n", pkgPath, err)
								processingErr = err
								return
							} else {
								errChan <- fmt.Errorf("error processing package %s: %w", pkgPath, err)
								processingErr = err
								return
							}
						}

						// Only append if we found structs
						if len(fileInfos) > 0 {
							packageInfos = append(packageInfos, fileInfos...)
						}
					}()

					if processingErr != nil && !skipErrors {
						break
					}
				}

				// If we found any structs, add them to the results
				if len(packageInfos) > 0 {
					allInfosMutex.Lock()
					allInfos = append(allInfos, packageInfos)
					allInfosMutex.Unlock()
				}

				// Mark the package as processed
				graph.SetProcessed(pkgPath, processingErr)

				logf("Processed package %s\n", pkgPath)
			}(pkgPath, pkgNode.Package)
		}

		// Wait a bit before checking for more ready packages
		// This reduces CPU usage while waiting
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check for errors if we're not skipping them
	if !skipErrors {
		select {
		case err := <-errChan:
			return allInfos, err
		default:
			// No errors
		}
	} else {
		// Log errors but don't fail
		errors := graph.GetProcessingErrors()
		if len(errors) > 0 {
			logf("Warning: Encountered %d errors during processing\n", len(errors))
			for pkgPath, err := range errors {
				logf("  %s: %v\n", pkgPath, err)
			}
		}
	}

	logf("Analysis complete. Found structs in %d packages.\n", len(allInfos))
	return allInfos, nil
}
