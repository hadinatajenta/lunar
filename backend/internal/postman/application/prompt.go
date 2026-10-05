package application

import (
	"fmt"
	"strings"

	"lunar/backend/internal/postman/domain"
)

const SystemPromptText = `You extract HTTP API endpoints from a source repository and emit one Postman Collection v2.1 document describing them.

You will receive a file manifest and the content of selected files. Read them together: a route is often declared in one file and implemented in another.

Output contract:
1. Emit JSON only. No prose, no Markdown fence, no explanation.
2. Emit one Postman Collection v2.1 object with info, item, and variable.
3. Emit one request per endpoint, carrying method, url with :param placeholders, query parameters, headers, and the request body when a handler reads one.
4. Completeness is mandatory. For every key you identify, emit the key even when you do not know its value. An unknown value is an empty string. NEVER omit a key you identified.
5. Group requests by resource or controller into folders.
6. Never invent an endpoint you did not see. A route whose handler you cannot find is still emitted, with the description "handler not found".
7. Use {{baseUrl}} as the host variable.
8. Keep the description to one line, naming the source file.`

const vendorRule = `Selected files: files under src/vendor are first-party service code and must be read; files under any other vendor directory, node_modules, or a framework bundle must be ignored.`

const wireGuidance = `Wire: read wire_gen.go for the handlers that are constructed and passed to the router; wire.go only declares providers and does not show which constructor runs.`

var frameworkAppendices = map[domain.Framework]string{
	domain.FrameworkGoNetHTTP: `Framework appendix, Go net/http: routes are registered with http.HandleFunc, http.Handle, mux.HandleFunc, mux.Handle, and method-aware patterns such as "GET /path". Include every registered pattern, including the ones registered on a mux built inside a constructor.`,
	domain.FrameworkGin:       `Framework appendix, Gin: every route is registered as route.METHOD("/path", handler) on an engine or on a group. Resolve the r.Group prefixes INCLUDING NESTED GROUPS, combined with each route.METHOD path. atlas declares r.Group("/v1/user") and then route.GET("/level", ...); aurora nests route.Group("/program-mdr") inside r.Group("/v1").`,
	domain.FrameworkFiber:     `Framework appendix, Fiber: routes are app.Get, app.Post, and the other method helpers, prefixes come from Group, and app.Use registers middleware that may mount a sub application.`,
	domain.FrameworkEcho:      `Framework appendix, Echo: routes are e.GET, e.POST, and the other method helpers, prefixes come from e.Group, and each handler binds its own request target.`,
	domain.FrameworkPHPPlain:  `Framework appendix, PHP without a framework: requests arrive through the front controller and read $_GET and $_POST, so derive the method and the parameters from the front-controller routing table and the handler body.`,
	domain.FrameworkLaravel:   `Framework appendix, Laravel: read routes/api.php and routes/web.php, expand Route::apiResource and Route::resource into their REST actions, follow controller actions to their methods, and use the form requests to describe the request body.`,
	domain.FrameworkLumen:     `Framework appendix, Lumen: read routes/web.php only, because Lumen has no api.php split, and expand each $router->get and $router->post registration together with its controller action.`,
	domain.FrameworkNode:      `Framework appendix, Node.js: read the Express app.get and app.post registrations, follow router mounts such as app.use("/users", usersRouter) to the router file, and treat Koa and Fastify equivalents the same way.`,
}

func FrameworkAppendix(framework domain.Framework, usesWire bool) string {
	appendix := frameworkAppendices[framework]
	if appendix == "" {
		appendix = wireGuidance
	} else if usesWire {
		appendix = appendix + "\n\n" + wireGuidance
	}
	return appendix
}

func SystemPrompt(classification domain.Classification) string {
	return strings.Join([]string{
		SystemPromptText,
		FrameworkAppendix(classification.Framework, classification.UsesWire),
		vendorRule,
	}, "\n\n")
}

func UserPrompt(service string, selection domain.Selection) string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "Service: %s\n", service)
	fmt.Fprintf(&builder, "Selected files: %d\n", len(selection.Files))
	fmt.Fprintf(&builder, "Total bytes: %d\n", selection.TotalBytes)
	builder.WriteString("\nFile manifest:\n")
	for _, file := range selection.Files {
		fmt.Fprintf(&builder, "- %s (%d bytes)\n", file.RelativePath, file.SizeBytes)
	}
	builder.WriteString("\nFile contents:\n")
	for _, file := range selection.Files {
		fmt.Fprintf(&builder, "\n===== %s =====\n%s\n", file.RelativePath, file.Content)
	}
	return builder.String()
}
