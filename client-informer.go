package main

import (
	"flag"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
)

func main() {

	// kubeconfig路径
	var kubeconfig *string

	// 如果home目录存在
	if home := homedir.HomeDir(); home != "" {
		// kubeconfig默认值取$home/.kube/config
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// kubeconfig默认值取空
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

	}

	// 解析kubeconfig值
	flag.Parse()

	// 通过路径获取config配置
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	// 初始化 client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err.Error())
	}

	// 停止信号
	stopper := make(chan struct{})
	defer close(stopper)

	// 初始化 informer
	/*
	Shared指的是多个 lister 共享同一个 cache，而且资源的变化会同时通知到 cache 和 listers。这个解释和上面图所展示的内容的是一致的，
	cache 我们在 Indexer 的介绍中已经分析过了，lister 指的就是 OnAdd、OnUpdate、OnDelete 这些回调函数背后的对象。
	 */
	factory := informers.NewSharedInformerFactory(clientset, 0)
	nodeInformer := factory.Core().V1().Nodes()
	informer := nodeInformer.Informer()

	defer runtime.HandleCrash()

	// 启动 informer，list & watch
	go factory.Start(stopper)

	// 从 apiserver 同步资源，即 list
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	// 使用自定义 handler
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: onAdd,
		UpdateFunc: func(interface{}, interface{}) { fmt.Println("update not implemented") },
		// 此处省略 workqueue 的使用
		DeleteFunc: func(interface{}) { fmt.Println("delete not implemented") },
	})

	// 创建 lister
	nodeLister := nodeInformer.Lister()

	// 从 lister 中获取所有 items
	nodeList, err := nodeLister.List(labels.Everything())

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("nodelist:", nodeList)

	<-stopper
}

func onAdd(obj interface{}) {
	node := obj.(*corev1.Node)
	fmt.Println("add a node:", node.Name)
}
