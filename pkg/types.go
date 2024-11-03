package pkg

import (
	"html/template"

	"github.com/labstack/echo/v4"
)

type ReactRoute struct {
	Path        string
	CacheExpiry int64
	Head        Head
	Props       func(params ...map[string]string) map[string]interface{}
	Middleware  []echo.MiddlewareFunc
}
type Store func() map[string]interface{}

type Head struct {
	Title       string
	Description string
	Favicon     Favicon
	CssLinks    []CssLink
	JsLinks     []JsLink
	MetaTags    []MetaTag
}

type MetaTag struct {
	Name         string
	Content      string
	DynamicAttrs map[string]string
}
type Favicon struct {
	Href string
	Type string
}

type CssLink struct {
	Href         string
	DynamicAttrs map[string]string
}
type JsLink struct {
	Src          string
	DynamicAttrs map[string]string
}
type Template struct {
	HTML *template.Template
}
