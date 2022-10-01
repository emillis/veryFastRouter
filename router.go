package veryFastRouter

import (
	"fmt"
	"net/http"
)

//===========[CACHE/STATIC]====================================================================================================

//bufferSize is the maximum number of segments that the route can consist of. Increasing this doesn't appear
//to affect performance of the application, only it's memory footprint.
const bufferSize = 50

//Method defines http method (GET, POST, PUT, etc...)
type Method string

//HandlerFunc defines how a request handler should look like
type HandlerFunc func(http.ResponseWriter, *http.Request)

//Allowed method definitions
var (
	GET  Method = "GET"
	POST Method = "POST"
)

//AllMethods slice contains all available methods
var AllMethods = []Method{GET, POST}

//===========[STRUCTS]====================================================================================================

type pathDetails struct {
	count    int
	segments [bufferSize]string
}

type segment struct {
	value      string
	isVariable bool
	ok         bool
}

//HttpRouter implements Handler interface
type HttpRouter struct {
	//staticRoutes store all the routes that do not have variables in them
	staticRoutes map[string]*route

	//variableRoutes store all the routes that contain variables in them
	variableRoutes []*route

	//httpStatusCodeHandlers hold all the default/custom handlers to various http status codes
	httpStatusCodeHandlers httpStatusCodeHandlers
}

//HttpStatusCodeHandler allows you to set up custom handlers for various http status codes,
//e.g. 404, 405...
func (r *HttpRouter) HttpStatusCodeHandler(statusCode int, handler HandlerFunc) {
	//At first, checking whether the status code exist in the httpStatusCodeHandlers,
	//if not, it means that code is not supported
	if _, exist := r.httpStatusCodeHandlers.handlers[statusCode]; !exist {
		panic(fmt.Sprintf("status code \"%d\" is not supported!", statusCode))
	}

	if handler == nil {
		panic(fmt.Sprintf("handler is not defined for status code \"%d\"!", statusCode))
	}

	//Assigning newly supplied handler in the place of the default one. The purpose of the wrapper
	//is to write http status code by default, in case it's forgotten in the implementation supplied
	r.httpStatusCodeHandlers.handlers[statusCode] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		handler(w, r)
	}
}

//HandleFunc adds a new http request handler for the pattern defined. You can also define
//methods to which this handler is going to respond to. If nil is passed as methods, a default
//or a custom 405 handler will be invoked. For the handler to response to all methods, you
//should use in AllMethods that's defined in this module
func (r *HttpRouter) HandleFunc(pattern string, methods []Method, handler HandlerFunc) {
	route, err := r.addRoute(pattern)
	if err != nil {
		panic(err)
	}

	route.methods = methods
	if route.methods == nil || len(route.methods) == 0 {
		panic(fmt.Sprintf("method for pattern \"%s\" are not defined!", pattern))
	}

	route.handler = handler
	if route.handler == nil {
		panic(fmt.Sprintf("handler for pattern \"%s\" is not defined!", pattern))
	}
}

//findRoute returns pointer to route based on path supplied
func (r *HttpRouter) findRoute(path string) *route {
	path = processPath(path)

	if router, exist := r.staticRoutes[path]; exist {
		return router
	}

	pd := &pathDetails{
		count:    0,
		segments: [50]string{},
	}

	//Splitting the supplied path into its segments
	for i := len(path) - 1; i >= 0; i-- {
		//If the character is not "/", continue to the next character
		if path[i] != 47 {
			continue
		}

		pd.segments[pd.count] = path[i:]

		path = path[:i]
		pd.count++
	}

	for i := 0; i < len(r.variableRoutes); i++ {
		if !r.variableRoutes[i].compare(pd) {
			continue
		}

		return r.variableRoutes[i]
	}

	return nil
}

//addRoute parses pattern supplied and adds it to the HttpRouter
func (r *HttpRouter) addRoute(pattern string) (*route, error) {
	route, err := newRoute(pattern)
	if err != nil {
		return nil, err
	}

	if !route.hasVariables {
		r.staticRoutes[route.originalPattern] = route
		return route, nil
	}

	r.variableRoutes = append(r.variableRoutes, route)
	return route, nil
}

//ServerHTTP serves the requests
func (r *HttpRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//Looking for route withing the defined handlers
	route := r.findRoute(req.URL.Path)

	//This is where custom 404 handler can be established
	if route == nil {
		r.httpStatusCodeHandlers.handlers[http.StatusNotFound](w, req)
		return
	}

	//Checks whether the method of the request is allowed for this handler
	allowedMethod := false
	for i := 0; i < len(route.methods); i++ {
		if string(route.methods[i]) != req.Method {
			continue
		}

		allowedMethod = true

		break
	}

	if !allowedMethod {
		r.httpStatusCodeHandlers.handlers[http.StatusMethodNotAllowed](w, req)
		return
	}

	route.handler(w, req)
}

//===========[FUNCTIONALITY]====================================================================================================

//newSegment returns a new segment based on the string supplied
func newSegment(seg string) segment {
	return segment{
		value:      seg,
		isVariable: seg[1] == 58,
		ok:         true,
	}
}

//splitPath splits path and returns a slice of its segments
func splitPath(path string) []segment {
	var buffer []segment
	var j int

	for i := len(path) - 1; i >= 0; i-- {
		if path[i] != 47 {
			continue
		}

		buffer = append(buffer, newSegment(path[i:]))
		path = path[:i]
		i = len(path)

		j++
	}

	return buffer
}

//processPath check for critical errors within the path supplied. Also, removes trailing "/" sign if present
func processPath(s string) string {
	//if s == "" {
	//	return s, errors.New("path supplied cannot be an empty string")
	//}

	//if s[0] != 47 {
	//	return s, errors.New("path must begin with \"/\"")
	//}

	if s[len(s)-1] == 47 && len(s) > 1 {
		return s[:len(s)-1]
	}

	return s
}

//newRoute returns pointer to a new route created from path supplied
func newRoute(path string) (*route, error) {
	path = processPath(path)

	r := route{
		originalPattern: path,
		segments:        splitPath(path),
	}

	for _, segment := range r.segments {
		if !segment.isVariable {
			continue
		}

		r.hasVariables = true
		break
	}

	return &r, nil
}

//NewRouter crates a new instance of HttpRouter and returns pointer to it
func NewRouter() *HttpRouter {
	return &HttpRouter{
		staticRoutes:           map[string]*route{},
		variableRoutes:         []*route{},
		httpStatusCodeHandlers: newCustomHttpCodeHandlers(),
	}
}
