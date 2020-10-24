package httprouter

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	PanicPatternPathWildcardConflict                     = "conflicting wildcard path '%s' in path segment '%s'"
	PanicPatternHandlerAlreadyExists                     = "a handler already exists for path '%s'"
	PanicPatternMultipleWildcardsInOnePathSegment        = "multiple wildcards in path '%s'"
	PanicPatternNodeHasNoChildAtPosition                 = "a node with path '%s' has no child node at position %d"
	PanicPatternWildcardSegmentConflictWithExistingChild = "wildcard segment conflicts with an existing child in path '%s'"
)

func countRequestPathParams(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' {
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

type nodeType uint8

const (
	static nodeType = iota
	root
	param
	catchAll
)

// The node type structure represents a node of a radix tree of URI paths that contain zero or more variable path
// segments. These URI paths are referred as routes as it associates a URI path to a handler function.
//
// Routes
//
// The following examples describes the key concept:
//
// - The path `/api/vi/users` is a route that has no variable path segments, and a handler function may be resolved
//   for a matching URI path.
//
// Purpose and scope
//
// The radix tree breakdown of routes is meant to Resolve the handler and values of variable path segments with
// strong emphasis on performance. Path parameters refer to parsed values of the variable path segments, as defined
// by the route.
//
// If is not in the scope of the radix tree implementation to perform any operations upon resolving a URI to a
// specific path, other than providing it to the user calling the Resolve function.
//
// Child nodes
//
// Children and indices are kept in sync with respect to the slice index, i.e. the first byte character of a path
// of child, found in children at a given index, is found in the indices by the same index.
type node struct {
	// The path is a part or segment of the route path.
	path string

	// The routePath is the route path this node is associated to.
	routePath string

	// The children is a slice of child nodes ordered by the priority of the child node in a descending order.
	// The child node with the highest priority as the first element.
	// Likewise the child with the lowest priority as the last element.
	children []*node

	// The indices is a slice of byte characters, where each element holds the first byte character of a path of a child.
	// Note that indices and children are logically related
	indices []byte

	// The wildChild boolean field indicates whether the node has any children with variable path segments.
	wildChild bool

	// The nodeType field indicates the type of the node. Refer to nodeType for details on various types of nodes.
	nodeType nodeType

	maxParameters uint8

	// The priority field reflects the priority of the node in respect to the sibling nodes.
	// This field is logically related to indices and children, as both slices are ordered based on the priority.
	priority uint32

	handler http.Handler
}

//goland:noinspection GoAssignmentToReceiver
func (n *node) AddRoute(path string, handler http.Handler) {
	// Returns the length of a common prefix of `path1` and `path2` arguments.
	// This length can be interpreted also as the position (or index) marking the end of the longest common prefix.
	commonPrefixLength := func(path1, path2 string) int {
		// Use the length of the shortest path as the upper boundary.
		maxLength := min(len(path1), len(path2))

		// Then, iterate through each (byte) character and check
		// if the path strings hold the same character at given position.
		for pos := 0; pos < maxLength; pos++ {
			if path1[pos] != path2[pos] {
				return pos
			}
		}
		return maxLength
	}

	n.priority++
	parameterCount := countRequestPathParams(path)

	routePath := path

	if len(n.path) > 0 || len(n.children) > 0 {
	walk:
		for {
			// Update the current node parameter count,
			// based on the previously counted variable path segments, if needed
			if parameterCount > n.maxParameters {
				n.maxParameters = parameterCount
			}

			// Capture the longest common prefix of a path by updating the read cursor until finding the first character
			pos := commonPrefixLength(path, n.path)

			// Split
			if pos < len(n.path) {
				child := node{
					path:      n.path[pos:],
					wildChild: n.wildChild,
					indices:   n.indices,
					children:  n.children,
					handler:   n.handler,
					priority:  n.priority - 1,
				}

				// Update the maximum number of variable path segments for all child nodes associated to the node.
				for childNodeIndex := range child.children {
					if child.children[childNodeIndex].maxParameters > child.maxParameters {
						child.maxParameters = child.children[childNodeIndex].maxParameters
					}
				}

				n.children = []*node{&child}
				n.indices = []byte{n.path[pos]}
				n.path = path[:pos]
				n.handler = nil
				n.wildChild = false
			}

			// Add a new child node to the node.
			if pos < len(path) {
				path = path[pos:]

				if n.wildChild {
					n = n.children[0]
					n.priority++

					// Update the parameter count of the node, if needed
					if parameterCount > n.maxParameters {
						n.maxParameters = parameterCount
					}
					parameterCount--

					if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
						if len(n.path) >= len(path) || path[len(n.path)] == '/' {
							continue walk
						}
					}

					msg := fmt.Sprintf(PanicPatternPathWildcardConflict, path, n.path)
					panic(msg)
				}

				characterAtIndex := path[0]

				// If a path parameter is followed by a slash,
				// given that the current path segment has one and only one child node.
				// Then, proceed processing the remaining path segments.
				if n.nodeType == param && characterAtIndex == '/' && len(n.children) == 1 {
					n = n.children[0]
					n.priority++
					continue walk
				}

				// If a child node exists that starts with a byte character that matches the byte character
				// of the currently processed path segment, increment its priority.
				// Then, proceed processing the remaining path segments.
				for index, character := range n.indices {
					if character == characterAtIndex {
						index = n.incrementChildNodePriorityAndSwapIfNeeded(index)
						n = n.children[index]
						continue walk
					}
				}

				// If none of the above conditions hold, treat the path segment as a new child node,
				// given that it is not defining a variable path segment.
				if characterAtIndex != ':' && characterAtIndex != '*' {
					n.indices = append(n.indices, characterAtIndex)
					child := &node{
						routePath:     routePath,
						maxParameters: parameterCount,
					}
					n.children = append(n.children, child)
					n.incrementChildNodePriorityAndSwapIfNeeded(len(n.indices) - 1)
					n = child
				}
				n.insertChild(parameterCount, routePath, path, handler)
				return
			} else if pos == len(path) {
				// Treat the child node as a leaf node
				if n.handler != nil {
					msg := fmt.Sprintf(PanicPatternHandlerAlreadyExists, path)
					panic(msg)
				}
				n.routePath = routePath
				n.handler = handler
			}
			return
		}
	} else {
		n.insertChild(parameterCount, routePath, path, handler)
	}
}

