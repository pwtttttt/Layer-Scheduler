# Layer-Scheduler

## Introduction
Layer-aware and Resource-adaptive container Scheduler (**LRScheduler**)  utilizes container **image layer information** to design and implement a node scoring and container scheduling mechanism, which can effectively **reduce download cost** when deploying containers and its **weight dynamically adapts** to enhance load balancing in edge clusters, increasing layer sharing scores when resource load is low to use idle resources effectively.


## Quick Start

1. Make sure the **runtime** of Kubernetes is **Docker**

2. Please upload all the local images and images that will be deployed in containers to **Docker registry**

3. Log in all nodes to the image repository you are using

4. Set the site, account and password of your docker registry in the file named 'main.go'

5. Create a scheduler-config.yaml file and specify the use of the layer-scheduler (LRScheduler) for scheduling

6. Everytime you edit the files in the folder called 'layer', you are supposed to compile ‘main.go’ again by: 
`go build -o kube-scheduler ./main.go`

7. Start layer-scheduler (LRScheduler) which helps you get the logs of images, layers and score by:
`./kube-scheduler --authentication-kubeconfig=/etc/kubernetes/scheduler.conf --authorization-kubeconfig=/etc/kubernetes/scheduler.conf --config=/etc/kubernetes/scheduler-config.yaml`

8. Deploy a Pod and specify in the yaml file that it should use the  layer-scheduler (LRScheduler) for scheduling


## Prerequisites

1. OS-Image：CentOS Linux 7 (Core)
  
3. k8s version：v1.23.8
   
5. Docker version 20.10.8
   
7. go version go1.18 linux/amd64
   
9. Kubelet Version：1.23.8
    
11. Client Version: v1.23.8
    
13. Server Version: v1.23.8
    
15. Container Runtime Version:  docker://20.10.8
