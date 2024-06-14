# K8s Manager

K8s Manager is a command line application written in Go that allows the user to interact with a Kubernetes cluster. It allows the user to view and change the current context and namespace, as well as view the pods running in the selected namespace.

## Requirements

- Go 1.22.0 or higher
- Access to a Kubernetes cluster and a valid kubeconfig configuration file

## Dependencies

The following Go libraries are used in this project:

- github.com/charmbracelet/bubbles v0.18.0
- github.com/charmbracelet/bubbletea v0.26.4
- github.com/charmbracelet/lipgloss v0.11.0
- k8s.io/api v0.30.1
- k8s.io/apimachinery v0.30.1
- k8s.io/client-go v0.30.1

## Usage

Run the program without arguments to start the user interface. Use the arrow keys to navigate and the Enter key to select an item. You can press 'n' to change the current namespace and 'c' to change the current context.

## Contributing

Contributions are welcome! Please feel free to open an issue or pull request.

## License

This project is licensed under the terms of the GLWTS License.
