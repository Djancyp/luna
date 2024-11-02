package luna

import (
	"text/template"

	"github.com/Djancyp/luna/pkg"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type PropsResponse struct {
	Path string `json:"path"`
}
type Engine struct {
	Logger    zerolog.Logger
	Server    *echo.Echo
	Config    Config
	Cache     []Cache
	HotReload *HotReload
}

type Cache struct {
	ID   string
	Path string
	HTML *template.Template
	Body string
	CSS  string
	JS   string
}

type Config struct {
	ENV                 string `default:"development"`
	RootPath            string `default:"frontend/"`
	ServerEntryPoint    string `default:"frontend/src/entry-client.tsx"`
	ClientEntryPoint    string `default:"frontend/src/entry-server.tsx"`
  FaviconPath         string `default:"frontend/src/assets/favicon.ico"`
	AssetsPath          string `default:"frontend/src/assets/"`
	PublicPath          string `default:"public/"`
	TailwindCSS         bool   `default:"false"`
	HotReloadServerPort int    `default:"8080"`
	Routes              []pkg.ReactRoute
}
