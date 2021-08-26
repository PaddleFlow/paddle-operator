# Paddle Operator

[English](./README.md) | 简体中文

## 概述

Paddle Operator makes it easy to run [paddle](https://www.paddlepaddle.org.cn/)
distributed training job on kubernetes by providing PaddleJob custom resource etc.

## 快速上手
### 前提条件

* Kubernetes >= 1.8
* kubectl

### 安装

配置好 Kubernetes 集群后, 您可以使用 *deploy* 文件夹下的 yaml 配置文件来安装 Paddle Operator。
(对于 kubernetes v1.16+ 的集群使用 *deploy/v1* 的配置文件， kubernetes 1.15- 的集群使用 *deploy/v1beta1*)。

创建 PaddleJob 自定义资源,
```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml
```

创建成功后，可以通过以下命令来查看创建的自定义资源
```shell
kubectl get crd
NAME                                    CREATED AT
paddlejobs.batch.paddlepaddle.org       2021-02-08T07:43:24Z
```

然后可以通过下面的命令来部署 controller

```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```

通过以下命令可以查看部署在集群中的 controller
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

### 使用Volcano调度任务

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

### 弹性训练

Elastic feature depend on etcd present, which should be set for controller as args,
```
  --etcd-server=paddle-elastic-etcd.paddle-system.svc.cluster.local:2379      # enable elastic
```

then, job as in *deploy/elastic/resnet.yaml* can be handled correctly.

### 数据缓存与加速

受到 [Fluid](https://github.com/fluid-cloudnative/fluid) 项目的启发，Paddle Operator里添加了样本缓存组件，旨在将远程的样本数据缓存到训练集群本地，加速 PaddleJob 作业的执行效率。

样本缓存组件目前支持的功能有：

- __加速训练任务摄取样本数据__

    Paddle Operator的样本缓存组件使用 [JuiceFS](https://github.com/juicedata/juicefs) 作为缓存引擎，能够加速远程样本的访问速度，尤其是在海量小文件的场景模型训练任务能有显著的提升。

- __缓存亲和性调度__

    创建好样本数据集后，缓存组件会自动的将样本数据预热到训练集群中，当后续有训练任务需要使用这个样本集时，缓存组件能够将训练任务调度到有缓存的节点，大大缩短 PaddleJob 执行时间，一定程度也能提高 GPU 资源利用率。

- __支持多种数据管理作业__

    缓存组件通过 SampleJob 自定义资源，为用户提供了多种数据集管理命令，包括将远程样本数据同步到缓存引擎的 sync 作业，用于数据预热的 warmup 作业，清理缓存数据的 clear 作业 和 淘汰历史数据的 rmr 作业。

更多关于样本缓存组件的信息请参考[扩展功能](./docs/zh_CN/ext-toc.md)

### 卸载

卸载 Paddle Operator 前，请确保命名空间 paddle-system 下的 PaddleJob 都被清理掉了。

```shell
kubectl delete -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```

## 高级配置

More configuration can be found in Makefile, clone this repo and enjoy it.
If you have any questions or concerns about the usage, please do not hesitate to contact us.

## 更多资料

关于在 Kubernetes 集群使用 Paddle Operator 做分布式训练的更多示例可以参考 [FleetX](https://fleet-x.readthedocs.io/en/latest/paddle_fleet_rst/paddle_on_k8s.html)
