// Package dependency builds and flattens dependency graphs.
package dependency

import (
	"sort"
	"sync"
)

// insertUnique inserts s into set, preserving order. If s is already in set,
// it is not added. The augmented set is returned.
func insertUnique(set []string, s string) []string {
	i := sort.SearchStrings(set, s)
	if i >= len(set) || set[i] != s {
		set = append(set, "")
		copy(set[i+1:], set[i:])
		set[i] = s
	}
	return set
}

// A Graph is a collection of targets and their dependencies.
type Graph struct {
	once    sync.Once
	targets []string
	nodes   map[string][]string
}

func (g *Graph) init() {
	g.once.Do(func() { g.nodes = make(map[string][]string) })
}

// Add adds a dependency to a Graph.
func (g *Graph) Add(target, dependency string) {
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
func (g *Graph) Flatten(walk func(string)) {
	g.init()
	visited := make(map[string]bool, len(g.nodes))
	for _, tgt := range g.targets {
		if !visited[tgt] {
			visited[tgt] = true
			g.flatten(walk, g.nodes[tgt], visited)
			walk(tgt)
		}
	}
}

func (g *Graph) flatten(fn func(string), targets []string, visited map[string]bool) {
	for _, tgt := range targets {
		if !visited[tgt] {
			visited[tgt] = true
			g.flatten(fn, g.nodes[tgt], visited)
			fn(tgt)
		}
	}
}