//goland:noinspection GoAssignmentToReceiver
func (n *node) Resolve(path string) (http.Handler, *PathParameters, bool) {
	var (
		handler http.Handler
		ps      *PathParameters
		tsr     bool
	)
walk:
	for {
		if len(path) > len(n.path) {
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):]
				if !n.wildChild {
					characterAtIndex := path[0]
					for i, index := range n.indices {
						if characterAtIndex == index {
							n = n.children[i]
							continue walk
						}
					}
					tsrf := func() bool {
						return path == "/" && n.handler != nil
					}
					tsr = tsrf()

					return handler, ps, tsr
				}

				n = n.children[0]
				switch n.nodeType {
				case param:
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					if ps == nil {
						ps = NewPathParameters(n.routePath, n.maxParameters)
					}

					i := len(ps.parameters)
					ps.parameters = ps.parameters[:i+1]
					ps.parameters[i].Key = n.path[1:]
					ps.parameters[i].Value = path[:end]

					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}
						tsrf := func() bool {
							return len(path) == end+1
						}
						tsr = tsrf()
						return handler, ps, tsr
					}

					if handler = n.handler; handler != nil {
						if ps != nil && len(n.routePath) > 0 {
							ps.route = n.routePath
						}
						return handler, ps, tsr
					}

					if len(n.children) == 1 {
						tsrf := func() bool {
							if n.path == "/" && n.handler != nil {
								return true
							}
							return false
						}
						n = n.children[0]
						tsr = tsrf()
					}
					return handler, ps, tsr

				case catchAll:
					if ps == nil {
						ps = NewPathParameters(n.routePath, n.maxParameters)
					}

					parameterCount := len(ps.parameters)
					ps.parameters = ps.parameters[:parameterCount+1]
					ps.parameters[parameterCount].Key = n.path[2:]
					ps.parameters[parameterCount].Value = path

					handler = n.handler

					return handler, ps, tsr

				default:
					panic("invalid node type")
				}
			}
		} else if path == n.path {
			if handler = n.handler; handler != nil {
				if ps != nil && len(n.routePath) > 0 {
					ps.route = n.routePath
				}
				return handler, ps, tsr
			}

			for i, index := range n.indices {
				if index != '/' {
					continue
				}
				tsrf := func() bool {
					if n.path == "/" && n.handler != nil {
						return true
					}
					if n.nodeType == catchAll && n.children[0].handler != nil {
						return true
					}
					return false
				}
				n = n.children[i]
				if ps != nil {
					ps.route = n.routePath
				}
				tsr = tsrf()
				return handler, ps, tsr
			}
			return handler, ps, tsr
		}

		tsrf := func() bool {
			if path == "/" {
				return true
			}
			if len(n.path) == len(path)+1 && n.path[len(path)] == '/' && path == n.path[:len(n.path)-1] && n.handler != nil {
				return true
			}
			return false
		}
		tsr = tsrf()
		return handler, ps, tsr
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Internal
/////////////////////////////////////////////////////////////////////////////////////////////////////////////

