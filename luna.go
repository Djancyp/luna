package luna

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/Djancyp/luna/pkg"
	"github.com/Djancyp/luna/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func New(config Config) (*Engine, error) {

	server := echo.New()
	server.Static("/assets", config.AssetsPath)
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
			if route.Path == to {
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

func (e *Engine) InitilizeFrontend() error {
	e.GET("/*", func(c echo.Context) error {
		var html *template.Template
		params := pkg.CreateTemplateData{}

		// Loop through routes to find a matching path
		for _, route := range e.Config.Routes {
			if route.Path == c.Request().URL.Path {
				var p map[string]interface{}
				// get the promps
				if matched, params := pkg.MatchPath(route.Path, c.Request().URL.Path); matched {
					if params != nil {
						p = route.Props(params)
					} else {
						p = route.Props()
					}
				}
				client, err := pkg.BuildClient(e.Config.ENV)
				if err != nil {
					e.Logger.Error().Msgf("Error building client: %s", err)
					return c.String(http.StatusInternalServerError, "Error building client")
				}

				server, err := pkg.BuildServer(route.Path, p, e.Config.ENV)
				if err != nil {
					e.Logger.Error().Msgf("Error building server: %s", err)
					return c.String(http.StatusInternalServerError, "Error building server")
				}

				ServerHtml, err := pkg.RenderServer(server.JS, route.Path)
				ClientHtml, err := pkg.RenderClientWithProps(client.JS, p, route.Path)

				// Generate CSS links as template.HTML to avoid HTML escaping
				var links []template.HTML
				for _, css := range route.Head.CssLinks {
					cssLink := template.HTML(fmt.Sprintf("<link href=\"/assets/%s\" rel=\"stylesheet\" />", css.Href))
					links = append(links, cssLink)
				}

				// Load the template
				html, err = pkg.GetHTML()
				if err != nil {
					fmt.Println("Template loading error:", err)
					return c.String(http.StatusInternalServerError, "Error loading template")
				}

				// Prepare data for the template
				params = pkg.CreateTemplateData{
					Title:           route.Head.Title,
					Description:     route.Head.Description,
					CssLinks:        links,
					JsLinks:         []template.HTML{}, // Populate with JS links if needed
					RenderedContent: template.HTML(ServerHtml),
					JS:              template.JS(ClientHtml),
					CSS:             template.CSS(server.CSS),
				}
				break
			}
		}

		// Check if a matching route was found and html template is initialized
		if html == nil {
			fmt.Println("Page not found")
			return echo.NewHTTPError(http.StatusNotFound, "Page not found")
		}

		// Render the template with parameters
		if err := html.Execute(c.Response().Writer, params); err != nil {
			fmt.Println("Template execution error:", err)
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
