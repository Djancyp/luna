package pkg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/buke/quickjs-go"
	"github.com/rs/zerolog"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

type Asset struct {
	Path    string
	Content string
}

var textEncoderPolyfill = `function TextEncoder(){}TextEncoder.prototype.encode=function(string){var octets=[];var length=string.length;var i=0;while(i<length){var codePoint=string.codePointAt(i);var c=0;var bits=0;if(codePoint<=0x0000007F){c=0;bits=0x00}else if(codePoint<=0x000007FF){c=6;bits=0xC0}else if(codePoint<=0x0000FFFF){c=12;bits=0xE0}else if(codePoint<=0x001FFFFF){c=18;bits=0xF0}octets.push(bits|(codePoint>>c));c-=6;while(c>=0){octets.push(0x80|((codePoint>>c)&0x3F));c-=6}i+=codePoint>=0x10000?2:1}return octets};function TextDecoder(){}TextDecoder.prototype.decode=function(octets){var string="";var i=0;while(i<octets.length){var octet=octets[i];var bytesNeeded=0;var codePoint=0;if(octet<=0x7F){bytesNeeded=0;codePoint=octet&0xFF}else if(octet<=0xDF){bytesNeeded=1;codePoint=octet&0x1F}else if(octet<=0xEF){bytesNeeded=2;codePoint=octet&0x0F}else if(octet<=0xF4){bytesNeeded=3;codePoint=octet&0x07}if(octets.length-i-bytesNeeded>0){var k=0;while(k<bytesNeeded){octet=octets[i+k+1];codePoint=(codePoint<<6)|(octet&0x3F);k+=1}}else{codePoint=0xFFFD;bytesNeeded=octets.length-i}string+=String.fromCodePoint(codePoint);i+=bytesNeeded+1}return string};`
var processPolyfill = `var process = {env: {NODE_ENV: "development"}};`
var consolePolyfill = `var console = {log: function(){}};`

type JobRunner struct {
	Logger           zerolog.Logger
	Path             string
	ServerEntryPoint string
	ClientEntryPoint string
	ClientJS         string
	Env              string
	Routes           []ReactRoute
}

type serverRenderResult struct {
	html string
	css  string
	err  error
}

type clientRenderResult struct {
	js  string
	err error
}

func MatchPath(routePath, actualPath string) (bool, map[string]string) {
	// Convert /product/:id to a regex
	regexPattern := regexp.MustCompile(`:[^/]+`).ReplaceAllString(routePath, `([^/]+)`)

	// Compile the full regex
	matched, _ := regexp.MatchString("^"+regexPattern+"$", actualPath)
	if !matched {
		return false, nil
	}

	// Extract parameters using the regex
	re := regexp.MustCompile("^" + regexPattern + "$")
	matches := re.FindStringSubmatch(actualPath)

	// Create a map for captured parameters
	params := make(map[string]string)
	for i, name := range regexp.MustCompile(`:[^/]+`).FindAllString(routePath, -1) {
		// Use the parameter name without the ':' character
		params[name[1:]] = matches[i+1]
	}
	return true, params
}

type BuildResult struct {
	JS  string
	CSS string
}

var Loader = map[string]esbuild.Loader{
	".png":   esbuild.LoaderFile,
	".svg":   esbuild.LoaderFile,
	".jpg":   esbuild.LoaderFile,
	".jpeg":  esbuild.LoaderFile,
	".gif":   esbuild.LoaderFile,
	".bmp":   esbuild.LoaderFile,
	".woff2": esbuild.LoaderFile,
	".woff":  esbuild.LoaderFile,
	".ttf":   esbuild.LoaderFile,
	".eot":   esbuild.LoaderFile,
	".mp4":   esbuild.LoaderFile,
	".webm":  esbuild.LoaderFile,
	".wav":   esbuild.LoaderFile,
	".mp3":   esbuild.LoaderFile,
	".m4a":   esbuild.LoaderFile,
	".aac":   esbuild.LoaderFile,
	".oga":   esbuild.LoaderFile,
	".json":  esbuild.LoaderFile,
	".txt":   esbuild.LoaderFile,
	".xml":   esbuild.LoaderFile,
	".csv":   esbuild.LoaderFile,
	".ts":    esbuild.LoaderTS,
	".tsx":   esbuild.LoaderTSX,
	".js":    esbuild.LoaderJS,
	".jsx":   esbuild.LoaderJSX,
	".css":   esbuild.LoaderCSS,
	".html":  esbuild.LoaderFile,
}

