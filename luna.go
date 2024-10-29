package luna

import (
	"net/http"
	"os"

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
	return &Engine{
		Logger: zerolog.New(os.Stdout).With().Timestamp().Logger(),
		Server: server,
		Config: config,
	}, nil
}

func (e *Engine) CheckApp(config Config) {
	err := utils.IsFolderExist(config.AssetsPath)
	if err != nil {
		e.Logger.Error().Msgf("Assets folder not found: %s", config.AssetsPath)
		panic(err)
	}

	err = utils.IsFileExist(config.EnteryPoint)
	if err != nil {
		e.Logger.Error().Msgf("EnteryPoint file not found: %s", config.EnteryPoint)
		panic(err)
	}
}

func (e *Engine) Get(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodGet, path, h, m...)
}

func (e *Engine) Post(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPost, path, h, m...)
}

func (e *Engine) Delete(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodDelete, path, h, m...)
}

func (e *Engine) Put(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPut, path, h, m...)
}

func (e *Engine) Patch(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.Server.Add(http.MethodPatch, path, h, m...)
}
func (e *Engine) Start(address string) {
	e.Server.Logger.Fatal(e.Server.Start(address))
}
