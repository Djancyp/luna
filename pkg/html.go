package pkg

import (
	"bytes"
	"html/template"
)

const templateHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    {{if .Description}}
    <meta name="description" content={{ .Description }} />
    {{end}}
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    {{if .Title}}
    <title>{{.Title}}</title>
    {{end}}
    {{ range .CssLinks }}
    <link href="{{.Href}}" rel="stylesheet" {{ range $key, $value := .DynamicAttrs }}{{ $key }}="{{ $value }}"{{ end }} />
    {{ end }}
    {{ range .JsLinks }}
    <script src="{{.Src}}" {{ range $key, $value := .DynamicAttrs }}{{ $key }}="{{ $value }}"{{ end }}></script>
    {{ end }}
    {{ if .CSS }}
    <style>
    {{ .CSS }}
    </style>
    {{ end }}
  </head>
  <body>
    <div id="root">{{ .RenderedContent }}</div>
    <script type="module">
      {{ .JS }}
    </script>
  </body>
</html>
`

func GetHTML() (*template.Template, error) {
	templ, err := template.New("html").Parse(templateHTML)
	if err != nil {
		return nil, err
	}
	return templ, nil
}

type CreateTemplateData struct {
	Title           string
	Description     string
	CssLinks        []string
	JsLinks         []string
	CSS             string
	JS              string
	RenderedContent string
}

func CreateTemplate(data CreateTemplateData) (*template.Template, error) {

	// Parse the base template
	tmpl, err := GetHTML()
	if err != nil {
		return nil, err
	}

	// Apply the data to the base template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	// Re-parse for additional sections
	baseTemplate, err := template.New("html").Parse(buf.String())
	if err != nil {
		return nil, err
	}

	return baseTemplate, nil

}
