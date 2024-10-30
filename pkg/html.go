package pkg

import (
	"bytes"
	"html/template"
)

const templateHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="description" content={{.description}} />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{.Title}}</title>
    {{ range .CssLinks }}
    <link href="{{.Href}}" rel="stylesheet" {{ range $key, $value := .DynamicAttrs }}{{ $key }}="{{ $value }}"{{ end }} />
    {{ end }}
    {{ range .JsLinks }}
    <script src="{{.Src}}" {{ range $key, $value := .DynamicAttrs }}{{ $key }}="{{ $value }}"{{ end }}></script>
    {{ end }}
    <style>
    {{ .CSS }}
    </style>
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

func CreateTemplate(data Head) (*template.Template, error) {

	// add the CSS and JS links to the template templateHTML
	// return  *template.Template
	tmpl, err := GetHTML()
	if err != nil {
		return &template.Template{}, err
	}

	// Create a buffer to store the rendered HTML
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return &template.Template{}, err
	}
	temp, err := template.New("html").Parse(buf.String())
	if err != nil {
		return &template.Template{}, err
	}

	return temp, nil

}
