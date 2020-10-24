package httprouter

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
// Fixture types
///////////////////////////////////////////////////////////////////////////////////////////////////

type pathConflictTestFixture struct {
	path     string
	conflict bool
}

type findCaseInsensitivePathFixture struct {
	in    string
	out   string
	found bool
	slash bool
}

type requestRoutingFixture []struct {
	path       string
	nilHandler bool
	route      string
	parameters *PathParameters
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Test functions
///////////////////////////////////////////////////////////////////////////////////////////////////

func TestTree_CountParams(t *testing.T) {
	if countRequestPathParams("/path/:param1/static/*catch-all") != 2 {
		t.Fail()
	}
	if countRequestPathParams(strings.Repeat("/:param", 256)) != 255 {
		t.Fail()
	}
}

func TestTree_AddAndGetRoute(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
	}

	for _, route := range routes {
		test.addRoute(route, route)
	}

	fixture := requestRoutingFixture{
		{
			"/a",
			false,
			"/a",
			nil,
		},
		{
			"/",
			true,
			"",
			nil,
		},
		{
			"/hi",
			false,
			"/hi",
			nil,
		},
		{
			"/contact",
			false,
			"/contact",
			nil,
		},
		// key mismatch
		{
			"/co",
			false,
			"/co",
			nil,
		},
		{
			"/con",
			true,
			"",
			nil,
		},
		// key mismatch
		{
			"/cona",
			true,
			"",
			nil,
		},
		// no matching child
		{
			"/no",
			true,
			"",
			nil,
		},
		{
			"/ab",
			false,
			"/ab",
			nil,
		},
	}

	test.assertResolutions(fixture)
	test.assertNodePriorities()
	test.assertNodeMaxParameters()
}

func TestTree_Wildcard(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name",
		"/user_:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
	}

	for _, route := range routes {
		test.addRoute(route, route)
	}

	fixture := requestRoutingFixture{
		{
			path:       "/",
			nilHandler: false,
			route:      "/",
			parameters: nil,
		},
		{
			path:       "/cmd/test/",
			nilHandler: false,
			route:      "/cmd/:tool/",
			parameters: NewPathParameters("/cmd/:tool/", 1).
				AddParameter("tool", "test"),
		},
		{
			path:       "/cmd/test",
			nilHandler: true,
			route:      "",
			parameters: NewPathParameters("/cmd/test", 1).
				AddParameter("tool", "test"),
		},
		{
			path:       "/cmd/test/3",
			nilHandler: false,
			route:      "/cmd/:tool/:sub",
			parameters: NewPathParameters("/cmd/:tool/:sub", 2).
				AddParameter("tool", "test").
				AddParameter("sub", "3"),
		},
		{
			path:       "/src/",
			nilHandler: false,
			route:      "/src/*filepath",
			parameters: NewPathParameters("/src/*filepath", 1).
				AddParameter("filepath", "/"),
		},
		{
			path:       "/src/some/file.png",
			nilHandler: false,
			route:      "/src/*filepath",
			parameters: NewPathParameters("/src/*filepath", 1).
				AddParameter("filepath", "/some/file.png"),
		},
		{
			path:       "/search/",
			nilHandler: false,
			route:      "/search/",
			parameters: nil,
		},
		{
			path:       "/search/someth!ng+in+ünìcodé",
			nilHandler: false,
			route:      "/search/:query",
			parameters: NewPathParameters("/search/:query", 1).
				AddParameter("query", "someth!ng+in+ünìcodé"),
		},
		{
			path:       "/search/someth!ng+in+ünìcodé/",
			nilHandler: true,
			route:      "",
			parameters: NewPathParameters("", 1).
				AddParameter("query", "someth!ng+in+ünìcodé"),
		},
		{
			path:       "/user_gopher",
			nilHandler: false,
			route:      "/user_:name",
			parameters: NewPathParameters("/user_:name", 1).
				AddParameter("name", "gopher"),
		},
		{
			path:       "/user_gopher/about",
			nilHandler: false,
			route:      "/user_:name/about",
			parameters: NewPathParameters("/user_:name/about", 1).
				AddParameter("name", "gopher"),
		},
		{
			path:       "/files/js/inc/framework.js",
			nilHandler: false,
			route:      "/files/:dir/*filepath",
			parameters: NewPathParameters("/files/:dir/*filepath", 2).
				AddParameter("dir", "js").
				AddParameter("filepath", "/inc/framework.js"),
		},
		{
			path:       "/info/gordon/public",
			nilHandler: false,
			route:      "/info/:user/public",
			parameters: NewPathParameters("/info/:user/public", 1).
				AddParameter("user", "gordon"),
		},
		{
			path:       "/info/gordon/project/go",
			nilHandler: false,
			route:      "/info/:user/project/:project",
			parameters: NewPathParameters("/info/:user/project/:project", 2).
				AddParameter("user", "gordon").
				AddParameter("project", "go"),
		},
	}

	test.assertResolutions(fixture)
	test.assertNodePriorities()
	test.assertNodeMaxParameters()
}

