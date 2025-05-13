# k8s-rules-viewer

A terminal UI (TUI) tool for visualizing Kubernetes deployment, service, pod, and rules compliance information, with Krakend config checks.

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

---
