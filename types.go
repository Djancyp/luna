package luna

import (
	"text/template"

	"github.com/Djancyp/luna/internal"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type PropsResponse struct {
	Path string `json:"path"`
}
type Engine struct {
	Logger   zerolog.Logger
	Server   *echo.Echo
	Config   Config
	Cache    []Cache
	Template internal.Template
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
	ENV         string `default:"development"`
	EnteryPoint string `default:"frontend/src/entry-client.tsx"`
	AssetsPath  string `default:"frontend/src/assets/"`
	TailwindCSS bool   `default:"false"`
	Routes      []internal.ReactRoute
}
