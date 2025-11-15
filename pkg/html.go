package pkg

import (
	"bytes"
	"html/template"
	"strings"
)

const templateHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    {{if .Description}}
    <meta name="description" content="{{ .Description }}" />
    {{end}}
		{{ range .MainHead }}
		{{ . }}
		{{ end }}
    {{if .Favicon}}
    <link rel="icon" type="image/svg+xml" href="{{ .Favicon }}" />
    {{end}}
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    {{if .Title}}
    <title>{{.Title}}</title>
    {{end}}
    {{ range .CssLinks }}
      {{ . }}
    {{ end }}

    {{ range .JsLinks }}
    <script src="{{.}}"></script>
    {{ end }}

    {{ if .CSS }}
    <style>
      {{ .CSS }}
    </style>
    {{ end }}

    {{if .Dev}}
    <script>
      function createWebSocket(url, maxReconnectAttempts = 10, reconnectInterval = 5000) {
        let socket = null;
        let reconnectAttempts = 0;
        let shouldReconnect = true;

        function connect() {
          try {
            socket = new WebSocket(url);

            socket.onopen = () => {
              console.log("WebSocket connected");
              socket.send(1);
              reconnectAttempts = 0; // Reset attempts on successful connection
            };

            socket.onmessage = (event) => {
              if (event.data === "reload") {
                console.log("Change detected, reloading...");
                window.location.reload();
              }
            };

            socket.onclose = () => {
              console.log("WebSocket disconnected");
              if (shouldReconnect && reconnectAttempts < maxReconnectAttempts) {
                console.log('Attempting to reconnect... (' + (reconnectAttempts + 1) + '/' + maxReconnectAttempts + ')');
                setTimeout(() => {
                  reconnectAttempts++;
                  connect();
                }, reconnectInterval);
              } else if (reconnectAttempts >= maxReconnectAttempts) {
                console.error("Max reconnection attempts reached");
              }
            };

            socket.onerror = (error) => {
              console.error("WebSocket error:", error);
              // Connection will be handled by onclose event
            };
          } catch (e) {
            console.error("Failed to create WebSocket:", e);
            if (shouldReconnect && reconnectAttempts < maxReconnectAttempts) {
              setTimeout(() => {
                reconnectAttempts++;
                connect();
              }, reconnectInterval);
            }
          }
        }

        connect();

        // Return a way to close the socket and stop reconnection attempts
        return {
          close: () => {
            shouldReconnect = false;
            if (socket) {
              socket.close();
            }
          }
        };
      }

      // Initialize WebSocket with reconnection
      const wsConnection = createWebSocket("{{ .SWUrl }}");
    </script>
    {{end}}
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
	Favicon         string
	CssLinks        []template.HTML
	JsLinks         []template.HTML
	CSS             template.CSS
	JS              template.JS
	RenderedContent template.HTML
	Dev             bool
	SWUrl           string
	MainHead        []template.HTML
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

// SanitizeRenderedContent removes potentially dangerous content from rendered HTML
func SanitizeRenderedContent(content string) string {
	// Remove potential script tags and their content
	content = removeScriptTags(content)
	// Remove potential event handlers
	content = removeEventHandlers(content)
	// Remove potential javascript: URLs
	content = removeJavascriptURLs(content)
	return content
}

// removeScriptTags removes script tags and their content
func removeScriptTags(content string) string {
	// This is a simple implementation - in a production system, you'd want to use a proper HTML parser
	// For now, we'll use a simple replace approach
	for {
		start := strings.Index(strings.ToLower(content), "<script")
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(content), "</script>")
		if end == -1 {
			// If no closing tag, remove from opening to end
			content = content[:start]
			break
		}
		end += len("</script>")
		content = content[:start] + content[end:]
	}
	return content
}