//goland:noinspection GoAssignmentToReceiver
func (n *node) insertChild(parameterCount uint8, routePath, path string, handler http.Handler) {
	var offset int

	for i, max := 0, len(path); parameterCount > 0; i++ {
		characterAtIndex := path[i]
		if characterAtIndex != ':' && characterAtIndex != '*' {
			continue
		}

		if len(n.children) > 0 {
			msg := fmt.Sprintf(PanicPatternWildcardSegmentConflictWithExistingChild, path)
			panic(msg)
		}

		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			case ':', '*':
				msg := fmt.Sprintf(PanicPatternMultipleWildcardsInOnePathSegment, path)
				panic(msg)
			default:
				end++
			}
		}

		if end-i < 2 {
			panic("wildcards must be named with a non-empty name")
		}

		if characterAtIndex == ':' {
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			child := &node{
				routePath:     routePath,
				nodeType:      param,
				maxParameters: parameterCount,
			}

			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++

			parameterCount--

			if end < max {
				n.path = path[offset:end]
				offset = end
				child := &node{
					maxParameters: parameterCount,
					priority:      1,
				}
				n.children = []*node{child}
				n = child
			}
		} else {
			if end != max || parameterCount > 1 {
				panic("catch-all routes are only allowed at the end of the path")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root")
			}

			i--
			if path[i] != '/' {
				panic("no / before catch-all")
			}

			n.path = path[offset:i]

			child := &node{
				wildChild:     true,
				nodeType:      catchAll,
				maxParameters: 1,
			}
			n.children = []*node{child}
			n.indices = []byte{path[i]}
			n = child
			n.priority++

			child = &node{
				path:          path[i:],
				routePath:     routePath,
				nodeType:      catchAll,
				maxParameters: 1,
				handler:       handler,
				priority:      1,
			}
			n.children = []*node{child}
			return
		}
	}

	n.path = path[offset:]
	n.handler = handler
}

// Makes a case-insensitive lookup of the given path and tries to find a handler.
// It can optionally also fix trailing slashes.
// It returns the case-corrected path and a bool indicating whether the lookup
// was successful.
func (n *node) findCaseInsensitivePath(path string, fixTrailingSlash bool) (fixedPath string, found bool) {
	const stackBufSize = 128

	// Use a static sized buffer on the stack in the common case.
	// If the path is too long, allocate a buffer on the heap instead.
	buf := make([]byte, 0, stackBufSize)
	if l := len(path) + 1; l > stackBufSize {
		buf = make([]byte, 0, l)
	}

	ciPath := n.findCaseInsensitivePathRec(
		path,
		buf,       // Preallocate enough memory for new path
		[4]byte{}, // Empty rune buffer
		fixTrailingSlash,
	)

	return string(ciPath), ciPath != nil
}

// Shift bytes in array by n bytes left
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3], 0}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

