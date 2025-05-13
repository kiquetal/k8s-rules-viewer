# k8s-rules-viewer

A terminal UI (TUI) tool for visualizing Kubernetes deployment, service, pod, and rules compliance information, with Krakend config checks.

## Table of Contents
- [TUI Layout](#tui-layout-ascii-art)
- [How to Run](#how-to-run)
- [Keyboard Shortcuts](#keyboard-shortcuts)
- [Using the GitHub Actions Build](#using-the-github-actions-build)
- [Module Verification](#module-verification)

## TUI Layout (ASCII Art)

```
+---------------------------------------------------------------+
|         k8s-viewer-rules - Label: <label> - Namespace: <ns>   |
+---------------------------------------------------------------+
| +-------------------+ +-------------------+ +---------------+ |
| | Deployment        | | Service           | | Pod Monitoring| |
| | Details           | | Details           | | (label: ...)  | |
| |-------------------| |-------------------| |---------------| |
| |                   | |                   | |               | |
| |                   | |                   | |               | |
| +-------------------+ +-------------------+ +---------------+ |
+---------------------------------------------------------------+
|                Rules Compliance                               |
|---------------------------------------------------------------|
|                                                               |
+---------------------------------------------------------------+
|           Krakend Config Check (<krakend-map>)                |
|---------------------------------------------------------------|
|                                                               |
+---------------------------------------------------------------+
| Use Tab to switch focus between panels.                       |
| Use arrow keys to scroll content. Press Ctrl+C to exit.       |
+---------------------------------------------------------------+
```

## How to Run

1. **Build the CLI:**

   ```sh
   go build -o k8s-rules-viewer ./cmd/main.go
   ```

2. **Set your kubeconfig (if not default):**

   By default, the tool uses `$HOME/.kube/config`.  
   To use a different config, set the `KUBECONFIG` environment variable:

   ```sh
   export KUBECONFIG=/path/to/your/kubeconfig
   ```

3. **Run the CLI:**

   ```sh
   ./k8s-rules-viewer \
     -label <app-label> \
     -namespace <namespace> \
     -krakend-map <krakend-configmap-name>
   ```

   - `-label`: Application label to filter resources (default: `py-kannel`)
   - `-namespace`: Kubernetes namespace (default: `default`)
   - `-krakend-map`: Krakend ConfigMap name (default: `krakend-config`)

   Example:

   ```sh
   ./k8s-rules-viewer -label my-app -namespace prod -krakend-map krakend-prod
   ```

## Keyboard Shortcuts

- **Tab / Shift+Tab**: Switch focus between panels
- **Arrow keys**: Scroll content in focused panel
- **Ctrl+C**: Exit the application

## Using the GitHub Actions Build

You can download the pre-built executable from the GitHub Actions artifacts:

1. Go to the Actions tab in the repository
2. Select the most recent "Manual Operations Workflow" run
3. Download the artifact for your environment
4. Extract and make the binary executable:
   ```sh
   chmod +x k8s-rules-viewer
   ```

## Module Verification

To verify your Go module setup is correct:

```sh
# Verify module dependencies are correctly configured
go mod verify

# If you need to update dependencies
go mod tidy

# To ensure you're using Go version 1.24.0 as specified in go.mod
go version
```

---

`
