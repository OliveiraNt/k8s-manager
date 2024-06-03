package main

import (
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

func listContexts() map[string]*api.Context {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		panic(err)
	}

	return config.Contexts
}

func getCurrent() (string, string) {
	config, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		panic(err)
	}
	currentContext := config.CurrentContext
	currentNs := config.Contexts[currentContext].Namespace

	return currentContext, currentNs
}

func setContext(clusterName string, namespace string, usr string) {

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

// Get pods (use namespace)
func getPods(namespace string) []v1.Pod {
	cs := getClientSet()

	pds, err := cs.CoreV1().Pods(namespace).List(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	return pds.Items
}

func watchPods(namespace string) <-chan watch.Event {
	cs := getClientSet()

	w, err := cs.CoreV1().Pods(namespace).Watch(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return w.ResultChan()

}

// Get namespaces
func getNamespaces() []v1.Namespace {
	cs := getClientSet()

	ns, err := cs.CoreV1().Namespaces().List(goContext.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)

	}
	return ns.Items
}

// Get pod container logs
func getPodContainerLogs(namespace string, p string, c string, o io.Writer) error {
	tl := int64(50)
	cs := getClientSet()

	opts := &v1.PodLogOptions{
		Container: c,
		TailLines: &tl,
	}

	req := cs.CoreV1().Pods(namespace).GetLogs(p, opts)

	readCloser, err := req.Stream(goContext.TODO())
	if err != nil {
		return err
	}

	_, err = io.Copy(o, readCloser)

	readCloser.Close()

	return err
}

// Column helper: Restarts
func columnHelperRestarts(cs []v1.ContainerStatus) string {
	r := 0
	for _, c := range cs {
		r = r + int(c.RestartCount)
	}
	return strconv.Itoa(r)
}

// Column helper: Age
func columnHelperAge(t metav1.Time) string {
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

// Column helper: Status
func columnHelperStatus(s v1.PodStatus) string {
	return fmt.Sprintf("%s", s.Phase)
}

// Column helper: Ready
func columnHelperReady(cs []v1.ContainerStatus) string {
	cr := 0
	for _, c := range cs {
		if c.Ready {
			cr = cr + 1
		}
	}
	return fmt.Sprintf("%d/%d", cr, len(cs))
}
