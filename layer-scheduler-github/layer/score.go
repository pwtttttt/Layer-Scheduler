package layer

import (
	"context"
	"fmt"
	"math"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type LayerPro struct {
	catchHandler *ImageMetadataLists
	imageHandler *DockerImages
	handle       framework.Handle
}

var _ = framework.ScorePlugin(&LayerPro{})

const Name = "LayerPro" //LayerPro是  插件名称

func (pl *LayerPro) Name() string {
	return Name
}

func (pl *LayerPro) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.Infof("enter plugin LayerPro")
	nodeInfo, err := pl.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		klog.Errorf("getting node %q from Snapshot: %v", nodeName, err)
		return 0, framework.AsStatus(fmt.Errorf("getting node %q from Snapshot: %w", nodeName, err))
	}
	// 注册一个控制镜像的服务，里面包括docker客户端
	pl.ImageHandlerRegister(nodeInfo)
	// 初始化一个列表，里面是pod下所有container的镜像名称
	imageNames := []DockerImageName{}
	for _, c := range pod.Spec.Containers {
		imageNames = append(imageNames, DockerImageName(c.Image))
	}

	layerExistSize, resScore := pl.ComputeLayerScore(imageNames, nodeInfo.Node().Name)

	layerSizeMB := layerExistSize / 1024 / 1024
	weight := pl.computeWeight(nodeInfo, nodeInfo.Node().Name, layerSizeMB, pod)
	klog.Infof("当前节点: %s, 权重: %v, 镜像名称: %s", nodeInfo.Node().Name, weight, imageNames)

	res := float64(resScore) * weight
	klog.Infof("当前节点: %s, 层调度器原始得分: %d, 层调度器加权分数: %d", nodeInfo.Node().Name, int64(resScore), int64(res))
	return int64(res), nil
}

func (pl *LayerPro) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	c, err := NewImageMetadataListFromCache("/etc/kubernetes/cache.json")
	if err != nil {
		return nil, err
	}
	return &LayerPro{
		catchHandler: c,
		handle:       h,
	}, nil
}

func (pl *LayerPro) ImageHandlerRegister(nodeInfo *framework.NodeInfo) {
	var nodeAddress string
	for _, addr := range nodeInfo.Node().Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			// 获取node节点IP
			nodeAddress = addr.Address
			break
		}
	}
	if len(nodeAddress) == 0 {
		klog.Error("node address is empty")
	}
	cacheFilePath := pl.catchHandler.CatchFile
	di, err := NewDockerImage(nodeAddress, cacheFilePath)
	if err != nil {
		klog.Errorf("初始化docker客户端失败, err: %s", err)
		panic(err)
	}
	pl.imageHandler = di
}

func (pl *LayerPro) GetImageLayer(imageName string) (ImageMetadata, error) {
	return pl.imageHandler.GetImageLayer(imageName, pl.catchHandler)
}

func (pl *LayerPro) ImageExist(imageName string) (bool, error) {
	return pl.imageHandler.CheckImageExistOnLocal(imageName)
}

func (pl *LayerPro) ComputeLayerScore(images []DockerImageName, nodeName string) (int64, int64) {
	klog.Infof("镜像：%v", images)
	allLocalImages := pl.imageHandler.ListAllLocalImagesInRepo("docker.bnuzh.top")
	// 从缓存中拿到本地所有镜像的层信息
	allLocalImageLayers := pl.getLayers(allLocalImages)
	// 拿到所有pod image 需要的层信息
	podImageLayers := pl.getLayers(images)
	klog.Infof("镜像层: %v", podImageLayers)

	// 获取本地已有的层信息
	layerExist := []LayerMetadata{}
	for _, podLayer := range podImageLayers {
		for _, localLayer := range allLocalImageLayers {
			if podLayer == localLayer {
				layerExist = append(layerExist, podLayer)
				break
			}
		}
	}
	klog.Infof("节点: %s, 已有镜像层: %v", nodeName, layerExist)
	// 本地已有的层总大小
	layerExistSize := ComputeLayerSize(layerExist)
	// pod需要的层总大小
	layerRequestSize := ComputeLayerSize(podImageLayers)
	klog.Infof("节点：%s 已有镜像层总大小: %d  pod所需层总大小: %d  待下载：%d ", nodeName, layerExistSize, layerRequestSize, (layerRequestSize-layerExistSize)/1024/1024)
	return layerExistSize, int64(float64(layerExistSize) / float64(layerRequestSize) * 100 / 2)
}