func TestTree_WildcardConflict(t *testing.T) {
	test := newTreeRoutingTest(t)

	fixture := []pathConflictTestFixture{
		{
			path:     "/cmd/:tool/:sub",
			conflict: false,
		},
		{
			path:     "/cmd/vet",
			conflict: true,
		},
		{
			path:     "/src/*filepath",
			conflict: false,
		},
		{
			path:     "/src/*filepathx",
			conflict: true,
		},
		{
			path:     "/src/",
			conflict: true,
		},
		{
			path:     "/src1/",
			conflict: false,
		},
		{
			path:     "/src1/*filepath",
			conflict: true,
		},
		{
			path:     "/src2*filepath",
			conflict: true,
		},
		{
			path:     "/search/:query",
			conflict: false,
		},
		{
			path:     "/search/invalid",
			conflict: true,
		},
		{
			path:     "/user_:name",
			conflict: false,
		},
		{
			path:     "/user_x",
			conflict: true,
		},
		{
			path:     "/user_:name",
			conflict: false,
		},
		{
			path:     "/id:id",
			conflict: false,
		},
		{
			path:     "/id/:id",
			conflict: true,
		},
	}

	test.assertRouteConflicts(fixture)
}

func TestTree_ChildConflict(t *testing.T) {
	test := newTreeRoutingTest(t)

	fixture := []pathConflictTestFixture{
		{
			path:     "/cmd/vet",
			conflict: false,
		},
		{
			path:     "/cmd/:tool/:sub",
			conflict: true,
		},
		{
			path:     "/src/AUTHORS",
			conflict: false,
		},
		{
			path:     "/src/*filepath",
			conflict: true,
		},
		{
			path:     "/user_x",
			conflict: false,
		},
		{
			path:     "/user_:name",
			conflict: true,
		},
		{
			path:     "/id/:id",
			conflict: false,
		},
		{
			path:     "/id:id",
			conflict: true,
		},
		{
			path:     "/:id",
			conflict: true,
		},
		{
			path:     "/*filepath",
			conflict: true,
		},
	}

	test.assertRouteConflicts(fixture)
}

func TestTree_DuplicatePath(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}

	for _, route := range routes {
		recv := catchPanic(func() {
			test.addRoute(route, route)
		})
		if recv != nil {
			t.Fatalf("[addRoute] panic occurred, expected none, '%s': %v", route, recv)
		}

		recv = catchPanic(func() {
			test.addRoute(route, empty)
		})
		if recv == nil {
			t.Fatalf("[addRoute] expected panic, none occurred, route: '%s'", route)
		}
	}

	fixture := requestRoutingFixture{
		{
			path:       "/",
			nilHandler: false,
			route:      "/",
			parameters: nil,
		},
		{
			path:       "/doc/",
			nilHandler: false,
			route:      "/doc/",
			parameters: nil,
		},
		{
			path:       "/src/some/file.png",
			nilHandler: false,
			route:      "/src/*filepath",
			parameters: NewPathParameters("/src/*filepath", 1).
				AddParameter("filepath", "/some/file.png"),
		},
		{
			path:       "/search/someth!ng+in+ünìcodé",
			nilHandler: false,
			route:      "/search/:query",
			parameters: NewPathParameters("/search/:query", 1).
				AddParameter("query", "someth!ng+in+ünìcodé"),
		},
		{
			path:       "/user_gopher",
			nilHandler: false,
			route:      "/user_:name",
			parameters: NewPathParameters("/user_:name", 1).
				AddParameter("name", "gopher"),
		},
	}

	test.assertResolutions(fixture)
}

