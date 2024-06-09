package kubernetes

import (
	"bufio"
	goContext "context"
	"flag"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"os"
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
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create the client set
	cs, err := kubernetes.NewForConfig(cc)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return cs
}

func ListContexts() map[string]*api.Context {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return config.Contexts
}

func GetCurrent() (string, string, string) {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	name := config.CurrentContext
	namespace := config.Contexts[name].Namespace
	user := config.Contexts[name].AuthInfo

	return name, namespace, user
}

func SetContext(clusterName string, namespace string, usr string) {

	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
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
func GetPods(namespace string) []v1.Pod {
	cs := getClientSet()

	pds, err := cs.CoreV1().Pods(namespace).List(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return pds.Items
}

func WatchPods(namespace string) watch.Interface {
	cs := getClientSet()

	w, err := cs.CoreV1().Pods(namespace).Watch(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return w

}

// GetNamespaces Get namespaces
func GetNamespaces() []v1.Namespace {
	cs := getClientSet()

	ns, err := cs.CoreV1().Namespaces().List(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)

	}
	return ns.Items
}

// Get pod container logs
func getPodLogs(namespace string, p string, logChan chan<- string) error {
	tl := int64(50)
	cs := getClientSet()

	opts := &v1.PodLogOptions{
		InsecureSkipTLSVerifyBackend: true,
		TailLines:                    &tl,
		Follow:                       true, // Follow the log stream of the pod
	}

	req := cs.CoreV1().Pods(namespace).GetLogs(p, opts)

	readCloser, err := req.Stream(goContext.TODO())
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

		logChan <- line
	}

	err = readCloser.Close()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return nil
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
