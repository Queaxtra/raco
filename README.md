![](assets/banner.png)

# Raco

High-performance terminal-based HTTP client

## Features

- 3-panel terminal UI (Sidebar, Request Panel, Response Panel)
- HTTP, WebSocket, and gRPC support
- Postman, OpenAPI, and HAR import/export
- File Upload/Download (multipart/form-data)
- Mouse & keyboard support
- Clipboard integration (Drag-to-select and auto-copy)
- Git-friendly storage (JSON/YAML)
- Environment variable support
- Environment inheritance and secret-backed variables
- JSON auto-formatting
- Request History tracking
- Collection revision history and snapshots
- Collection hooks, tags, assertions, and extractors
- Real-time Metrics Dashboard
- Command Palette for quick request access
- Fast HTTP client with timeout control
- Vim-style navigation
- CLI diagnostics, config management, and shell completion
- Desktop notifications when the terminal is in the background (macOS/Linux)

## Installation

### Quick Install (Recommended)

**macOS/Linux:**
```bash
curl -sSL https://raw.githubusercontent.com/Queaxtra/raco/main/install.sh | sh
```

**Or using go install:**
```bash
go install github.com/Queaxtra/raco@latest
```

**Or using Make:**
```bash
make build    # Build binary
make install  # Build and install to /usr/local/bin
```

**Or manually:**
```bash
go build -o raco
sudo mv raco /usr/local/bin/
```

**Update to latest (when already installed):**
```bash
raco update
```

## Usage

### CLI Overview

```bash
raco req -m GET -r https://api.example.org
raco import openapi ./openapi.yaml
raco export har my-collection -o ./traffic.har
raco run my-collection -e production --report-format markdown
raco doctor --json
raco config list
```

### Main Commands
- `raco req` - Execute a single HTTP request
- `raco ws` - Open a WebSocket connection
- `raco grpc` - Run a gRPC request, reflection, or scaffold flow
- `raco col` - Manage collections, requests, hooks, tags, assertions, extractors, and history
- `raco env` - Manage environments, inheritance, secrets, and health checks
- `raco import` - Import `postman`, `openapi`, or `har`
- `raco export` - Export a collection as `openapi` or `har`
- `raco run` - Run a saved collection with filters, snapshots, and reports
- `raco config` - Manage local Raco config
- `raco doctor` - Check storage, notifications, and local setup
- `raco completion` - Print shell completion for `bash`, `zsh`, or `fish`

### Mouse Support
- Click on any panel to focus it
- Click on collections to select
- Drag to select text and copy to clipboard automatically

### Keyboard Shortcuts

**Global (vim-style)**
- `q` / `Ctrl+C` - Quit
- `Tab` / `Shift+Tab` - Next / previous field or panel
- `h` / `l` - Focus sidebar / focus request panel
- `e` - Send request (execute)
- `w` - Save request (write)
- `:` / `/` / `Ctrl+P` - Command palette
- `Esc` - Unfocus / back
- `Ctrl+B` - Toggle sidebar
- `F1` - Dashboard

**Sidebar**
- `j` / `k` - Navigate down/up
- `Ctrl+U` / `Ctrl+D` - Page up/down
- `gg` - First item (press `g` twice)
- `G` - Last item
- `Enter` - Expand/collapse collection or load request

**Request Panel**
- `Tab` / `Shift+Tab` - Next / previous input
- `←` / `→` or `h` / `l` - Change method (GET/POST/…/WS/gRPC)
- `e` / `Ctrl+R` - Send request
- `w` / `Ctrl+W` - Save request
- `Ctrl+S` / `Ctrl+D` - Add / delete header
- `Ctrl+F` / `Ctrl+X` - Add / remove file
- `p` - Open request preview

**Response Panel**
- `j` / `k` - Scroll
- `Ctrl+U` / `Ctrl+D` - Half-page up/down
- `gg` / `G` - Top / bottom
- `Tab` / `h` - Back to sidebar
- `Esc` - Back

**Command Palette**
- `j` / `k` or `↑` / `↓` - Move selection
- `Ctrl+U` / `Ctrl+D` - Page up/down
- `gg` / `G` - Top / bottom
- `Enter` - Load selected request
- `Esc` - Close palette

