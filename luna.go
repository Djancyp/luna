package luna

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Djancyp/luna/pkg"
	"github.com/Djancyp/luna/utils"
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
			if route.Path == to {
				p := make(map[string]interface{})
				if route.Props != nil {
					p = route.Props()
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
		Logger: zerolog.New(os.Stdout).With().Timestamp().Logger(),
		Server: server,
		Config: config,
	}

	if config.ENV != "production" {
		app.HotReload = newHotReload(app)
		app.HotReload.Start(config.RootPath)
	}

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

	err = utils.IsFileExist(config.ServerEntryPoint)
	if err != nil {
		e.Logger.Error().Msgf("EnteryPoint file not found: %s", config.ServerEntryPoint)
		os.Exit(1)
	}
	err = utils.IsFileExist(config.ClientEntryPoint)
	if err != nil {
		e.Logger.Error().Msgf("EnteryPoint file not found: %s", config.ClientEntryPoint)
		os.Exit(1)
	}
	if config.TailwindCSS != false {
		err = utils.IsFileExist(config.RootPath + "tailwind.config.js")
		if err != nil {
			e.Logger.Error().Msgf("TailwindCSS file not found: tailwind.config.js")
			os.Exit(1)
		}
	}

	if config.ENV != "production" {
		for _, route := range config.Routes {
			for _, css := range *&route.Head.CssLinks {
				if !strings.Contains(css.Href, "https") {

					err = utils.IsFileExist(fmt.Sprintf("%s/%s", config.AssetsPath, css.Href))
					if err != nil {
						e.Logger.Error().Msgf("Css file not found: %s", config.AssetsPath+css.Href)
						os.Exit(1)
					}
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
	// Initialize Tailwind if configured
	var tailwindCSS string
	if e.Config.TailwindCSS {
		tailwindCSS = pkg.Tailwind(e.Config.RootPath)
	}

	job := pkg.JobRunner{
		ServerEntryPoint: e.Config.ServerEntryPoint,
		ClientEntryPoint: e.Config.ClientEntryPoint,
		Env:              e.Config.ENV,
	}
	manager := pkg.NewManager()

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
		protocol := "ws"
		if c.Request().TLS != nil {
			protocol = "wss" // Use wss if the page is served over HTTPS
		}
		swUrl := fmt.Sprintf("%s://%s:%d/ws", protocol, baseURL, e.Config.HotReloadServerPort)
		fmt.Println("swUrl", swUrl)

		// Check for cached page if in production mode
		if cachedItem, found := manager.GetCache(path); found && e.Config.ENV == "production" {
			return cachedItem.HTML.Execute(c.Response().Writer, pkg.CreateTemplateData{
				Title:           cachedItem.Title,
				Description:     cachedItem.Description,
				Favicon:         cachedItem.Favicon,
				CssLinks:        cachedItem.CSSLinks,
				RenderedContent: template.HTML(cachedItem.Body),
				JS:              template.JS(cachedItem.JS),
				CSS:             template.CSS(cachedItem.CSS),
				Dev:             e.Config.ENV != "production",
				SWUrl:           swUrl,
			})
		}

		// Route matching and template rendering
		for _, route := range e.Config.Routes {
			if route.Path != path {
				continue
			}
			handler := func(c echo.Context) error {
				var props map[string]interface{}
				if matched, routeParams := pkg.MatchPath(route.Path, path); matched && route.Props != nil {
					props = route.Props(routeParams)
				} else {
					props = map[string]interface{}{}
				}

				var client, server pkg.BuildResult
				var buildClientErr, buildServerErr error

				g, _ := errgroup.WithContext(context.Background())

				g.Go(func() error {
					client, buildClientErr = job.BuildClient(props, store)
					return buildClientErr
				})

				g.Go(func() error {
					server, buildServerErr = job.BuildServer(route.Path, props, store)
					return buildServerErr
				})

				// Wait for both functions to complete
				if err := g.Wait(); err != nil {
					if buildClientErr != nil {
						e.Logger.Error().Msgf("Error building client: %s", buildClientErr)
						return c.String(http.StatusInternalServerError, "Error building client")
					}
					if buildServerErr != nil {
						e.Logger.Error().Msgf("Error building server: %s", buildServerErr)
						return c.String(http.StatusInternalServerError, "Error building server")
					}
				}
				server.CSS = fmt.Sprintf("%s\n%s", server.CSS, tailwindCSS)

				serverHTML, err := pkg.RenderServer(server.JS, route.Path)
				if err != nil {
					e.Logger.Error().Msgf("Error rendering server HTML: %s", err)
					return c.String(http.StatusInternalServerError, "Error rendering server HTML")
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

				cacheData := pkg.Cache{
					ID:          path,
					Title:       route.Head.Title,
					Description: route.Head.Description,
					Favicon:     e.Config.FaviconPath,
					Path:        path,
					HTML:        htmlTemplate,
					Body:        serverHTML,
					CSS:         server.CSS,
					JS:          client.JS,
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
					RenderedContent: template.HTML(serverHTML),
					JS:              template.JS(client.JS),
					CSS:             template.CSS(server.CSS),
					Dev:             e.Config.ENV != "production",
					SWUrl:           swUrl,
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
		}

		e.Logger.Warn().Msgf("No matching route found for: %s", path)
		return c.String(http.StatusNotFound, "Page not found")
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
