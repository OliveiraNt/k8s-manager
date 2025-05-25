package kubernetes

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type config struct {
	kubeConfig *string
}

// Get configuration
func getConfig() config {
	c := config{}

	if home := homedir.HomeDir(); home != "" {
		c.kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		c.kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	return c
}

var kubeConfig = getConfig().kubeConfig

// Get Kubernetes client set
func getClientSet() (*kubernetes.Clientset, error) {
	// Use the current context in kubeconfig
	cc, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		log.Printf("[ERROR] Failed to build config from flags: %v", err)
		return nil, fmt.Errorf("failed to build config from flags: %w", err)
	}

	cc.Timeout = time.Second * 5

	// Create the client set
	cs, err := kubernetes.NewForConfig(cc)
	if err != nil {
		log.Printf("[ERROR] Failed to create client set: %v", err)
		return nil, fmt.Errorf("failed to create client set: %w", err)
	}

	return cs, nil
}

func ListContexts() map[string]*api.Context {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		log.Printf("[ERROR] Failed to load kubeconfig: %v", err)
		// Return an empty map instead of panicking
		return make(map[string]*api.Context)
	}

	return config.Contexts
}

func GetCurrent() (string, string, string) {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		log.Printf("[ERROR] Failed to load kubeconfig: %v", err)
		// Return empty values instead of panicking
		return "", "", ""
	}

	name := config.CurrentContext
	if name == "" {
		log.Printf("[WARNING] No current context set in kubeconfig")
		return "", "", ""
	}

	context, exists := config.Contexts[name]
	if !exists {
		log.Printf("[ERROR] Current context '%s' not found in kubeconfig", name)
		return name, "", ""
	}

	namespace := context.Namespace
	user := context.AuthInfo

	return name, namespace, user
}

func SetContext(clusterName string, namespace string, usr string) {

	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		panic(err)
	}
	ctx := api.NewContext()
	ctx.Cluster = clusterName
	ctx.Namespace = namespace
	ctx.AuthInfo = usr

	config.Contexts[clusterName] = ctx
	config.CurrentContext = clusterName

	err = clientcmd.WriteToFile(*config, *kubeConfig)
	if err != nil {
		panic(err)
	}
}

// GetPods Get pods (use namespace)
func GetPods(ctx context.Context, namespace string) ([]v1.Pod, error) {
	cs, err := getClientSet()
	if err != nil {
		return nil, err
	}

	pds, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pds.Items, nil
}

func WatchPods(ctx context.Context, namespace string) (watch.Interface, error) {
	cs, err := getClientSet()
	if err != nil {
		return nil, err
	}

	timeoutSeconds := int64(5)
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, err := cs.CoreV1().Pods(namespace).Watch(ctx, options)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// GetNamespaces Get namespaces
func GetNamespaces(ctx context.Context) ([]v1.Namespace, error) {
	cs, err := getClientSet()
	if err != nil {
		return nil, err
	}

	ns, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return ns.Items, nil
}

// GetPodLogs Get pod container logs
func GetPodLogs(ctx context.Context, namespace string, p string, logChan chan<- string) error {
	tl := int64(50)
	cs, err := getClientSet()
	if err != nil {
		log.Printf("[ERROR] Failed to get client set for pod logs: %v", err)
		return err
	}

	opts := &v1.PodLogOptions{
		InsecureSkipTLSVerifyBackend: true,
		TailLines:                    &tl,
		Follow:                       true, // Follow the log stream of the pod
	}

	req := cs.CoreV1().Pods(namespace).GetLogs(p, opts)

	readCloser, err := req.Stream(ctx)
	if err != nil {
		errMsg := fmt.Errorf("error in opening stream: %v", err)
		log.Println(errMsg)
		return err
	}

	reader := bufio.NewReader(readCloser)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF {
			break
		}

		select {
		case logChan <- line:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return readCloser.Close()
}

// ColumnHelperRestarts Column helper: Restarts
func ColumnHelperRestarts(cs []v1.ContainerStatus) string {
	r := 0
	for _, c := range cs {
		r = r + int(c.RestartCount)
	}
	return strconv.Itoa(r)
}

// ColumnHelperAge Column helper: Age
func ColumnHelperAge(t metav1.Time) string {
	d := time.Now().Sub(t.Time)

	if d.Hours() > 1 {
		if d.Hours() > 24 {
			ds := float64(d.Hours() / 24)
			return fmt.Sprintf("%.0fd", ds)
		} else {
			return fmt.Sprintf("%.0fh", d.Hours())
		}
	} else if d.Minutes() > 1 {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d.Seconds() > 1 {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	return "?"
}

// ColumnHelperStatus Column helper: Status
func ColumnHelperStatus(s v1.PodStatus) string {
	return fmt.Sprintf("%s", s.Phase)
}

// ColumnHelperReady Column helper: Ready
func ColumnHelperReady(cs []v1.ContainerStatus) string {
	cr := 0
	for _, c := range cs {
		if c.Ready {
			cr = cr + 1
		}
	}
	return fmt.Sprintf("%d/%d", cr, len(cs))
}

// GetDeployments Get deployments in a namespace
func GetDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	cs, err := getClientSet()
	if err != nil {
		return nil, err
	}

	deps, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deps.Items, nil
}

// WatchDeployments Watch deployments in a namespace
func WatchDeployments(ctx context.Context, namespace string) (watch.Interface, error) {
	cs, err := getClientSet()
	if err != nil {
		return nil, err
	}

	timeoutSeconds := int64(5)
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, err := cs.AppsV1().Deployments(namespace).Watch(ctx, options)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// ColumnHelperReplicas Column helper: Replicas
func ColumnHelperReplicas(d appsv1.DeploymentStatus) string {
	return fmt.Sprintf("%d/%d", d.ReadyReplicas, d.Replicas)
}