## Workflow

### Creating Collections
1. Press `Ctrl+N` to open collection creation modal
2. Type collection name and press `Enter`
3. Collection will appear in sidebar

### Saving Requests
1. Configure your request (URL, method, headers, body)
2. Press `Ctrl+W` to save
3. Enter request name and press `Enter`
4. Request will be saved to the currently expanded collection (or first collection if none expanded)

### Loading Requests
1. Navigate sidebar with `j/k`
2. Press `Enter` on a collection to expand/collapse
3. Press `Enter` on a request to load it
4. Press `Tab` to switch to request panel and modify if needed

### Using Command Palette
1. Press `Ctrl+P` to open the palette
2. Type to filter requests
3. Use `j/k` or `↑/↓` to navigate
4. Press `Enter` to load selected request

### Viewing Request History
1. Look for "History" section in sidebar
2. Use `j/k` to navigate to history entries
3. Press `Enter` on a history entry to reload it

### WebSocket / gRPC Connections
1. In Request Panel, use `←/→` to select WS or gRPC protocol
2. Enter the URL
3. Press `Ctrl+R` to connect
4. Type messages and press `Enter` to send
5. Press `Ctrl+Q` to disconnect

### Viewing Metrics Dashboard
1. Press `F1` anytime to open Dashboard
2. View request statistics, success rates, and recent activity
3. Press `Tab` or `Esc` to return to sidebar

## Import and Export

- `raco import postman <file>` - Import a Postman v2.1 collection
- `raco import openapi <file>` - Import an OpenAPI 3.x document
- `raco import har <file>` - Import captured browser or proxy traffic
- `raco export openapi <collection-id> -o <file>` - Export a collection as OpenAPI
- `raco export har <collection-id> -o <file>` - Export a collection as HAR

## Collections and Environments

- Collections support request hooks, request tags, assertions, extractors, revision history, and run-time snapshots.
- Environments support plain variables, secret-backed values, inheritance via `set-parent`, and health inspection via `env health`.
- `raco col history <collection-id>` lists stored revisions.
- `raco col diff <collection-id> <revision-a> <revision-b>` compares saved revisions.
- `raco col hooks set <collection-id> --setup <ref> --teardown <ref>` configures collection-level setup and teardown requests.
- `raco env rotate-secret <name> <key>` rotates a secret-backed variable without rewriting plaintext environment values.

## Collection Runs

- `raco run <collection-id> -e <env>` runs a collection in a selected environment.
- `--tag <tag>` filters requests by collection or request tags.
- `--graph` prints execution order before running.
- `--snapshot-dir <dir>` and `--snapshot-update` control snapshot verification.
- `--report-format text|json|junit|markdown|github|sarif` selects CI output format.

## Config and Diagnostics

- `raco config list` prints the active local config values.
- `raco config set <key> <value>` updates config keys such as snapshot, history, cookie, and script directories.
- `raco doctor` reports local storage and platform integration health.
- `raco completion bash|zsh|fish` prints shell completion scripts.

### Desktop notifications

When you run a request (TUI or CLI) or a collection run, Raco can send an OS-level notification so you see the result even if the terminal is not focused:

- **TUI:** Every in-app toast (e.g. “Request saved”, “Connected”, errors) also triggers a desktop notification.
- **CLI `raco req`:** On success you get “Request completed: &lt;status&gt;”; on failure, “Request failed: &lt;error&gt;”.
- **CLI `raco run`:** After the run, you get “&lt;collection&gt;: X passed, Y failed”.

Supported platforms: **macOS** (via `osascript`), **Linux** (via `notify-send`; install `libnotify` if needed). Other platforms show no desktop notification.

## Storage

Collections: `~/.raco/collections/*.json`
Environments: `~/.raco/environments/*.yaml`
History: `~/.raco/history/<collection-id>/*.json`
Snapshots: `~/.raco/snapshots/<collection-id>/*.json`
Cookies: `~/.raco/cookies/*.json`
Scripts: `~/.raco/scripts/*`
Config: `~/.raco/config.yaml`

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request with your changes. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
