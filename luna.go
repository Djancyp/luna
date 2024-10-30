package luna

import (
	"fmt"
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
		html, err := pkg.CreateTemplate(
			pkg.CreateTemplateData{
				Title:           "Test Route",
				Description:     "Test Route Description",
				CssLinks:        []string{},
				JsLinks:         []string{},
				RenderedContent: "Test Route",
				JS:              "",
				CSS:             "",
			},
		)
		if err != nil {
			return err
		}
		return html.Execute(c.Response().Writer, nil)
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
