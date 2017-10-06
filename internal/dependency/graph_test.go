package dependency

import (
	"fmt"
	"testing"
)

var flattenTests = [...]struct {
	edges   []string
	ordered []string
}{
	{
		edges: []string{
			"enemy.o -> enemy.c",
			"main.o -> main.c",
			"mygame -> enemy.o",
			"mygame -> main.o",
			"mygame -> player.o",
			"player.o -> player.c",
		},
		ordered: []string{
			"enemy.c",
			"enemy.o",
			"main.c",
			"main.o",
			"player.c",
			"player.o",
			"mygame",
		},
	},
	{
		// Order shouldn't matter
		edges: []string{
			"player.o -> player.c",
			"enemy.o -> enemy.c",
			"mygame -> main.o",
			"main.o -> main.c",
			"mygame -> player.o",
			"mygame -> enemy.o",
		},
		ordered: []string{
			"enemy.c",
			"enemy.o",
			"main.c",
			"main.o",
			"player.c",
			"player.o",
			"mygame",
		},
	},
	{
		// Loops are not followed
		edges: []string{
			"Mildred -> Yancy",
			"Mrs -> Junior",
			"Mrs -> Phillip",
			"Phillip -> Yancy",
			"Yancy -> Junior",
			"Yancy -> Phillip",
		},
		ordered: []string{
			"Junior",
			"Phillip",
			"Yancy",
			"Mildred",
			"Mrs",
		},
	},
}

func TestFlatten(t *testing.T) {
	for _, tt := range flattenTests {
		var graph Graph
		for _, edge := range tt.edges {
			var target string
			var dep string
			if _, err := fmt.Sscanf(edge, "%s -> %s", &target, &dep); err != nil {
				panic("bad test edge " + edge)
			}
			graph.Add(target, dep)
		}
		var i int
		graph.Flatten(func(vertex string) {
			if i >= len(tt.ordered) {
				t.Fatalf("advanced past expected output with %s", vertex)
			}
			if tt.ordered[i] != vertex {
				t.Errorf("got %q, wanted %q", vertex, tt.ordered[i])
			} else {
				t.Log(vertex)
			}
			i++
		})
	}
}
