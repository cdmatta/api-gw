package httprouter

import (
	"fmt"
	"io"
	"os"
	"strings"

	graph "github.com/awalterschulze/gographviz"
)

type nodeAttrsFunc func(node *node) map[string]string

// TreeDotWriter provides functions for writing the structure of the tree to
// write buffer. A convenience function for writing the structure directly to
// a file is also provided.
//
// TreeDotWriter writes the tree structure to a plain-text file using the DOT
// language specified by Graphviz. Using the Graphviz's graph description language.
type TreeDotWriter struct {
	ColorScheme                     string
	BaseColorForRootPath            uint32
	BaseColorForVariablePathSegment uint32
	BaseColorForStaticPathSegment   uint32
}

func NewTreeDotExporter() *TreeDotWriter {
	return &TreeDotWriter{
		ColorScheme:                     "set312",
		BaseColorForRootPath:            1,
		BaseColorForVariablePathSegment: 2,
		BaseColorForStaticPathSegment:   6,
	}
}

func (t *TreeDotWriter) WriteToFile(filename string, root *node) error {
	if file, err := os.Create(filename); err == nil {
		defer func() {
			if err := file.Close(); err != nil {
				panic(err)
			}
		}()
		if err := t.Write(file, root); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func (t *TreeDotWriter) Write(w io.Writer, root *node) error {
	const GraphName = "G"

	nodeAttrsFor := func(node *node) map[string]string {
		path := node.path

		attrs := map[string]string{
			"colorscheme": t.ColorScheme,
			"style":       "filled",
		}

		var color uint32

		if strings.HasPrefix(path, ":") || strings.HasPrefix(path, "*") {
			color = t.BaseColorForVariablePathSegment
		} else {
			color = t.BaseColorForStaticPathSegment
		}

		attrs["color"] = fmt.Sprintf("%d", color+node.priority)

		if len(node.children) == 0 {
			attrs["shape"] = "box"
		}

		return attrs
	}
	g := graph.NewEscape()

	if err := g.SetName(GraphName); err != nil {
		return err
	}

	if err := g.SetDir(true); err != nil {
		return err
	}

	var (
		rootPath        = fmt.Sprintf("%s[t=root]", root.path)
		escapedRootPath = `"` + rootPath + `"`
	)

	rootNodeAttrs := map[string]string{
		"color":       fmt.Sprintf("%d", t.BaseColorForRootPath),
		"colorscheme": t.ColorScheme,
		"style":       "filled",
		"shape":       "polygon",
	}

	if err := g.AddNode(GraphName, escapedRootPath, rootNodeAttrs); err != nil {
		return err
	}

	for childNodeIndex := range root.children {
		child := root.children[childNodeIndex]

		var (
			childLabel        = fmt.Sprintf("%s[p=%d]", child.path, child.priority)
			escapedChildLabel = `"` + childLabel + `"`
		)

		childNodeAttrs := nodeAttrsFor(child)

		if err := g.AddNode(escapedRootPath, escapedChildLabel, childNodeAttrs); err != nil {
			return err
		}

		if err := g.AddEdge(escapedRootPath, escapedChildLabel, true, nil); err != nil {
			return err
		}

		if err := treeToDotGraphRecursion(g, child, nodeAttrsFor); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(g.String())); err != nil {
		return err
	}

	return nil
}

func treeToDotGraphRecursion(g *graph.Escape, node *node, nodeAttrsFor nodeAttrsFunc) error {
	var (
		path     = node.path
		priority = node.priority
	)

	if len(path) == 0 {
		return nil
	}

	var (
		label        = fmt.Sprintf("%s[p=%d]", path, priority)
		escapedLabel = `"` + label + `"`
	)

	for childNodeIndex := range node.children {
		var (
			child = node.children[childNodeIndex]
		)

		if len(path) == 0 || len(child.path) == 0 {
			continue
		}

		var (
			childLabel        = fmt.Sprintf("%s[p=%d]", child.path, child.priority)
			escapedChildLabel = `"` + childLabel + `"`
		)

		childNodeAttrs := nodeAttrsFor(child)

		if err := g.AddNode(escapedLabel, escapedChildLabel, childNodeAttrs); err != nil {
			return err
		}

		if err := g.AddEdge(escapedLabel, escapedChildLabel, true, nil); err != nil {
			return err
		}

		if len(child.children) == 0 {
			continue
		}

		if err := treeToDotGraphRecursion(g, child, nodeAttrsFor); err != nil {
			return err
		}
	}

	return nil
}
