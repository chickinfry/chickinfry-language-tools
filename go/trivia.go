package chiklang

// attachTrivia distributes trivia to nodes following the rules in spec/cst.md:
// trivia on the same line after a node is that node's trailing trivia; the rest
// is the next node's leading trivia; anything left belongs to the file.
//
// This runs after parsing so the parser never has to think about whitespace.
func attachTrivia(root *Node, trivia []Trivia) {
	if len(trivia) == 0 {
		return
	}
	leaves := collectLeaves(root)
	index := 0
	for _, t := range trivia {
		// Find the first leaf that starts after this trivia ends.
		for index < len(leaves) && leaves[index].start < t.End {
			index++
		}
		if index == 0 {
			if len(leaves) == 0 {
				root.Leading = append(root.Leading, t)
				continue
			}
			leaves[0].node.Leading = append(leaves[0].node.Leading, t)
			continue
		}
		previousIndex := index - 1
		previous := leaves[previousIndex]
		// Same line as the previous leaf means it trails that leaf.
		if t.Kind != Newline && t.Range.Start.Line == previous.endLine {
			previous.node.Trailing = append(previous.node.Trailing, t)
			continue
		}
		if index < len(leaves) {
			leaves[index].node.Leading = append(leaves[index].node.Leading, t)
			continue
		}
		root.Trailing = append(root.Trailing, t)
	}
}

type leaf struct {
	start   int
	endLine int
	node    *Node
}

// collectLeaves lists every node that owns a token, in source order.
func collectLeaves(root *Node) []leaf {
	var leaves []leaf
	var walk func(*Node)
	walk = func(node *Node) {
		for _, token := range node.Tokens {
			leaves = append(leaves, leaf{start: token.Start, endLine: token.Line, node: node})
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)
	return leaves
}
