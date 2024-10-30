package luna

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Djancyp/luna/pkg"
	"github.com/Djancyp/luna/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func New(config Config) (*Engine, error) {

	server := echo.New()
	server.Static("/assets", config.AssetsPath)
	// make static public
	server.Use(middleware.CORS())
	server.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	server.POST("/props", func(c echo.Context) error {
		body := PropsResponse{}
		if err := c.Bind(&body); err != nil {
			return err
		}
		to := body.Path
		props := make(map[string]interface{})
		for _, route := range config.Routes {
			if route.Path == to && route.Props != nil {
				props[to] = route.Props()
			}
		}
		return c.JSON(200, props)

	})

	app := &Engine{
		Logger: zerolog.New(os.Stdout).With().Timestamp().Logger(),
		Server: server,
		Config: config,
	}

	app.HotReload = newHotReload(app)
	app.HotReload.Start()
	app.CheckApp(config)
	return app, nil
}

func (e *Engine) CheckApp(config Config) error {
	err := utils.IsFolderExist(config.AssetsPath)
	if err != nil {
		e.Logger.Error().Msgf("Assets folder not found: %s", config.AssetsPath)
		// stop the app no panic
		os.Exit(1)
	}

	err = utils.IsFileExist(config.EnteryPoint)
	if err != nil {
		e.Logger.Error().Msgf("EnteryPoint file not found: %s", config.EnteryPoint)
		os.Exit(1)
	}
	if config.ENV != "production" {
		for _, route := range config.Routes {
			for _, css := range *&route.Head.CssLinks {
				err = utils.IsFileExist(fmt.Sprintf("%s/%s", config.AssetsPath, css.Href))
				if err != nil {
					e.Logger.Error().Msgf("Css file not found: %s", config.AssetsPath+css.Href)
					os.Exit(1)
				}
			}
			for _, js := range *&route.Head.JsLinks {
				err = utils.IsFileExist(fmt.Sprintf("%s/%s", config.AssetsPath, js.Src))
				if err != nil {
					e.Logger.Error().Msgf("Js file not found: %s", config.AssetsPath+js.Src)
					os.Exit(1)
				}
			}
		}
	}

	return nil
}

func (e *Engine) InitializeFrontend() error {
	// Serve static files from the "frontend/public" directory
	// Catch-all route for serving dynamic content
	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path

		// Skip static assets (e.g., images, CSS, JS files)
		if filepath.Ext(path) != "" {
			// Let the static file handler handle this request
			return c.File(filepath.Join(e.Config.PublicPath, path))
		}

		var html *template.Template
		params := pkg.CreateTemplateData{}

		// Loop through routes to find a matching path
		for _, route := range e.Config.Routes {
			if route.Path == path {
				// Get route parameters and properties
				var props map[string]interface{}
				if matched, routeParams := pkg.MatchPath(route.Path, path); matched && route.Props != nil {
					if routeParams != nil {
						props = route.Props(routeParams)
					} else {
						props = route.Props()
					}
				} else {
					props = map[string]interface{}{}
				}

				// Build client and server assets
				client, err := pkg.BuildClient(e.Config.ENV)
				if err != nil {
					e.Logger.Error().Msgf("Error building client: %s", err)
					return c.String(http.StatusInternalServerError, "Error building client")
				}

				server, err := pkg.BuildServer(route.Path, props, e.Config.ENV)
				if err != nil {
					e.Logger.Error().Msgf("Error building server: %s", err)
					return c.String(http.StatusInternalServerError, "Error building server")
				}

				// Render server and client HTML
				serverHTML, err := pkg.RenderServer(server.JS, route.Path)
				if err != nil {
					e.Logger.Error().Msgf("Error rendering server HTML: %s", err)
					return c.String(http.StatusInternalServerError, "Error rendering server HTML")
				}
				clientHTML, err := pkg.RenderClientWithProps(client.JS, props, route.Path)
				if err != nil {
					e.Logger.Error().Msgf("Error rendering client HTML: %s", err)
					return c.String(http.StatusInternalServerError, "Error rendering client HTML")
				}

				// Generate CSS links
				var cssLinks []template.HTML
				for _, css := range route.Head.CssLinks {
					cssLink := template.HTML(fmt.Sprintf("<link href=\"/assets/%s\" rel=\"stylesheet\" />", css.Href))
					cssLinks = append(cssLinks, cssLink)
				}

				// Load the HTML template
				html, err = pkg.GetHTML()
				if err != nil {
					e.Logger.Error().Msgf("Template loading error: %s", err)
					return c.String(http.StatusInternalServerError, "Error loading template")
				}

				// Prepare template data
				params = pkg.CreateTemplateData{
					Title:           route.Head.Title,
					Description:     route.Head.Description,
					CssLinks:        cssLinks,
					JsLinks:         []template.HTML{}, // Populate with JS links if needed
					RenderedContent: template.HTML(serverHTML),
					JS:              template.JS(clientHTML),
					CSS:             template.CSS(server.CSS),
				}
				break
			}
		}

		// Check if a matching route was found and HTML template is initialized
		if html == nil {
			e.Logger.Warn().Msgf("No matching route found for: %s", path)
			return c.String(http.StatusNotFound, "Page not found")
		}

		// Render the template with parameters
		if err := html.Execute(c.Response().Writer, params); err != nil {
			e.Logger.Error().Msgf("Template execution error: %s", err)
			return c.String(http.StatusInternalServerError, "Error rendering page")
		}

		return nil
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

func (e *Engine) Use(middleware ...echo.MiddlewareFunc) {
	e.Server.Use(middleware...)
}