func (pl *LayerPro) getLayers(images []DockerImageName) []LayerMetadata {
	res := []LayerMetadata{}
	for _, img := range images {
		imageMata, err := pl.catchHandler.Search(img)
		if err != nil {
			klog.Errorf("查找镜像层失败，错误信息: %v", err)
			continue
		}
		res = append(res, imageMata.LayerMetadata...)
	}
	return res
}

func (pl *LayerPro) newCache() error {
	reg, err := NewRegistry(
		"Your Docker Registry Site",
		"Account",
		"Password",
	)
	if err != nil {
		klog.Fatalf("监听器启动失败, err: %s", err)
		return err
	}
	return reg.CreateCatch(pl.catchHandler.CatchFile)
}

func (pl *LayerPro) computeWeight(nodeInfo *framework.NodeInfo, nodeName string, layerSizeMB int64, pod *v1.Pod) float64 {
	config := pl.handle.KubeConfig()
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("初始化集群信息失败, err: %s", err)
		return 1.0
	}
	node, err := kc.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("获取集群信息失败, err: %s", err)
		return 1.0
	}
	podList, err := getAllRunningPod(kc, nodeName)
	if err != nil {
		klog.Errorf("获取当前节点pod信息失败，节点: %s, err: %v", nodeName, err)
	}

	allocatable := node.Status.Allocatable
	allocatedCpu, allocatedMem := getAllocatedResource(allocatable, podList)
	// klog.Infof("当前节点：%s, 剩余cpu: %d, 剩余内存: %d", nodeName, allocatedCpu, allocatedMem)
	// res := allocatedCpu / allocatable.Cpu().MilliValue()
	podReq, _ := resourcehelper.PodRequestsAndLimits(pod)
	// klog.Infof("当前节点：%s, 当前POD request cpu: %d, memory: %d", nodeName, podReq.Cpu().MilliValue(), podReq.Memory().Value())

	OccuCPU := float64(1 - float64(float64(allocatedCpu - podReq.Cpu().MilliValue()) / float64(allocatable.Cpu().MilliValue())))
	OccuMem := float64(1 - float64(float64(allocatedMem - podReq.Memory().Value()) / float64(allocatable.Memory().Value())))
	
	std := math.Abs((OccuCPU - OccuMem) / 2)
	// weight := 1 - std
	klog.Infof("当前节点：%s, 剩余cpu：%d  总CPU：%d  cpu占用: %f  剩余内存: %d  总内存：%d  内存占用：%f  资源标准差: %f", nodeName, allocatedCpu - podReq.Cpu().MilliValue(), allocatable.Cpu().MilliValue(), OccuCPU, allocatedMem - podReq.Memory().Value(), allocatable.Memory().Value(), OccuMem, std)
	
	if layerSizeMB > 10 && std < 0.16 && OccuCPU < 0.6 {
		return 2
	}
	return 0.5
}

func getAllRunningPod(c *kubernetes.Clientset, nodeName string) ([]v1.Pod, error) {
	opts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s,status.phase=%s", nodeName, string(v1.PodRunning)),
	}
	podLists, err := c.CoreV1().Pods("").List(context.TODO(), opts)
	return podLists.Items, err
}

// 返回已分配的cpu和内存
func getAllocatedResource(allocatable v1.ResourceList, podList []v1.Pod) (int64, int64) {
	allocatedCpu, allocatedMem := allocatable.Cpu().MilliValue(), allocatable.Memory().Value()
	for _, po := range podList {
		req, _ := resourcehelper.PodRequestsAndLimits(&po)
		cpuReq, memoryReq := req[v1.ResourceCPU], req[v1.ResourceMemory]
		allocatedCpu -= cpuReq.MilliValue()
		allocatedMem -= memoryReq.Value()
	}
	return allocatedCpu, allocatedMem
}
