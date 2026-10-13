package domain

import (
	"reflect"
	"testing"
)

func TestNodeWalkVisitsDepthFirstWithParent(t *testing.T) {
	root := Node{ID: 1, Children: []Node{
		{ID: 2, Children: []Node{{ID: 4}}},
		{ID: 3},
	}}

	var got [][2]int
	root.Walk(func(n Node, parent int) { got = append(got, [2]int{n.ID, parent}) })

	want := [][2]int{{1, 0}, {2, 1}, {4, 2}, {3, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("walk order = %v, want %v", got, want)
	}
}