func TestTree_EmptyWildcardName(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}

	for _, route := range routes {
		recv := catchPanic(func() {
			test.addRoute(route, empty)
		})
		if recv == nil {
			t.Fatalf("[addRoute] expected panic, none occurred, route: '%s'", route)
		}
	}
}

func TestTree_CatchAllConflict(t *testing.T) {
	test := newTreeRoutingTest(t)

	fixture := []pathConflictTestFixture{
		{
			path:     "/src2/",
			conflict: false,
		},
		{
			path:     "/src/*filepath/x",
			conflict: true,
		},
		{
			path:     "/src2/*filepath/x",
			conflict: true,
		},
	}

	test.assertRouteConflicts(fixture)
}

func TestTree_CatchAllConflictRoot(t *testing.T) {
	test := newTreeRoutingTest(t)

	fixture := []pathConflictTestFixture{
		{
			path:     "/",
			conflict: false,
		},
		{
			path:     "/*filepath",
			conflict: true,
		},
	}

	test.assertRouteConflicts(fixture)
}

func TestTree_DoubleWildcard(t *testing.T) {
	const expectedPanicMsgPrefix = "multiple wildcards in path"

	routes := [...]string{
		"/:foo:bar",
		"/:foo:bar/",
		"/:foo*bar",
	}

	for _, route := range routes {
		tree := &node{}
		recv := catchPanic(func() {
			tree.AddRoute(route, nil)
		})

		actualPanicMsg, panicked := recv.(string)

		if panicked == false {
			t.Fatalf("[addRoute] expected panic, none occurred, route: '%s'", route)
		}

		if strings.HasPrefix(actualPanicMsg, expectedPanicMsgPrefix) == false {
			t.Fatalf("[addRoute] unexpected panic message prefix, expected: '%s', actual: '%s', route: '%s'",
				expectedPanicMsgPrefix,
				actualPanicMsg,
				route,
			)
		}
	}
}

func TestTree_TrailingSlashRedirect(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
		"/api/hello/:name",
	}

	for _, route := range routes {
		recv := catchPanic(func() {
			test.addRoute(route, route)
		})
		if recv != nil {
			t.Fatalf("[addRoute] panic occurred, expected none, '%s': %v", route, recv)
		}
	}

	tsrRoutes := [...]string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/src",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/doc/",
	}

	for _, route := range tsrRoutes {
		handler, _, tsr := test.resolvePath(route)
		if handler != nil {
			t.Fatalf("[resolvePath] non-nil handler for TSR path route: '%s'", route)
		}
		if !tsr {
			t.Errorf("[resolvePath] expected TSR recommendation for route: '%s'", route)
		}
	}

	noTsrRoutes := [...]string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
		"/api/world/abc",
	}

	for _, route := range noTsrRoutes {
		handler, _, tsr := test.resolvePath(route)
		if handler != nil {
			t.Fatalf("[resolvePath] non-nil handler for non-TSR path, route: '%s'",
				route)
		}
		if tsr {
			t.Errorf("[resolvePath] expected no TSR recommendation for non-TSR path, route: '%s'",
				route)
		}
	}
}

