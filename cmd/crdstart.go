// crd-start.go
package cmd

import (
	"time"

	"net/http"
	_ "net/http/pprof"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	kubeinformers "k8s.io/client-go/informers"

	crdstartclientset "github.com/idevz/crd-start/pkg/client/clientset/versioned"
	crdstartinformers "github.com/idevz/crd-start/pkg/client/informers/externalversions"
)

const DefauleResyncDuration = time.Second * 30

var (
	masterURL     string
	kubeconfig    string
	klogV         string
	deploymentTpl string
)

// 这个方法在deployment中是没有的，deployment controller的控制器入口应该是写在其它地方
// 我怀疑是写在了：kubernetes-1.14.0\cmd\kube-controller-manager\app\apps.go：startDeploymentController
// 然后再被kubernetes-1.14.0\cmd\kube-controller-manager\app\controllermanager.go：NewControllerInitializers调用
// 控制器入口!!!
func Run() {
	klogInit()

	stopCh := SetupSignalHandler()
	// BuildConfigFromFlags 构建kubeconfig,当参数为空时，就使用kubernetes自动为Pod挂载的Service Account证书
	// 这时，控制器将以in-cluster的模式运行
	// 与之对应的是out-cluster模式,
	// which means, 自定义资源控制器不一定非要在k8s集群中运行,但为了享受k8s管理带来的各种好处，建议使用in-cluster
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("error build kubeConfig:%s", err.Error())
	}

	kubeClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error build kubeClientSet: %s", err.Error())
	}

	crdStartClientSet, err := crdstartclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error build crdStartClientSet: %s", err.Error())
	}

	go func() {
		// 启动了一个6060的http服务，是因为作者这里引入了net/http/pprof包
		// 方便后面对项目进行性能调优，源码学习等。
		// http://ip:6060/debug/pprof
		http.ListenAndServe(":6060", nil)
	}()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(
		kubeClientSet,
		DefauleResyncDuration)
	crdStartInformerFactory := crdstartinformers.NewSharedInformerFactory(
		crdStartClientSet,
		DefauleResyncDuration)

	// 创建CrdController对象
	// 参这个CrdController对象中，通过前面的kubeInformerFactory 获取所需要操作资源对象的Informer
	// 在这个例子中：作者对Deployment ReplicaSet Pods 以及自定义的Dcreaters对象进行操作
	crdController := NewCrdController(
		kubeClientSet, crdStartClientSet,
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Apps().V1().ReplicaSets(),
		kubeInformerFactory.Core().V1().Pods(),
		crdStartInformerFactory.Crdstart().V1alpha1().Dcreaters())

	kubeInformerFactory.Start(stopCh)
	crdStartInformerFactory.Start(stopCh)

	if err = crdController.Run(2, stopCh); err != nil {
		klog.Fatalf("error running controller: %s", err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeConfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	rootCmd.PersistentFlags().StringVar(&masterURL, "masterURL", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	rootCmd.PersistentFlags().StringVar(&klogV, "v", "1", "klog log level setting.")
	rootCmd.PersistentFlags().StringVar(&deploymentTpl, "deploymentTpl", "./build/artifacts/deployment-tpl.yaml", "deployment template")
}