// removeEventHandlers removes common event handlers that could execute JavaScript
func removeEventHandlers(content string) string {
	// Common event handlers that could execute JavaScript
	eventHandlers := []string{
		"onabort", "onafterprint", "onanimationend", "onanimationiteration", "onanimationstart",
		"onbeforeprint", "onbeforeunload", "onblur", "oncanplay", "oncanplaythrough",
		"onchange", "onclick", "onclose", "oncontextmenu", "oncopy", "oncut", "ondblclick",
		"ondrag", "ondragend", "ondragenter", "ondragleave", "ondragover", "ondragstart", "ondrop",
		"ondurationchange", "onended", "onerror", "onfocus", "onfocusin", "onfocusout",
		"onfullscreenchange", "onfullscreenerror", "onhashchange", "oninput", "oninvalid",
		"onkeydown", "onkeypress", "onkeyup", "onload", "onloadeddata", "onloadedmetadata",
		"onloadstart", "onmessage", "onmousedown", "onmouseenter", "onmouseleave", "onmousemove",
		"onmouseout", "onmouseover", "onmouseup", "onmousewheel", "onoffline", "ononline",
		"onpagehide", "onpageshow", "onpaste", "onpause", "onplay", "onplaying", "onpopstate",
		"onprogress", "onratechange", "onreset", "onresize", "onscroll", "onsearch",
		"onseeked", "onseeking", "onselect", "onstalled", "onstorage", "onsubmit", "onsuspend",
		"ontimeupdate", "ontoggle", "onunload", "onvolumechange", "onwaiting", "onwheel",
		"oncopy", "oncut", "onpaste", "onbeforecopy", "onbeforecut", "onbeforepaste",
		"onerrorupdate", "onhelp", "onmscontentzoom", "onmsmanipulationstatechanged", "onpropertychange",
		"onreadystatechange", "ontoggle", "onactivate", "onbeforeactivate", "onbeforedeactivate",
		"ondeactivate", "onended", "onloadstart", "onpause", "onplay", "onplaying", "onstop",
	}

	for _, handler := range eventHandlers {
		// Remove the attribute and its value
		content = removeAttribute(content, handler)
	}
	return content
}

// removeAttribute removes a specific attribute from HTML tags
func removeAttribute(html, attr string) string {
	// This is a simplified implementation - for production, use a proper HTML parser
	// Look for the attribute pattern and remove it
	for {
		attrPos := strings.Index(strings.ToLower(html), attr+"=")
		if attrPos == -1 {
			break
		}

		// Find the start of the tag
		tagStart := strings.LastIndex(html[:attrPos], "<")
		if tagStart == -1 {
			break
		}

		// Find the end of the tag
		tagEnd := strings.Index(html[tagStart:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd = tagStart + tagEnd + 1

		// Find the value bounds (within quotes)
		valueStart := strings.Index(html[attrPos:], "=") + attrPos + 1
		if valueStart >= tagEnd {
			break
		}

		quoteChar := html[valueStart]
		if quoteChar != '"' && quoteChar != '\'' {
			// If no quotes, find next space or >
			nextSpace := strings.IndexAny(html[valueStart:tagEnd], " >")
			if nextSpace != -1 {
				valueEnd := valueStart + nextSpace
				// Remove the entire attribute
				html = html[:attrPos] + html[valueEnd:]
				continue
			}
		} else {
			// Skip past the quote
			valueStart++
			valueEnd := strings.Index(html[valueStart:tagEnd], string(quoteChar))
			if valueEnd != -1 {
				valueEnd += valueStart + 1
				// Remove the entire attribute
				html = html[:attrPos] + html[valueEnd:]
				continue
			}
		}

		break // If we can't find value bounds, break to avoid infinite loop
	}

	return html
}

// removeJavascriptURLs removes javascript: URLs
func removeJavascriptURLs(content string) string {
	// Remove javascript: URLs in href, src, etc.
	for {
		jsPos := strings.Index(strings.ToLower(content), "javascript:")
		if jsPos == -1 {
			break
		}

		// Find the preceding quote or space to identify the start of the URL
		start := strings.LastIndex(content[:jsPos], "\"")
		if start == -1 {
			start = strings.LastIndex(content[:jsPos], "'")
		}
		if start == -1 {
			start = strings.LastIndex(content[:jsPos], " ")
		}
		if start == -1 {
			start = 0
		} else {
			start++
		}

		// Find the end of the URL (next quote, space, or >)
		end := strings.Index(content[jsPos:], "\"")
		temp := strings.Index(content[jsPos:], "'")
		if temp != -1 && (end == -1 || temp < end) {
			end = temp
		}
		temp = strings.Index(content[jsPos:], " ")
		if temp != -1 && (end == -1 || temp < end) {
			end = temp
		}
		temp = strings.Index(content[jsPos:], ">")
		if temp != -1 && (end == -1 || temp < end) {
			end = temp
		}

		if end != -1 {
			end = jsPos + end
			content = content[:start] + content[end:]
		} else {
			content = content[:start]
			break
		}
	}

	return content
}
