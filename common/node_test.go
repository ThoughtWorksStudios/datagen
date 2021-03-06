package common

import (
	"fmt"
	. "github.com/ThoughtWorksStudios/bobcat/test_helpers"
	"testing"
)

func TestNodeToString(t *testing.T) {
	location := NewLocation("eek", 2, 2, 2)
	nodeSet := NodeSet{&Node{Kind: "integer", Name: "blah"}}
	node := &Node{
		Kind:     "string",
		Name:     "blah",
		Value:    2,
		Ref:      location,
		Args:     nodeSet,
		Children: nodeSet,
	}

	actual := node.String()
	expected := fmt.Sprintf("{ Kind: \"%s\", Name: \"%s\", Value: %v, Args: %v, Children: %v }", "string", "blah", 2, nodeSet, nodeSet)
	AssertEqual(t, expected, actual)
}

func TestNewLocationReturnsValidLocation(t *testing.T) {
	AssertEqual(t, "whatever.spec:4:8 [byte 42]", NewLocation("whatever.spec", 4, 8, 42).String())
}

func TestHasRelation(t *testing.T) {
	noRelations := &Node{}
	withRelations := &Node{Related: &Node{}}
	Assert(t, !noRelations.HasRelation(), "if node does not have related node, should report false")
	Assert(t, withRelations.HasRelation(), "if node has related node, should report true")
}
