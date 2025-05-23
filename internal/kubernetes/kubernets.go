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
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

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
func getClientSet() *kubernetes.Clientset {
	// Use the current context in kubeconfig
	cc, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}

	// Create the client set
	cs, err := kubernetes.NewForConfig(cc)
	if err != nil {
		panic(err)
	}

	return cs
}

func ListContexts() map[string]*api.Context {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		panic(err)
	}

	return config.Contexts
}

func GetCurrent() (string, string, string) {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		panic(err)
	}
	name := config.CurrentContext
	namespace := config.Contexts[name].Namespace
	user := config.Contexts[name].AuthInfo

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
	cs := getClientSet()

	pds, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pds.Items, nil
}

func WatchPods(ctx context.Context, namespace string) (watch.Interface, error) {
	cs := getClientSet()

	w, err := cs.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return w, nil

}

// GetNamespaces Get namespaces
func GetNamespaces(ctx context.Context) ([]v1.Namespace, error) {
	cs := getClientSet()

	ns, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return ns.Items, nil
}

// GetPodLogs Get pod container logs
func GetPodLogs(ctx context.Context, namespace string, p string, logChan chan<- string) error {
	tl := int64(50)
	cs := getClientSet()

	opts := &v1.PodLogOptions{
		InsecureSkipTLSVerifyBackend: true,
		TailLines:                    &tl,
		Follow:                       true, // Follow the log stream of the pod
	}

	req := cs.CoreV1().Pods(namespace).GetLogs(p, opts)

	readCloser, err := req.Stream(ctx)
	if err != nil {
		errMsg := fmt.Errorf("errMsg in opening stream: %v", err)
		fmt.Println(errMsg)
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