func TestTree_FindCaseInsensitivePath(t *testing.T) {
	test := newTreeRoutingTest(t)

	routes := [...]string{
		"/hi",
		"/b/",
		"/ABC/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/doc/go/away",
		"/no/a",
		"/no/b",
	}

	for _, route := range routes {
		recv := catchPanic(func() {
			test.addRoute(route, route)
		})
		if recv != nil {
			t.Fatalf("[addRoute] panic occurred, route: '%s', msg: %v", route, recv)
		}
	}

	for _, route := range routes {
		foundPath, found := test.findCaseInsensitivePath(route, true)
		if !found {
			t.Errorf("[findCaseInsensitivePath] expected to find route, route: '%s'", route)
			continue
		}

		var (
			expectedPath = route
			actualPath   = foundPath
		)

		if actualPath != expectedPath {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%s', was: '%s', route: '%s'",
				expectedPath, actualPath, route)
		}
	}

	for _, route := range routes {
		foundPath, found := test.findCaseInsensitivePath(route, false)
		if !found {
			t.Errorf("[findCaseInsensitivePath] expected to find route, route: '%s'", route)
			continue
		}

		var (
			expectedPath = route
			actualPath   = foundPath
		)

		if actualPath != expectedPath {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%s', was: '%s', route: '%s'",
				expectedPath, actualPath, route)
		}
	}

	fixture := []findCaseInsensitivePathFixture{
		{
			in:    "/HI",
			out:   "/hi",
			found: true,
			slash: false,
		},
		{
			in:    "/HI/",
			out:   "/hi",
			found: true,
			slash: true,
		},
		{
			in:    "/B",
			out:   "/b/",
			found: true,
			slash: true,
		},
		{
			in:    "/B/",
			out:   "/b/",
			found: true,
			slash: false,
		},
		{
			in:    "/abc",
			out:   "/ABC/",
			found: true,
			slash: true,
		},
		{
			in:    "/abc/",
			out:   "/ABC/",
			found: true,
			slash: false,
		},
		{
			in:    "/aBc",
			out:   "/ABC/",
			found: true,
			slash: true,
		},
		{
			in:    "/aBc/",
			out:   "/ABC/",
			found: true,
			slash: false,
		},
		{
			in:    "/abC",
			out:   "/ABC/",
			found: true,
			slash: true,
		},
		{
			in:    "/abC/",
			out:   "/ABC/",
			found: true,
			slash: false,
		},
		{
			in:    "/SEARCH/QUERY",
			out:   "/search/QUERY",
			found: true,
			slash: false,
		},
		{
			in:    "/SEARCH/QUERY/",
			out:   "/search/QUERY",
			found: true,
			slash: true,
		},
		{
			in:    "/CMD/TOOL/",
			out:   "/cmd/TOOL/",
			found: true,
			slash: false,
		},
		{
			in:    "/CMD/TOOL",
			out:   "/cmd/TOOL/",
			found: true,
			slash: true,
		},
		{
			in:    "/SRC/FILE/PATH",
			out:   "/src/FILE/PATH",
			found: true,
			slash: false,
		},
		{
			in:    "/x/Y",
			out:   "/x/y",
			found: true,
			slash: false,
		},
		{
			in:    "/x/Y/",
			out:   "/x/y",
			found: true,
			slash: true,
		},
		{
			in:    "/X/y",
			out:   "/x/y",
			found: true,
			slash: false,
		},
		{
			in:    "/X/y/",
			out:   "/x/y",
			found: true,
			slash: true,
		},
		{
			in:    "/X/Y",
			out:   "/x/y",
			found: true,
			slash: false,
		},
		{
			in:    "/X/Y/",
			out:   "/x/y",
			found: true,
			slash: true,
		},
		{
			in:    "/Y/",
			out:   "/y/",
			found: true,
			slash: false,
		},
		{
			in:    "/Y",
			out:   "/y/",
			found: true,
			slash: true,
		},
		{
			in:    "/Y/z",
			out:   "/y/z",
			found: true,
			slash: false,
		},
		{
			in:    "/Y/z/",
			out:   "/y/z",
			found: true,
			slash: true,
		},
		{
			in:    "/Y/Z",
			out:   "/y/z",
			found: true,
			slash: false,
		},
		{
			in:    "/Y/Z/",
			out:   "/y/z",
			found: true,
			slash: true,
		},
		{
			in:    "/y/Z",
			out:   "/y/z",
			found: true,
			slash: false,
		},
		{
			in:    "/y/Z/",
			out:   "/y/z",
			found: true,
			slash: true,
		},
		{
			in:    "/Aa",
			out:   "/aa",
			found: true,
			slash: false,
		},
		{
			in:    "/Aa/",
			out:   "/aa",
			found: true,
			slash: true,
		},
		{
			in:    "/AA",
			out:   "/aa",
			found: true,
			slash: false,
		},
		{
			in:    "/AA/",
			out:   "/aa",
			found: true,
			slash: true,
		},
		{
			in:    "/aA",
			out:   "/aa",
			found: true,
			slash: false,
		},
		{
			in:    "/aA/",
			out:   "/aa",
			found: true,
			slash: true,
		},
		{
			in:    "/A/",
			out:   "/a/",
			found: true,
			slash: false,
		},
		{
			in:    "/A",
			out:   "/a/",
			found: true,
			slash: true,
		},
		{
			in:    "/DOC",
			out:   "/doc",
			found: true,
			slash: false,
		},
		{
			in:    "/DOC/",
			out:   "/doc",
			found: true,
			slash: true,
		},
		{
			in:    "/NO",
			out:   "",
			found: false,
			slash: true,
		},
		{
			in:    "/DOC/GO",
			out:   "",
			found: false,
			slash: true,
		},
	}

	for _, f := range fixture {
		route := f.in

		foundPath, found := test.findCaseInsensitivePath(route, true)

		var (
			expectedFound = f.found
			actualFound   = found
		)

		var (
			expectedPath = f.out
			actualPath   = foundPath
		)

		if expectedFound != actualFound {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%t', actual: '%t', route: '%s'",
				expectedFound,
				actualFound,
				route,
			)
			return
		}

		if actualFound && expectedPath != actualPath {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%s', actual: '%s', route: '%s'",
				expectedPath,
				actualPath,
				route,
			)
			return
		}
	}

	for _, f := range fixture {
		route := f.in

		foundPath, found := test.findCaseInsensitivePath(route, false)

		var (
			expectedFound = f.found
			actualFound   = found
		)

		var (
			expectedPath = f.out
			actualPath   = foundPath
		)

		if f.slash {
			if actualFound {
				t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%s', actual: '%s', route: '%s'",
					f.in,
					foundPath,
					route)
			}
			continue
		}

		if expectedFound != actualFound {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%t', actual: '%t', route: '%s'",
				expectedFound,
				actualFound,
				route,
			)
			return
		}

		if actualFound && expectedPath != actualPath {
			t.Errorf("[findCaseInsensitivePath] invalid result, expected: '%s', actual: '%s', route: '%s'",
				expectedPath, actualPath, route)
			return
		}
	}
}

