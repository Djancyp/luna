# Luna Framework - QWEN.md

## Project Overview

Luna is a Golang-based full-stack framework that leverages the Echo web framework for server-side rendering (SSR) of React applications. It provides a seamless, full-stack development experience for building dynamic and performant web applications, making it ideal for full-stack developers looking to integrate the power of Golang on the backend with the flexibility of React on the frontend.

**Status**: Beta quality - expect breaking changes and bugs

**Key Features**:
- Full backend API support with Echo
- Server-side rendering (SSR) with React
- Hot reloading development
- Route-based caching
- Custom middleware support
- Tailwind CSS integration
- WebSocket-based hot reloading

## Project Structure

```
luna/
├── cli/
│   ├── main.go
│   └── cmd/
│       ├── project.go
│       └── root.go
├── pkg/
│   ├── cache.go
│   ├── esbuild.go
│   ├── html.go
│   ├── tailwind.go
│   └── types.go
├── tests/
├── utils/
├── hotreload.go
├── luna.go
├── types.go
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

**Core Components**:
- `luna.go` - Main framework engine and application setup
- `types.go` - Core types and engine definitions
- `pkg/` - Core functionality packages
  - `esbuild.go` - Build system using esbuild
  - `html.go` - HTML template generation
  - `cache.go` - Route-based caching system
  - `types.go` - Type definitions for routes and configuration
- `hotreload.go` - Hot reload functionality with WebSocket support
- `cli/` - Command-line interface tools for project initialization

## Building and Running

### Prerequisites
- Go 1.21.6+
- Yarn or npm
- Node.js

### Installation
```bash
go get github.com/Djancyp/luna@v1.0.0
```

### Usage Example
```go
func main() {
    app, err := luna.New(luna.Config{
        ENV:                 "development", // or "production"
        RootPath:            "frontend/",
        PublicPath:          "frontend/public",
        AssetsPath:          "frontend/src/assets",
        ServerEntryPoint:    "frontend/src/entry-server.tsx",
        ClientEntryPoint:    "frontend/src/entry-client.tsx",
        TailwindCSS:         true,
        FaviconPath:         "favicon.svg",
        HotReloadServerPort: 3000,
        Store:               store.ReturnStore,
        Head: pkg.MainHead{
            Attributes: []string{
                `<link rel="manifest" href="/manifest.json" />`,
            },
        },
        Routes: []pkg.ReactRoute{
            {
                Path: "/",
                Head: pkg.Head{
                    Title:       "My App",
                    Description: "Home page",
                    CssLinks:    []pkg.CssLink{{Href: "main.css"}},
                },
                Middleware: []echo.MiddlewareFunc{
                    // custom middleware
                },
                CacheExpiry: time.Now().Add(365 * 24 * time.Hour).Unix(),
            },
            // Additional routes...
        },
    })
    if err != nil {
        panic(err)
    }
    
    app.Start(":8080")
}
```

### CLI Commands
- `luna-cli init` - Initialize a new Luna project with an interactive prompt

## Architecture

### Core Components

1. **Engine**: The main application engine that wraps Echo and manages the lifecycle
2. **Config**: Configuration struct that defines the application setup
3. **Route System**: Supports parameterized routes with middleware and caching
4. **Build System**: Uses esbuild for both client and server bundles
5. **SSR Engine**: Uses QuickJS for server-side rendering of React components
6. **Caching Layer**: Route-based caching with expiration times
7. **Hot Reload**: WebSocket-based hot reload for development

### Data Flow

1. Request comes in and is matched against configured routes
2. Route-specific middleware is applied
3. Route-specific props are generated via configured functions
4. Global store data is collected
5. Both client and server bundles are generated with embedded props/store
6. Server bundle is executed in QuickJS to generate HTML
7. HTML template is rendered with CSS, JS, and other head elements
8. Response is sent to the client

### Global Store vs Route Props

- **Global Store**: Available on all pages, configured once in the main config
- **Route Props**: Page-specific data generated per request

## Development Conventions

### Route Configuration
- Routes support parameterized paths (e.g., `/user/:id`)
- Each route can have specific middleware
- Cache expiry times can be configured per route
- Head tags (title, description, CSS, JS) can be set per route

### File Structure
- Frontend code typically in `frontend/` directory
- Server entry point: `entry-server.tsx` (for SSR)
- Client entry point: `entry-client.tsx` (for hydration)
- Assets in `src/assets/`
- Public files in `public/`

### Build Process
- Uses esbuild for both client and server builds
- Client bundle includes React hydration code
- Server bundle is used for SSR
- CSS is extracted and included in the HTML
- Assets are processed and copied to the output

## Testing

The project includes unit tests in the `tests/` directory, primarily testing:
- Engine initialization
- Route configuration
- Property handling
- Navigation endpoint

## Dependencies

Key dependencies include:
- `github.com/labstack/echo/v4` - Web framework
- `github.com/evanw/esbuild` - JavaScript bundler
- `github.com/buke/quickjs-go` - JavaScript engine for SSR
- `github.com/gorilla/websocket` - WebSocket support for hot reload
- `github.com/rs/zerolog` - Logging
- `github.com/spf13/cobra` - CLI command framework

## Special Notes

- The framework is in beta and may have breaking changes
- SSR is performed using QuickJS, which has some limitations compared to Node.js
- Hot reload is implemented using WebSocket connections
- Route-based caching can significantly improve performance
- The framework supports both TypeScript and JavaScript for frontend code
- Tailwind CSS integration is available through the configuration

## License

MIT License