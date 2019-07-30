// Package dependency builds and flattens dependency graphs.
package dependency // import "aqwari.net/xml/internal/dependency"

import (
	"sort"
	"sync"
)

// insertUnique inserts s into set, preserving order. If s is already in set,
// it is not added. The augmented set is returned.
func insertUnique(set []int, x int) []int {
	i := sort.SearchInts(set, x)
	if i >= len(set) || set[i] != x {
		set = append(set, 0)
		copy(set[i+1:], set[i:])
		set[i] = x
	}
	return set
}

// A Graph is a collection of targets and their dependencies.
type Graph struct {
	once    sync.Once
	targets []int
	nodes   map[int][]int
}

// Len returns the number of targets in the graph.
func (g *Graph) Len() int {
	return len(g.targets)
}

func (g *Graph) init() {
	g.once.Do(func() { g.nodes = make(map[int][]int) })
}

// Add adds a dependency to a Graph.
func (g *Graph) Add(target, dependency int) {
	g.init()
	g.targets = insertUnique(g.targets, target)
	g.nodes[target] = insertUnique(g.nodes[target], dependency)
}

// Flatten calls the walk function on each node in the Graph in topological
// order, starting with the leaves and traversing up to the roots.  The same
// Graph will always be traversed in the same order.
//
// Every vertex in the Graph is visited once; any cycles in the graph are
// skipped.
func (g *Graph) Flatten(walk func(int)) {
	g.init()
	visited := make(map[int]bool, len(g.nodes))
	for _, tgt := range g.targets {
		if !visited[tgt] {
			visited[tgt] = true
			g.flatten(walk, g.nodes[tgt], visited)
			walk(tgt)
		}
	}
}

func (g *Graph) flatten(fn func(int), targets []int, visited map[int]bool) {
	for _, tgt := range targets {
		if !visited[tgt] {
			visited[tgt] = true
			g.flatten(fn, g.nodes[tgt], visited)
			fn(tgt)
		}
	}
}
