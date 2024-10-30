package internal

import "html/template"

type ReactRoute struct {
	Path     string
	MetaData MetaData
	Props    func() map[string]interface{}
}

type MetaData struct {
	Title       string
	Description string
	CssLinks    *[]CssLink
	JsLinks     *[]JsLink
	MetaTags    *[]MetaTag
}

type MetaTag struct {
	Name         string
	Content      string
	DynamicAttrs map[string]string
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
	HTML     *template.Template
	CSSLinks CssLink
	JSLinks  JsLink
}
