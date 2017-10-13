package dependency

import (
	"fmt"
	"strings"
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

		counter := 0
		index := make(map[string]int)
		rindex := make(map[int]string)

		t.Log(strings.Join(tt.edges, "\n"))
		for _, edge := range tt.edges {
			var target, dep string
			if _, err := fmt.Sscanf(edge, "%s -> %s", &target, &dep); err != nil {
				panic("bad test edge " + edge)
			}
			if _, ok := index[target]; !ok {
				index[target] = counter
				rindex[counter] = target
				counter++
			}
			if _, ok := index[dep]; !ok {
				index[dep] = counter
				rindex[counter] = dep
				counter++
			}
			graph.Add(index[target], index[dep])
		}
		var i int
		graph.Flatten(func(vertex int) {
			if i >= len(tt.ordered) {
				t.Fatalf("advanced past expected output with %s", rindex[vertex])
			}
			if index[tt.ordered[i]] != vertex {
				t.Errorf("got %q, wanted %q", rindex[vertex], tt.ordered[i])
			} else {
				t.Log(rindex[vertex])
			}
			i++
		})
		t.Log("")
	}
}