// Recursive case-insensitive lookup function used by n.findCaseInsensitivePath
//goland:noinspection GoAssignmentToReceiver
func (n *node) findCaseInsensitivePathRec(path string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) []byte {
	npLen := len(n.path)

walk: // Outer loop for walking the tree
	for len(path) >= npLen && (npLen == 0 || strings.EqualFold(path[1:npLen], n.path[1:])) {
		// Add common prefix to result
		oldPath := path
		path = path[npLen:]
		ciPath = append(ciPath, n.path...)

		if len(path) > 0 {
			// If this node does not have a wildcard (param or catchAll) child,
			// we can just look up the next child node and continue to walk down
			// the tree
			if !n.wildChild {
				// Skip rune bytes already processed
				rb = shiftNRuneBytes(rb, npLen)

				if rb[0] != 0 {
					// Old rune not finished
					characterAtIndex := rb[0]
					for i, c := range n.indices {
						if c == characterAtIndex {
							// continue with child node
							n = n.children[i]
							npLen = len(n.path)
							continue walk
						}
					}
				} else {
					// Process a new rune
					var rv rune

					// Find rune start.
					// Runes are up to 4 byte long,
					// -4 would definitely be another rune.
					var off int
					for max := min(npLen, 3); off < max; off++ {
						if i := npLen - off; utf8.RuneStart(oldPath[i]) {
							// read rune from cached path
							rv, _ = utf8.DecodeRuneInString(oldPath[i:])
							break
						}
					}

					// Calculate lowercase bytes of current rune
					lo := unicode.ToLower(rv)
					utf8.EncodeRune(rb[:], lo)

					// Skip already processed bytes
					rb = shiftNRuneBytes(rb, off)

					characterAtIndex := rb[0]
					for i, c := range n.indices {
						// Lowercase matches
						if c == characterAtIndex {
							// must use a recursive approach since both the
							// uppercase byte and the lowercase byte might exist
							// as an index
							if out := n.children[i].findCaseInsensitivePathRec(
								path, ciPath, rb, fixTrailingSlash,
							); out != nil {
								return out
							}
							break
						}
					}

					// If we found no match, the same for the uppercase rune,
					// if it differs
					if up := unicode.ToUpper(rv); up != lo {
						utf8.EncodeRune(rb[:], up)
						rb = shiftNRuneBytes(rb, off)

						characterAtIndex := rb[0]
						for i, c := range n.indices {
							// Uppercase matches
							if c == characterAtIndex {
								// Continue with child node
								n = n.children[i]
								npLen = len(n.path)
								continue walk
							}
						}
					}
				}

				// Nothing found. We can recommend to redirect to the same URL
				// without a trailing slash if a leaf exists for that path
				if fixTrailingSlash && path == "/" && n.handler != nil {
					return ciPath
				}
				return nil
			}

			n = n.children[0]
			switch n.nodeType {
			case param:
				// Find param end (either '/' or path end)
				end := 0
				for end < len(path) && path[end] != '/' {
					end++
				}

				// Add param value to case insensitive path
				ciPath = append(ciPath, path[:end]...)

				// We need to go deeper!
				if end < len(path) {
					if len(n.children) > 0 {
						// Continue with child node
						n = n.children[0]
						npLen = len(n.path)
						path = path[end:]
						continue
					}

					// ... but we can't
					if fixTrailingSlash && len(path) == end+1 {
						return ciPath
					}
					return nil
				}

				if n.handler != nil {
					return ciPath
				} else if fixTrailingSlash && len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists
					n = n.children[0]
					if n.path == "/" && n.handler != nil {
						return append(ciPath, '/')
					}
				}
				return nil

			case catchAll:
				return append(ciPath, path...)

			default:
				panic("invalid node type")
			}
		} else {
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if n.handler != nil {
				return ciPath
			}

			// No handle found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i, c := range n.indices {
					if c == '/' {
						n = n.children[i]
						if len(n.path) == 1 && n.handler != nil {
							return append(ciPath, '/')
						}
						if n.nodeType == catchAll && n.children[0].handler != nil {
							return append(ciPath, '/')
						}
						return nil
					}
				}
			}
			return nil
		}
	}

	// Nothing found.
	if fixTrailingSlash == false {
		return nil
	}

	// Try to fix the path by adding / removing a trailing slash
	if path == "/" {
		return ciPath
	}

	if n.handler == nil {
		return nil
	}

	if len(path)+1 != npLen {
		return nil
	}

	if n.path[len(path)] != '/' {
		return nil
	}

	if strings.EqualFold(path[1:], n.path[1:len(path)]) {
		return append(ciPath, n.path...)
	}

	return nil
}

// Increments priority of the given child and reorders if necessary
func (n *node) incrementChildNodePriorityAndSwapIfNeeded(posOfChildToPrioritize int) int {
	if posOfChildToPrioritize >= len(n.children) {
		msg := fmt.Sprintf(PanicPatternNodeHasNoChildAtPosition, n.path, posOfChildToPrioritize)
		panic(msg)
	}

	// First, increment the priority of the child node of the node at the given position
	n.children[posOfChildToPrioritize].priority++
	priority := n.children[posOfChildToPrioritize].priority

	// Then, check if the child which priority was incremented has a sibling, prior to sorting
	if posOfChildToPrioritize == 0 {
		return posOfChildToPrioritize
	}

	// Then, reorder the children based on the priorities so that slices holding the children and respective
	// indexed character remain sorted by priority.
	//
	// Now that priority of a child node at the specified position was incremented by one, the child node will
	// be traversed starting from the previous sibling node until all previous sibling have been traversed.

	// Previous siblings are all those child nodes that have a small p
	for siblingChildPos := posOfChildToPrioritize - 1; siblingChildPos >= 0; siblingChildPos-- {
		// Stop traversing through the previous siblings, when the sibling has a higher priority.
		// Given that the children of a node are stored in a slice in a descending priority order,
		// traversing is stopped, if a previous sibling has a higher priority, as the remaining
		// previous siblings will holder a higher priority as well.

		if n.children[siblingChildPos].priority >= priority {
			break
		}

		nodeToSwap := n.children[siblingChildPos]
		n.children[siblingChildPos] = n.children[posOfChildToPrioritize]
		n.children[posOfChildToPrioritize] = nodeToSwap

		characterIndexEntryToSwap := n.indices[siblingChildPos]
		n.indices[siblingChildPos] = n.indices[posOfChildToPrioritize]
		n.indices[posOfChildToPrioritize] = characterIndexEntryToSwap

		posOfChildToPrioritize--
	}

	return posOfChildToPrioritize
}
