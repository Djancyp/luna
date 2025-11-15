package luna

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Djancyp/luna/pkg"
	"github.com/Djancyp/luna/utils"
	esbuildapi "github.com/evanw/esbuild/pkg/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type NavigateRequest struct {
	Path        string                 `json:"path"`
	Props       map[string]interface{} `json:"props"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
}

func New(config Config) (*Engine, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	server := echo.New()
	server.Static("/assets", config.AssetsPath)
	// make static public
	server.Use(middleware.CORS())
	server.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	server.POST("/navigate", func(c echo.Context) error {
		// check middleware
		body := PropsResponse{}
		if err := c.Bind(&body); err != nil {
			return err
		}
		to := body.Path
		props := make(map[string]interface{})
		res := NavigateRequest{}
		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, props)
		}
		for _, route := range config.Routes {
			matched, params := pkg.MatchPath(route.Path, to)
			if matched || route.Path == to {
				p := make(map[string]interface{})
				if route.Props != nil {
					p = route.Props(c, params)
				} else {
					p = make(map[string]interface{})
				}
				handler = func(c echo.Context) error {
					props[to] = p
					res.Path = to
					res.Props = props
					res.Title = route.Head.Title
					res.Description = route.Head.Description
					return nil
				}
				if route.Middleware != nil {
					for _, middleware := range route.Middleware {
						handler = middleware(handler) // Wrap the handler with each middleware
					}
				}
				handler(c)
			}
		}
		return c.JSON(http.StatusOK, res)
	})
	app := &Engine{
		Logger:       zerolog.New(os.Stdout).With().Timestamp().Logger(),
		Server:       server,
		Config:       config,
		CacheManager: pkg.NewManager(),
		RouteIndex:   NewRouteTrie(),
	}

	// Build the route index for efficient lookup
	for i := range config.Routes {
		app.RouteIndex.Insert(config.Routes[i].Path, &config.Routes[i])
	}

	if config.ENV != "production" {
		app.HotReload = newHotReload(app)
		app.HotReload.Start(config.RootPath)
	}
	app.CheckApp(config)
	return app, nil
}

func (e *Engine) CheckApp(config Config) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 10) // Buffered channel to collect errors

	// Helper function to handle file/folder existence checks and send errors
	checkExistence := func(path string, isFolder bool, errMsg string) {
		defer wg.Done()
		var err error
		if isFolder {
			err = utils.IsFolderExist(path)
		} else {
			err = utils.IsFileExist(path)
		}
		if err != nil {
			e.Logger.Error().Msgf(errMsg, path)
			errCh <- err // Send error to the channel
		}
	}

	// Check the assets folder
	wg.Add(1)
	go checkExistence(config.AssetsPath, true, "Assets folder not found: %s")

	// Check server and client entry points
	wg.Add(1)
	go checkExistence(config.ServerEntryPoint, false, "EntryPoint file not found: %s")
	wg.Add(1)
	go checkExistence(config.ClientEntryPoint, false, "EntryPoint file not found: %s")

	// Check TailwindCSS file if required
	if config.TailwindCSS {
		wg.Add(1)
		go checkExistence(config.RootPath+"tailwind.config.js", false, "TailwindCSS file not found: %s")
	}

	// Additional checks for routes in non-production environments
	if config.ENV != "production" {
		for _, route := range config.Routes {
			// Check CSS files
			for _, css := range route.Head.CssLinks {
				if !strings.Contains(css.Href, "https") {
					wg.Add(1)
					go checkExistence(config.AssetsPath+"/"+css.Href, false, "Css file not found: %s")
				}
			}
			// Check JS files
			for _, js := range route.Head.JsLinks {
				wg.Add(1)
				go checkExistence(config.AssetsPath+"/"+js.Src, false, "Js file not found: %s")
			}
		}
	}

	// Wait for all checks to finish
	wg.Wait()
	close(errCh)

	// Collect errors, if any
	for err := range errCh {
		if err != nil {
			return err // Return the first error encountered
		}
	}

	return nil
}

func (e *Engine) InitializeFrontend() error {
	// Initialize Tailwind if configured
	var tailwindCSS string
	if e.Config.TailwindCSS {
		css, err := pkg.Tailwind(e.Config.RootPath)
		if err != nil {
			e.Logger.Error().Msgf("Failed to generate Tailwind CSS: %v", err)
			// Continue without Tailwind CSS to avoid crashing the application
			tailwindCSS = ""
		} else {
			tailwindCSS = css
		}
	}

	job := pkg.JobRunner{
		ServerEntryPoint: e.Config.ServerEntryPoint,
		ClientEntryPoint: e.Config.ClientEntryPoint,
		Env:              e.Config.ENV,
	}
	// Use the Engine's CacheManager
	manager := e.CacheManager

	var client, server pkg.BuildResult
	var buildClientErr, buildServerErr error
	g, _ := errgroup.WithContext(context.Background())

	if client.JS == "" || server.JS == "" {
		g.Go(func() error {
			client, buildClientErr = job.BuildClient()
			return buildClientErr
		})

		g.Go(func() error {
			server, buildServerErr = job.BuildServer()
			// create a file for server
			return buildServerErr
		})

		// Wait for both functions to complete
		if err := g.Wait(); err != nil {
			if buildClientErr != nil {
				e.Logger.Error().Msgf("Error building client: %s", buildClientErr)
			}
			if buildServerErr != nil {
				e.Logger.Error().Msgf("Error building server: %s", buildServerErr)
			}
		}
	}

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path

		// Serve static files directly
		if filepath.Ext(path) != "" {
			return c.File(filepath.Join(e.Config.PublicPath, path))
		}
		var store map[string]interface{}

		if e.Config.Store != nil {
			store = e.Config.Store(c)
		}
		baseURL := c.Request().Host
		// clean base url if has port
		baseURL = strings.Split(baseURL, ":")[0]

		// Determine WebSocket protocol based on request protocol
		protocol := "ws"
		if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
			protocol = "wss"
		}

		swUrl := fmt.Sprintf("%s://%s:%d/ws", protocol, baseURL, e.Config.HotReloadServerPort)

		// add main css or js
		// convert aattributes to html

		attributes := make([]template.HTML, len(e.Config.Head.Attributes))
		for i, attr := range e.Config.Head.Attributes {
			attributes[i] = template.HTML(attr)
		}

		// Route matching and template rendering using the route index for efficiency
		route, params, found := e.RouteIndex.Search(path)
		if !found {
			// Check for exact match (fallback for paths that don't use parameters)
			for _, r := range e.Config.Routes {
				if r.Path == path {
					route = &r
					params = make(map[string]string)
					found = true
					break
				}
			}
		}

		if found && route != nil {
			// check where does request come from
			handler := func(c echo.Context) error {
				var props map[string]interface{}
				if route.Props != nil {
					props = route.Props(c, params)
				} else {
					props = map[string]interface{}{}
				}

				if store == nil {
					store = map[string]interface{}{}
				}
				jsonProps, error := json.Marshal(props)
				if error != nil {
					return error
				}
				jsonStore, error := json.Marshal(store)
				if error != nil {
					return error
				}
				cjs := esbuildapi.Transform(client.JS, esbuildapi.TransformOptions{
					Define: map[string]string{
						"props":  string(jsonProps),
						"store":  string(jsonStore),
						"global": "globalThis",
					},
				})
				sjs := esbuildapi.Transform(server.JS, esbuildapi.TransformOptions{
					Define: map[string]string{
						"props":  string(jsonProps),
						"store":  string(jsonStore),
						"global": "globalThis",
					},
				})
				if tailwindCSS != "" {
					server.CSS = fmt.Sprintf("%s\n%s", server.CSS, tailwindCSS)
				}

				serverHTML, err := pkg.RenderServer(string(sjs.Code), route.Path)
				if err != nil {
					e.Logger.Error().Msgf("Error rendering server HTML: %s", err)
					return c.String(http.StatusInternalServerError, "Error rendering server HTML")
				}

				// Sanitize the rendered HTML content to prevent XSS
				sanitizedHTML := pkg.SanitizeRenderedContent(serverHTML)

				if cachedItem, found := manager.GetCache(path); found {
					// Check for cached page if in production mode
					// Combine cached CSS with Tailwind CSS
					combinedCSS := cachedItem.CSS
					if tailwindCSS != "" {
						combinedCSS = fmt.Sprintf("%s\n%s", combinedCSS, tailwindCSS)
					}
					return cachedItem.HTML.Execute(c.Response().Writer, pkg.CreateTemplateData{
						Title:           cachedItem.Title,
						Description:     cachedItem.Description,
						Favicon:         cachedItem.Favicon,
						CssLinks:        cachedItem.CSSLinks,
						RenderedContent: template.HTML(sanitizedHTML),
						JS:              template.JS(string(cjs.Code)),
						CSS:             template.CSS(combinedCSS),
						Dev:             e.Config.ENV != "production",
						SWUrl:           swUrl,
						MainHead:        attributes,
					})
				}
				// Collect CSS and JS links
				cssLinks := make([]template.HTML, len(route.Head.CssLinks))
				for i, css := range route.Head.CssLinks {
					// Check if css is a third-party link by looking for "https" in the URL
					if strings.Contains(css.Href, "https") {
						cssLinks[i] = template.HTML(fmt.Sprintf("<link href=\"%s\" rel=\"stylesheet\" />", css.Href))
					} else {
						cssLinks[i] = template.HTML(fmt.Sprintf("<link href=\"/assets/%s\" rel=\"stylesheet\" />", css.Href))
					}
				}
				jsLinks := make([]template.HTML, len(route.Head.JsLinks))
				for i, js := range route.Head.JsLinks {
					jsLinks[i] = template.HTML(fmt.Sprintf("<script src=\"/assets/%s\" type=\"module\"></script>", js.Src))
				}

				// Load HTML template once
				htmlTemplate, err := pkg.GetHTML()
				if err != nil {
					e.Logger.Error().Msgf("Template loading error: %s", err)
					return c.String(http.StatusInternalServerError, "Error loading template")
				}

				// Combine server CSS with Tailwind CSS for the cache
				combinedCSS := server.CSS
				if tailwindCSS != "" {
					combinedCSS = fmt.Sprintf("%s\n%s", combinedCSS, tailwindCSS)
				}

				cacheData := pkg.Cache{
					ID:          path,
					Title:       route.Head.Title,
					Description: route.Head.Description,
					Favicon:     e.Config.FaviconPath,
					Path:        path,
					HTML:        htmlTemplate,
					Body:        serverHTML,
					CSS:         combinedCSS,
					JS:          string(cjs.Code),
					CSSLinks:    cssLinks,
					Expiration:  route.CacheExpiry,
				}
				manager.AddCache(cacheData)

				// Render response with template data
				templateData := pkg.CreateTemplateData{
					Title:           route.Head.Title,
					Description:     route.Head.Description,
					Favicon:         e.Config.FaviconPath,
					CssLinks:        cssLinks,
					JsLinks:         jsLinks,
					RenderedContent: template.HTML(sanitizedHTML),
					JS:              template.JS(cjs.Code),
					CSS:             template.CSS(server.CSS),
					Dev:             false,
					SWUrl:           swUrl,
					MainHead:        attributes,
				}

				return htmlTemplate.Execute(c.Response().Writer, templateData)
			}

			if route.Middleware != nil {
				for _, middleware := range route.Middleware {
					handler = middleware(handler) // Wrap the handler with each middleware
				}
			}

			// Execute the handler with the middleware chain applied
			return handler(c)
		} else {
			e.Logger.Warn().Msgf("No matching route found for: %s", path)
			return c.String(http.StatusNotFound, "Page not found 1")
		}
	})

	return nil
}

func (e *Engine) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodGet, path, h, m...)
}

func (e *Engine) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPost, path, h, m...)
}

func (e *Engine) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodDelete, path, h, m...)
}

func (e *Engine) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPut, path, h, m...)
}

func (e *Engine) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPatch, path, h, m...)
}

func (e *Engine) Start(address string) {
	e.Server.Logger.Fatal(e.Server.Start(address))
}

func (e *Engine) Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group {
	return e.Server.Group(prefix, m...)
}

func (e *Engine) Static(prefix, root string) {
	e.Server.Static(prefix, root)
}

func (e *Engine) Close() {
	if e.HotReload != nil {
		e.HotReload.Stop()
	}
	if e.CacheManager != nil {
		e.CacheManager.Close()
	}
	// Add any other cleanup code here
}

func (e *Engine) Use(middleware ...echo.MiddlewareFunc) {
	e.Server.Use(middleware...)
}

func validateConfig(config Config) error {
	if config.RootPath == "" {
		return fmt.Errorf("RootPath is required")
	}

	if config.ServerEntryPoint == "" {
		return fmt.Errorf("ServerEntryPoint is required")
	}

	if config.ClientEntryPoint == "" {
		return fmt.Errorf("ClientEntryPoint is required")
	}

	if config.AssetsPath == "" {
		return fmt.Errorf("AssetsPath is required")
	}

	if config.PublicPath == "" {
		return fmt.Errorf("PublicPath is required")
	}

	if len(config.Routes) == 0 {
		return fmt.Errorf("At least one route must be configured")
	}

	// Validate each route
	for i, route := range config.Routes {
		if route.Path == "" {
			return fmt.Errorf("route at index %d: Path is required", i)
		}
		// Check if path is a valid URL pattern
		if !isValidRoutePath(route.Path) {
			return fmt.Errorf("route at index %d: invalid path pattern '%s'", i, route.Path)
		}
	}

	// We no longer validate Tailwind config here since it's handled in the Tailwind function during build

	return nil
}

// RoutePathComponent represents a path component (either a literal or parameter)
type RoutePathComponent struct {
	Type  string // "literal", "param", or "wildcard"
	Value string // The literal value or param name
}

// RouteTrieNode represents a node in the route trie
type RouteTrieNode struct {
	children       map[string]*RouteTrieNode
	paramChild     *RouteTrieNode  // For route parameters like :id
	paramName      string          // Name of the parameter (e.g., "id" for ":id")
	wildcardChild  *RouteTrieNode  // For wildcard routes like *path
	wildcardName   string          // Name of the wildcard parameter
	route          *pkg.ReactRoute // The route associated with this path (if this is a complete route)
	originalPath   string          // Original path pattern for parameter name resolution
	pathComponents []RoutePathComponent // Components of the original path
}

// RouteTrie is the main trie structure for route matching
type RouteTrie struct {
	root *RouteTrieNode
}

// NewRouteTrie creates a new route trie
func NewRouteTrie() *RouteTrie {
	return &RouteTrie{
		root: &RouteTrieNode{
			children: make(map[string]*RouteTrieNode),
		},
	}
}

// Insert adds a route to the trie
func (t *RouteTrie) Insert(path string, route *pkg.ReactRoute) {
	current := t.root

	// Parse the path into components and store parameter names
	components := parsePathComponents(path)

	// Store the original path and components for parameter name resolution
	currentPath := make([]RoutePathComponent, len(components))
	copy(currentPath, components)

	for i, component := range components {
		isLastComponent := i == len(components)-1

		if component.Type == "param" {
			if current.paramChild == nil {
				current.paramChild = &RouteTrieNode{
					children: make(map[string]*RouteTrieNode),
					paramName: component.Value,
				}
			}
			current = current.paramChild
		} else if component.Type == "wildcard" {
			if current.wildcardChild == nil {
				current.wildcardChild = &RouteTrieNode{
					children: make(map[string]*RouteTrieNode),
					wildcardName: component.Value,
				}
			}
			current = current.wildcardChild
		} else { // literal
			if _, exists := current.children[component.Value]; !exists {
				current.children[component.Value] = &RouteTrieNode{
					children: make(map[string]*RouteTrieNode),
				}
			}
			current = current.children[component.Value]
		}

		if isLastComponent {
			current.route = route
			current.originalPath = path
			current.pathComponents = currentPath
		}
	}
}

// parsePathComponents parses a path string like "/users/:id/posts/*rest" into individual components
func parsePathComponents(path string) []RoutePathComponent {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	components := make([]RoutePathComponent, 0, len(segments))

	for _, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			components = append(components, RoutePathComponent{
				Type:  "param",
				Value: strings.TrimPrefix(segment, ":"),
			})
		} else if segment == "*" {
			components = append(components, RoutePathComponent{
				Type:  "wildcard",
				Value: "path", // default name for wildcard parameter
			})
		} else {
			components = append(components, RoutePathComponent{
				Type:  "literal",
				Value: segment,
			})
		}
	}

	return components
}

// Search finds a route that matches the given path and extracts parameters
func (t *RouteTrie) Search(path string) (*pkg.ReactRoute, map[string]string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")

	route, params, found := t.searchNode(t.root, segments, make(map[string]string), 0)

	if found {
		return route, params, true
	}

	return nil, nil, false
}

// searchNode recursively searches the trie for a matching route
func (t *RouteTrie) searchNode(node *RouteTrieNode, segments []string, params map[string]string, componentIndex int) (*pkg.ReactRoute, map[string]string, bool) {
	if len(segments) == 0 {
		if node.route != nil {
			return node.route, params, true
		}
		return nil, nil, false
	}

	currentSegment := segments[0]
	remainingSegments := segments[1:]

	// First try exact match
	if child, exists := node.children[currentSegment]; exists {
		route, newParams, found := t.searchNode(child, remainingSegments, params, componentIndex+1)
		if found {
			return route, newParams, true
		}
	}

	// Then try parameter match
	if node.paramChild != nil && node.paramName != "" {
		newParams := make(map[string]string)
		for k, v := range params {
			newParams[k] = v
		}
		// Add the parameter value using the parameter name from the original path
		newParams[node.paramName] = currentSegment
		route, finalParams, found := t.searchNode(node.paramChild, remainingSegments, newParams, componentIndex+1)
		if found {
			return route, finalParams, true
		}
	}

	// Then try wildcard match
	if node.wildcardChild != nil && node.wildcardName != "" {
		newParams := make(map[string]string)
		for k, v := range params {
			newParams[k] = v
		}
		// For wildcard, join the remaining segments
		wildcardValue := strings.Join(remainingSegments, "/")
		newParams[node.wildcardName] = wildcardValue
		// Wildcard matches consume the rest, so pass empty segments
		route, finalParams, found := t.searchNode(node.wildcardChild, []string{}, newParams, componentIndex+1)
		if found {
			return route, finalParams, true
		}
	}

	return nil, nil, false
}

func isValidRoutePath(path string) bool {
	// Basic validation: path must start with '/' and contain valid characters
	if path == "" || path[0] != '/' {
		return false
	}

	// Check for valid characters in path
	for _, r := range path {
		if !isValidPathChar(r) {
			return false
		}
	}

	return true
}

func isValidPathChar(r rune) bool {
	// Valid path characters include letters, numbers, hyphens, underscores, dots, slashes, and colons for parameters
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
		   r == '/' || r == '-' || r == '_' || r == '.' || r == ':' || r == '*' || r == '(' || r == ')'
}

func mapToJSObject(data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}: // Handle nested maps
		var parts []string
		for key, value := range v {
			parts = append(parts, fmt.Sprintf("%s: %s", key, mapToJSObject(value)))
		}
		return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
	case string: // Handle strings with quotes
		return fmt.Sprintf("\"%s\"", v)
	default: // Handle other types (numbers, etc.)
		return fmt.Sprintf("%v", v)
	}
}