func TestTree_InvalidNodeType(t *testing.T) {
	const expectedPanicMsgPrefix = "invalid node type"

	test := newTreeRoutingTest(t)

	test.addRoute("/", "/")
	test.addRoute("/:page", "/:page")

	test.tree.children[0].nodeType = 42

	route := "/test"

	recv := catchPanic(func() {
		test.resolvePath(route)
	})

	actualPanicMsg, panicked := recv.(string)

	if panicked == false {
		t.Fatalf("[addRoute] expected panic, none occurred, route: '%s'", route)
	}

	if strings.HasPrefix(actualPanicMsg, expectedPanicMsgPrefix) == false {
		t.Fatalf("[addRoute] unexpected panic message prefix, expected: '%s', actual: '%s', route: '%s'", expectedPanicMsgPrefix, actualPanicMsg, route)
	}

	recv = catchPanic(func() {
		test.findCaseInsensitivePath(route, true)
	})
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Type structure for facilitating various tests
///////////////////////////////////////////////////////////////////////////////////////////////////

type treeRoutingTest struct {
	handlerInvokedWithPath string
	testing                *testing.T
	tree                   *node
}

func newTreeRoutingTest(testing *testing.T) *treeRoutingTest {
	return &treeRoutingTest{
		testing: testing,
		tree:    &node{},
	}
}

func (r *treeRoutingTest) addRoute(path string, handlerInvokedWithPath string) {
	if handlerInvokedWithPath == empty {
		r.tree.AddRoute(path, nil)
		return
	}

	handler := func() http.HandlerFunc {
		return func(http.ResponseWriter, *http.Request) {
			r.handlerInvokedWithPath = handlerInvokedWithPath
		}
	}
	r.tree.AddRoute(path, handler())
}

func (r *treeRoutingTest) resolvePath(path string) (http.Handler, *PathParameters, bool) {
	return r.tree.Resolve(path)
}

func (r *treeRoutingTest) findCaseInsensitivePath(path string, fixTrailingSlash bool) (string, bool) {
	return r.tree.findCaseInsensitivePath(path, fixTrailingSlash)
}

func (r *treeRoutingTest) assertResolutions(fixture requestRoutingFixture) {
	for _, f := range fixture {
		route := f.path

		handler, parameters, _ := r.tree.Resolve(route)

		var (
			expectedNilHandler = f.nilHandler
			actualNilHandler   = handler == nil
		)

		var (
			actualHandler = handler
		)

		if expectedNilHandler != actualNilHandler {
			r.testing.Errorf("[resolvePath] invalid result: expected: %t, actual: %t, route:'%s'",
				expectedNilHandler,
				actualNilHandler,
				route,
			)
			continue
		}

		if actualHandler == nil {
			continue
		}

		handler.ServeHTTP(nil, nil)

		var (
			expectedHandlerPath = f.route
			actualHandlerPath   = r.handlerInvokedWithPath
		)

		if expectedHandlerPath != actualHandlerPath {
			r.testing.Errorf("[handler] invalid result: expected: '%s', actual: '%s', route: '%s'",
				expectedHandlerPath,
				actualHandlerPath,
				route,
			)
		}

		var (
			expectedParameters = f.parameters
			actualParameters   = parameters
		)

		if !reflect.DeepEqual(expectedParameters, actualParameters) {
			r.testing.Errorf("[resolvePath] invalid result, expected: '%+v', actual: '%+v', route: '%s'",
				expectedParameters,
				actualParameters,
				route,
			)
		}
	}
}

func (r *treeRoutingTest) assertRouteConflicts(fixture []pathConflictTestFixture) {
	for _, f := range fixture {
		recv := catchPanic(func() {
			r.addRoute(f.path, empty)
		})

		if f.conflict {
			if recv == nil {
				r.testing.Errorf("[addRoute] no panic for conflicting route '%s'", f.path)
			}
			continue
		}

		if recv != nil {
			r.testing.Errorf("[addRoute] unexpected panic for route '%s': %v", f.path, recv)
		}
	}
}

func (r *treeRoutingTest) assertNodeMaxParameters() {
	r.assertNodeMaxParametersRecursive(r.tree)
}

func (r *treeRoutingTest) assertNodeMaxParametersRecursive(n *node) uint8 {
	var maxParameters uint8

	for index := range n.children {
		parameters := r.assertNodeMaxParametersRecursive(n.children[index])
		if parameters > maxParameters {
			maxParameters = parameters
		}
	}

	if n.nodeType != static && !n.wildChild {
		maxParameters++
	}

	if n.maxParameters != maxParameters {
		r.testing.Errorf("inconsistent node maximum parameters, expected: %d, actual: %d, path: '%s'",
			n.maxParameters,
			maxParameters,
			n.path,
		)
	}

	return maxParameters
}

func (r *treeRoutingTest) assertNodePriorities() {
	r.assertNodePrioritiesRecursive(r.tree)
}

func (r *treeRoutingTest) assertNodePrioritiesRecursive(n *node) uint32 {
	var priority uint32

	for i := range n.children {
		priority += r.assertNodePrioritiesRecursive(n.children[i])
	}

	if n.handler != nil {
		priority++
	}

	if n.priority != priority {
		r.testing.Errorf("inconsistent node priority, expected: %d, actual: %d, path: '%s'",
			n.priority,
			priority,
			n.path,
		)
	}

	return priority
}