func (j JobRunner) BuildClient(props map[string]interface{}, store map[string]interface{}) (BuildResult, error) {
	env := j.Env
	if store == nil {
		store = map[string]interface{}{}
	}
	jsonProps, error := json.Marshal(props)
	if error != nil {
		return BuildResult{}, error
	}
	jsonStore, error := json.Marshal(store)
	if error != nil {
		return BuildResult{}, error
	}
	opt := esbuild.Build(esbuild.BuildOptions{
		EntryPoints:       []string{j.ClientEntryPoint},
		Outdir:            "/",
		AssetNames:        fmt.Sprintf("%s/[name]", strings.TrimPrefix("/assets/", "/")),
		Bundle:            true,
		Write:             false,
		Metafile:          false,
		MinifyWhitespace:  env == "production",
		MinifyIdentifiers: env == "production",
		MinifySyntax:      env == "production",
		Loader:            Loader,
		Define: map[string]string{
			"props": string(jsonProps),
			"store": string(jsonStore),
		},
	})

	if len(opt.Errors) > 0 {
		return BuildResult{},
			fmt.Errorf("build error: %v", opt.Errors[0].Text)
	}

	result := BuildResult{}
	for _, file := range opt.OutputFiles {

		if strings.HasSuffix(file.Path, ".css") {
			result.CSS = string(file.Contents)
		} else if strings.HasSuffix(file.Path, ".js") {
			result.JS = string(file.Contents)
		}
	}

	return result, nil
}

func (j JobRunner) BuildServer(path string, props map[string]interface{}, store map[string]interface{}) (BuildResult, error) {
	env := j.Env

	if store == nil {
		store = map[string]interface{}{}
	}
	jsonProps, error := json.Marshal(props)
	if error != nil {
		panic(error)
	}

	jsonStore, error := json.Marshal(store)
	if error != nil {
		panic(error)
	}

	opt := esbuild.Build(esbuild.BuildOptions{
		EntryPoints:       []string{j.ServerEntryPoint},
		Bundle:            true,
		Write:             false,
		Outdir:            "/",
		Format:            esbuild.FormatESModule, // Use ES Module format
		Platform:          esbuild.PlatformBrowser,
		Target:            esbuild.ES2020,
		AssetNames:        fmt.Sprintf("%s/[name]", strings.TrimPrefix("/assets/", "/")),
		MinifyWhitespace:  env == "production",
		MinifyIdentifiers: env == "production",
		MinifySyntax:      env == "production",
		Define: map[string]string{
			"props": string(jsonProps),
			"store": string(jsonStore),
		},
		Banner: map[string]string{
			"js": textEncoderPolyfill + processPolyfill + consolePolyfill,
		},
		Loader: Loader,
	})

	if len(opt.Errors) > 0 {
		return BuildResult{},
			fmt.Errorf("build error: %v", opt.Errors[0].Text)
	}

	result := BuildResult{}
	for _, file := range opt.OutputFiles {

		if strings.HasSuffix(file.Path, ".css") {
			result.CSS = string(file.Contents)
		} else if strings.HasSuffix(file.Path, ".js") {
			result.JS = string(file.Contents)
		}
	}

	return result, nil
}

func RenderServer(js string, path string) (string, error) {
	// Initialize QuickJS runtime with module support
	rt := quickjs.NewRuntime(quickjs.WithModuleImport(true))
	defer rt.Close()

	ctx := rt.NewContext()
	defer ctx.Close()

	_, err := ctx.LoadModule(js, "server")
	if err != nil {
		panic(err)
	}

	opt := quickjs.EvalAwait(true)

	// Print the JSON representation
	script := fmt.Sprintf(`
      globalThis.URL = class {
          constructor(url) {
            this.href = url;
          }
        };
        const window = {
          location: {
            pathname: "%s"
          }
        };
      async function start() {
          try {
              const { render } = await import("server");
              const { html } = render("%s");  // Use the dynamic path here
              globalThis.result = html;
          } catch (e) {
              globalThis.result = "Error: " + e.toString();
          }
      }
      start();`, path, path)
	_, err = ctx.Eval(script, opt)
	if err != nil {
		panic(err)
	}
	return ctx.Globals().Get("result").String(), nil
}

func RenderClientWithProps(js string, props map[string]interface{}, path string) (string, error) {
	jsonProps, error := json.Marshal(props)
	if error != nil {
		return js, error
	}
	newJs := fmt.Sprintf(`
  globalThis.props = {'%s':%s};
  %s
  `, path, jsonProps, js)

	return newJs, nil
}
