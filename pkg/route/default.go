package route

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/cloudretic/router/pkg/middleware"
	"github.com/cloudretic/router/pkg/path"
)

//=====PARTS=====

// LITERALS

// Literal route Parts match and pass without additionally transforming
// the request.

// stringPart; literal string part
type stringPart struct {
	val string
}

func build_stringPart(val string) (*stringPart, error) {
	return &stringPart{val}, nil
}

func (part *stringPart) Match(ctx *routeMatchContext, token string) bool {
	if part.val == token {
		return true
	} else {
		return false
	}
}

// WILDCARDS

// Wildcard route Parts store parameters for use by the router in handlers.
// They use a syntax of [wildcard] to denote their name, and can additionally
// be qualified by some conditions by splitting with the : character.

// wildcardParts always match, and add the token as a request param.
type wildcardPart struct {
	param string
}

func build_wildcardPart(param string) (*wildcardPart, error) {
	return &wildcardPart{param}, nil
}

func (part *wildcardPart) Match(ctx *routeMatchContext, token string) bool {
	token = token[1:]
	if part.param != "" {
		setParam(ctx, part.param, token)
	}
	return true
}

func (part *wildcardPart) ParameterName() string {
	return part.param
}

func (part *wildcardPart) SetParameterName(s string) {
	part.param = s
}

// regexParts match against regular expressions.
// They're created using the syntax [wildcard]:{regex}
type regexPart struct {
	param string
	expr  *regexp.Regexp
}

func build_regexPart(param, expr string) (*regexPart, error) {
	expr_compiled, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	} else {
		return &regexPart{param, expr_compiled}, nil
	}
}

func (part *regexPart) Match(ctx *routeMatchContext, token string) bool {
	token = token[1:]
	// Match against regex
	matched := part.expr.FindString(token)
	if matched != token {
		return false
	}
	// If a parameter is set, act as a wildcard param.
	if part.param != "" {
		// If a token matched, store the matched value as a route Param
		setParam(ctx, part.param, token)
	}
	return true
}

func (part *regexPart) ParameterName() string {
	return part.param
}

func (part *regexPart) SetParameterName(s string) {
	part.param = s
}

// =====ROUTE=====

// defaultRoute is the default behavior for router, which is to match requests exactly.
type defaultRoute struct {
	origExpr string
	mws      []middleware.Middleware
	parts    []Part
	ctx      *routeMatchContext
}

// Tokenize and parse a route expression into a defaultRoute.
//
// See interface Route.
func build_defaultRoute(expr string) (*defaultRoute, error) {
	route := &defaultRoute{
		origExpr: expr,
		mws:      make([]middleware.Middleware, 0),
		parts:    make([]Part, 0),
		ctx:      newRMC(),
	}
	var token string
	for next := 0; next < len(expr); {
		token, next = path.Next(expr, next)
		part, err := parse(token)
		if err != nil {
			return nil, err
		}
		if pp, ok := part.(paramPart); ok {
			pn := pp.ParameterName()
			if pn != "" {
				route.ctx.Allocate(pn)
			}
		}
		route.parts = append(route.parts, part)
		if next == -1 {
			break
		}
	}
	return route, nil
}

// Get a string value unique to the route.
//
// See interface Route.
func (route *defaultRoute) Hash() string {
	return route.origExpr
}

// Get the length of the route.
// For defaultRoutes, this is the total number of Parts it contains.
//
// See interface Route.
func (route *defaultRoute) Length() int {
	return len(route.parts)
}

// Attach middleware to the route. Middleware is handled in attachment order.
//
// See interface Route.
func (route *defaultRoute) Attach(mw middleware.Middleware) {
	route.mws = append(route.mws, mw)
}

// Match a request and update its context.
//
// See interface Route.
func (route *defaultRoute) MatchAndUpdateContext(req *http.Request) *http.Request {
	route.ctx.ResetOnto(req.Context())
	// Check for path length
	expr := req.URL.Path
	if strings.Count(expr, "/") != len(route.parts) {
		return nil
	}
	// Run any attached middleware
	for _, mw := range route.mws {
		if req = mw(req); req == nil {
			return nil
		}
	}

	var token string
	var partIdx int
	for next := 0; next < len(expr); {
		part := route.parts[partIdx]
		token, next = path.Next(expr, next)
		if ok := part.Match(route.ctx, token); !ok {
			return nil
		}
		partIdx++
		if next == -1 {
			break
		}
	}
	return req.WithContext(route.ctx)
}