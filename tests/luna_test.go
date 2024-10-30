package luna

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Djancyp/luna"
	"github.com/Djancyp/luna/internal"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Mock configuration
	mockConfig := luna.Config{
		AssetsPath:  "./assets",
		EnteryPoint: "./assets/index.html",
		Routes: []internal.ReactRoute{
			{
				Path: "/test",
				Props: func() map[string]interface{} {
					return map[string]interface{}{
						"name": "Test Route",
						"id":   123,
					}
				},
				Head: internal.Head{
					Title:       "Test Route",
					Description: "Test Route Description",
					CssLinks: []internal.CssLink{
						{
							Href: "test.css",
							DynamicAttrs: map[string]string{
								"rel": "stylesheet",
							},
						},
					},
					JsLinks: []internal.JsLink{
						{
							Src: "test.js",
							DynamicAttrs: map[string]string{
								"type": "module",
							},
						},
					},
				},
			},
		},
	}

	// Call New with the mock configuration
	app, err := luna.New(mockConfig)
	assert.NoError(t, err)
	assert.NotNil(t, app)

	// Test the /props endpoint
	reqBody := luna.PropsResponse{Path: "/test"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/props", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := app.Server.NewContext(req, rec)
	app.Server.Router().Find(http.MethodPost, "/props", c)
	// Execute request
	err = c.Handler()(c) // Directly call the handler
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Validate response
	assert.Contains(t, response, "/test")
	assert.Equal(t, "Test Route", response["/test"].(map[string]interface{})["name"])
	assert.Equal(t, float64(123), response["/test"].(map[string]interface{})["id"])
}

func TestCheckApp(t *testing.T) {
	// Mock configuration
	mockConfig := luna.Config{
		AssetsPath:  "./assets",
		EnteryPoint: "./assets/index.html",
	}

	// Call New with the mock configuration
	app, err := luna.New(mockConfig)
	assert.NoError(t, err)
	assert.NotNil(t, app)

	// Test CheckApp
	err = app.CheckApp(mockConfig)
	assert.NoError(t, err)
}
