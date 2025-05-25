# K8s Manager

K8s Manager is a terminal user interface (TUI) application written in Go that provides a convenient way to interact with Kubernetes clusters. 
It offers real-time monitoring and management of Kubernetes resources including contexts, namespaces, pods, deployments, and logs.

## Features

- **Context Management**: View and switch between Kubernetes contexts
- **Namespace Management**: View and switch between namespaces in the current context
- **Pod Monitoring**: Real-time view of pods in the selected namespace
- **Deployment Monitoring**: Real-time view of deployments in the selected namespace
- **Log Viewing**: Stream logs from pods in real-time with optimized memory usage
- **Error Handling**: Robust error handling with informative error messages
- **Memory Optimization**: Prevents excessive memory usage during long-running sessions
- **Enhanced Scrolling**: Improved scroll speed for better navigation through logs

## Requirements

- Go 1.24.0 or higher
- Access to a Kubernetes cluster
- Valid kubeconfig file (compatible with client v1.29)

## Installation

### Option 1: Download pre-built binaries

You can download pre-built binaries for your platform from the [GitHub Releases](https://github.com/OliveiraNt/k8s-manager/releases) page.

1. Go to the [Releases page](https://github.com/OliveiraNt/k8s-manager/releases)
2. Download the binary for your operating system (Windows, macOS, or Linux)
3. Extract the archive if necessary
4. Make the binary executable (Linux/macOS only): `chmod +x k8s-manager`

### Option 2: Build from source

Clone the repository and build the application:

```bash
git clone https://github.com/OliveiraNt/k8s-manager.git
cd k8s-manager
go build -o k8s-manager ./cmd
```

## Usage

Run the program without arguments to start the user interface:

```bash
./k8s-manager
```

### Navigation

The application has several views that you can navigate between:

#### Pods View (Default)
- Arrow keys: Navigate through the list of pods
- `c`: Switch to Context view
- `n`: Switch to Namespace view
- `d`: Switch to Deployments view
- `Enter`: View logs of the selected pod
- `q` or `Ctrl+C`: Quit the application

#### Deployments View
- Arrow keys: Navigate through the list of deployments
- `c`: Switch to Context view
- `n`: Switch to Namespace view
- `p`: Switch to Pods view
- `Esc`: Return to Pods view
- `q` or `Ctrl+C`: Quit the application

#### Context View
- Arrow keys: Navigate through the list of contexts
- `Enter`: Select a context and return to Pods view
- `Esc`: Return to Pods view without changing context
- `q` or `Ctrl+C`: Quit the application

#### Namespace View
- Arrow keys: Navigate through the list of namespaces
- `Enter`: Select a namespace and return to Pods view
- `Esc`: Return to Pods view without changing namespace
- `q` or `Ctrl+C`: Quit the application

#### Logs View
- Mouse wheel: Scroll through logs (enhanced scroll speed)
- Arrow keys: Scroll through logs
- `Esc`: Return to Pods view
- `q` or `Ctrl+C`: Quit the application

## Contributing

Contributions are welcome! Please feel free to open an issue or pull request.

## License

This project is licensed under the terms of the GLWTS License.
