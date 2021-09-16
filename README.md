# Paddle Operator

English | [简体中文](./README-zh_CN.md)

## Overview

Paddle Operator makes it easy to run [paddle](https://www.paddlepaddle.org.cn/)
distributed training job on kubernetes by providing PaddleJob custom resource etc.

## Quick Start
### Prerequisites

* Kubernetes >= 1.8
* kubectl

### Installation

With kubernetes ready, you can install paddle operator with configuration in *deploy* folder 
(use *deploy/v1* for kubernetes v1.16+ or *deploy/v1beta1* for kubernetes 1.15-).

Create PaddleJob crd,
```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml
```

A succeed creation leads to result as follows,
```shell
kubectl get crd
NAME                                    CREATED AT
paddlejobs.batch.paddlepaddle.org       2021-02-08T07:43:24Z
```

Then deploy controller,

```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```

the ready state of controller would be as follow,
```shell
kubectl -n paddle-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
paddle-controller-manager-698dd7b855-n65jr   1/1     Running   0          1m
```

By default, paddle controller runs in namespace *paddle-system* and only controls jobs in that namespace.
To run controller in a different namespace or controll jobs in other namespaces, you can edit `charts/paddle-operator/values.yaml` and install the helm chart.
You can also edit kustomization files or edit `deploy/v1/operator.yaml` directly for that purpose.

### Run demo paddlejob

Deploy your first paddlejob demo with
```shell
kubectl -n paddle-system apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/examples/wide_and_deep.yaml
```

Check pods status
```shell
kubectl -n paddle-system get pods
```

Check paddle job status
```shell
kubectl -n paddle-system get pdj
```

### Work with Volcano

Enable volcano before installation, add the following args in *deploy/v1/operator.yaml*
```
containers:
- args:
  - --leader-elect
  - --namespace=paddle-system  # watch this ns only
  - --scheduling=volcano       # enable volcano
  command:
  - /manager
```

then, job as in *deploy/examples/wide_and_deep_volcano.yaml* can be handled correctly.

### Elastic Trainning

Elastic feature depend on etcd present, which should be set for controller as args,
```
  --etcd-server=paddle-elastic-etcd.paddle-system.svc.cluster.local:2379      # enable elastic
```

then, job as in *deploy/elastic/resnet.yaml* can be handled correctly.

### Data Caching and Acceleration

Inspired by the [Fluid](https://github.com/fluid-cloudnative/fluid), we have implemented data cache component in this project, aims to cache sample data locally in the Kubernetes cluster and accelerate the execution efficiency of PaddleJob.

**Features**：

- __Accelerate PaddleJobs to Acquire Sample Data__

  Paddle Operator provides data acceleration for PaddleJob by using [JuiceFS](https://github.com/juicedata/juicefs) as the cache engine, especially in the scene of massive and small files, which can be significantly improved.

- __Co-Orchestration for Sample Data and PaddleJobs__

  After the SampleSet is created, the cache component will automatically warm up the sample data to the Kubernetes cluster. When this SampleSet is needed for subsequent training jobs, the cache component can schedule the training jobs to nodes with cache, greatly shortening the PaddleJob Execution time, and sometimes it can also improve GPU resource utilization.

- __Support Multiple Data Management Operations__
  
  The cache component provides multiple data management operations through the SampleJob custom resources, including a sync job that synchronizes remote sample data to the cache engine, a warmup job for data prefetch, a clear job for clearing cached data, and a rmr job for removing unused data in cache engine.
 
More information about the cache component, please refer to [extensions](./docs/zh_CN/ext-overview.md)

### Uninstall
Simply
```shell
kubectl delete -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```
## Advanced usage

More configuration can be found in Makefile, clone this repo and enjoy it.
If you have any questions or concerns about the usage, please do not hesitate to contact us.

## More Information

Please refer to the
[Fleex](https://fleet-x.readthedocs.io/en/latest/paddle_fleet_rst/paddle_on_k8s.html) 
for more information about paddle configuration, and refer to the [API docs](./docs/en/api_doc.md) for more information custom resource definition.
