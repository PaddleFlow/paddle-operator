# Paddle Operator 样本缓存组件快速上手

本文档主要讲述了如何安装部署 Paddle Operator 样本缓存组件，并通过实际的示例演示了缓存组件的基础功能。

## 前提条件

* Kubernetes >= 1.8
* kubectl

## 安装缓存组件

1. 安装自定义资源，与 [README](../../README-zh_CN.md) 中描述的步骤一样，如果您使用的是0.4以前的版本，可以再次运行该命令来更新。

创建 PaddleJob / SampleSet / SampleJob 自定义资源,
```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml
```

创建成功后，可以通过以下命令来查看创建的自定义资源
```shell
$ kubectl get crd | grep batch.paddlepaddle.org
NAME                                    CREATED AT
paddlejobs.batch.paddlepaddle.org       2021-08-23T08:45:17Z
samplejobs.batch.paddlepaddle.org       2021-08-23T08:45:18Z
samplesets.batch.paddlepaddle.org       2021-08-23T08:45:18Z
```

2. 部署 Operator，与 [README](../../README-zh_CN.md) 中描述的步骤一样，如果您使用的是0.4以前的版本，可以再次运行该命令来更新。
```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```

3. 部署样本缓存组件的 Controller
```shell
kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/extensions/controllers.yaml
```

通过以下命令可以查看部署好的 Controller
```shell
$ kubectl -n paddle-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
paddle-controller-manager-776b84bfb4-5hd4s   1/1     Running   0          60s
paddle-samplejob-manager-69b4944fb5-jqqrx    1/1     Running   0          60s
paddle-sampleset-manager-5cd689db4d-j56rg    1/1     Running   0          60s
```

4. 安装 CSI 存储插件

目前 Paddle Operator 样本缓存组件仅支持 [JuiceFS](https://github.com/juicedata/juicefs/blob/main/README_CN.md) 作为底层的样本缓存引擎，样本访问加速和缓存相关功能主要由缓存引擎来驱动。

部署 JuiceFS CSI Driver
```shell
kubectl apply -f https://raw.githubusercontent.com/juicedata/juicefs-csi-driver/master/deploy/k8s.yaml
```

部署好 CSI 驱动后，您可以通过以下命令查看状态，Kubernetes 集群中的每一个 worker 节点应该都会有一个 **juicefs-csi-node** Pod。
```shell
$ kubectl -n kube-system get pods -l app.kubernetes.io/name=juicefs-csi-driver
NAME                                          READY   STATUS    RESTARTS   AGE
juicefs-csi-controller-0                      3/3     Running   0          13d
juicefs-csi-node-87f29                        3/3     Running   0          13d
juicefs-csi-node-8h2z5                        3/3     Running   0          13d
juicefs-csi-node-d5grm                        3/3     Running   0          13d
```

注意：如果 Kubernetes 无法发现 CSI 驱动程序，并出现类似这样的错误：**driver name csi.juicefs.com not found in the list of registered CSI drivers**，这是由于 CSI 驱动没有注册到 kubelet 的指定路径，您可以通过下面的步骤进行修复。

在集群中的 worker 节点执行以下命令来获取 kubelet 的根目录
```shell
ps -ef | grep kubelet | grep root-dir
```

在上诉命令打印的内容中，找到 `--root-dir` 参数后面的值，这既是 kubelet 的根目录。然后将以下命令中的 `{{KUBELET_DIR}}` 替换为 kubelet 的根目录并执行该命令。
```shell
curl -sSL https://raw.githubusercontent.com/juicedata/juicefs-csi-driver/master/deploy/k8s.yaml | sed 's@/var/lib/kubelet@{{KUBELET_DIR}}@g' | kubectl apply -f -
```

更多详情信息可参考 [JuiceFS CSI Driver](https://github.com/juicedata/juicefs-csi-driver)

## 缓存组件使用示例

由于 JuiceFS 缓存引擎依靠 Redis 来存储文件的元数据，并支持多种对象存储作为数据存储后端，为了方便起见，本示例使用 Redis 同时作为元数据引擎和数据存储后端。不建议在大数据量的生产环境中使用 Redis 作为数据存储后端，您可以通过这篇文档查看 [JuiceFS 支持的对象存储和设置指南](https://github.com/juicedata/juicefs/blob/main/docs/zh_cn/how_to_setup_object_storage.md)

1. 准备 Redis 数据库

您可以很容易的在云计算平台购买到各种配置的云 Redis 数据库，本示例使用 Docker 在 Kubernetes 集群的 worker 节点上运行一个 Redis 数据库实例。
```shell
docker run -d --name redis \
	-v redis-data:/data \
	-p 6379:6379 \
	--restart unless-stopped \
	redis redis-server --appendonly yes
```


