# Paddle Operator

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
$ kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml
```

A succeed creation leads to result as follows,
```shell
$ kubectl get crd
NAME                                    CREATED AT
paddlejobs.batch.paddlepaddle.org       2021-02-08T07:43:24Z
```

Then deploy controller,

```shell
$ kubectl apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```

the ready state of controller would be as follow,
```shell
$ kubectl -n paddle-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
paddle-controller-manager-698dd7b855-n65jr   1/1     Running   0          1m
```

By default, paddle controller runs in namespace *paddle-system* and only controll jobs in that namespace.
To run controller in a different namespace or controll jobs in other namespaces, you can edit `charts/paddle-operator/values.yaml` and install the helm chart.
You can also edit kustomization files or edit `deploy/v1/operator.yaml` directly for that purpose.

### Run demo paddlejob

Deploy your first paddlejob demo with
```shell
$ kubectl -n paddle-system apply -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/examples/wide_and_deep.yaml
```

Check pods status
```shell
$ kubectl -n paddle-system get pods
```

Check paddle job status
```shell
$ kubectl -n paddle-system get pdj
```

### Run demo paddlejob on volcano

Volcano is a batch scheduling system based on kubernetes. Using it for task scheduling results in better performance.Volcano installation, you can refer to [this website](https://volcano.sh/en/docs/installation/).

Edit ctr-Volcano. yaml in the node.

```
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: ctr-volcano
spec:
  minAvailable: 4
  schedulerName: volcano
  policies:
  - event: PodEvicted
    action: RestartJob
  - event: PodFailed
    action: RestartJob
  tasks:
  - replicas: 2
    name: pserver
    template:
      metadata:
        labels:
          paddle-job-pserver: fluid-ctr
      spec:
        imagePullSecrets:
        - name: default-secret
        volumes:
        - hostPath:
            path: /home/work/
            type: ""
          name: seqdata
        containers:
        - image: volcanosh/edlctr:v1
          command:
          - paddle_k8s
          - start_fluid
          imagePullPolicy: IfNotPresent
          name: pserver
          volumeMounts:
          - mountPath: /mnt/seqdata
            name: seqdata
          resources:
            limits:
              cpu: 10
              memory: 30Gi
              ephemeral-storage: 10Gi
            requests:
              cpu: 1
              memory: 100M
              ephemeral-storage: 1Gi
          env:
          - name: GLOG_v
            value: "0"
          - name: GLOG_logtostderr
            value: "1"
          - name: TOPOLOGY
            value: ""
          - name: TRAINER_PACKAGE
            value: /workspace
          - name: NAMESPACE
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.namespace
          - name: POD_IP
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: POD_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.name
          - name: PADDLE_CURRENT_IP
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: PADDLE_JOB_NAME
            value: fluid-ctr
          - name: PADDLE_IS_LOCAL
            value: "0"
          - name: PADDLE_TRAINERS_NUM
            value: "2"
          - name: PADDLE_PSERVERS_NUM
            value: "2"
          - name: FLAGS_rpc_deadline
            value: "36000000"
          - name: ENTRY
            value: cd /workspace/ctr && python train.py --is_local 0 --cloud_train 1
          - name: PADDLE_PORT
            value: "30236"
          - name: LD_LIBRARY_PATH
            value: /usr/local/lib:/usr/local/nvidia/lib64:/usr/local/rdma/lib64:/usr/lib64/mlnx_ofed/valgrind
          - name: PADDLE_TRAINING_ROLE
            value: PSERVER
          - name: TRAINING_ROLE
            value: PSERVER
        restartPolicy: OnFailure
  - replicas: 2
    policies:
    - event: TaskCompleted
      action: CompleteJob
    name: trainer
    template:
      metadata:
        labels:
          paddle-job: fluid-ctr
      spec:
        imagePullSecrets:
        - name: default-secret
        volumes:
        - hostPath:
            path: /home/work/
            type: ""
          name: seqdata
        containers:
        - image: volcanosh/edlctr:v1
          command:
          - paddle_k8s
          - start_fluid
          imagePullPolicy: IfNotPresent
          name: trainer
          volumeMounts:
          - mountPath: /mnt/seqdata
            name: seqdata
          resources:
            limits:
              cpu: 10
              memory: 30Gi
              ephemeral-storage: 10Gi
            requests:
              cpu: 1
              memory: 100M
              ephemeral-storage: 10Gi
          env:
          - name: GLOG_v
            value: "0"
          - name: GLOG_logtostderr
            value: "1"
          - name: TOPOLOGY
          - name: TRAINER_PACKAGE
            value: /workspace
          - name: CPU_NUM
            value: "2"
          - name: NAMESPACE
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.namespace
          - name: POD_IP
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: POD_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.name
          - name: PADDLE_CURRENT_IP
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: PADDLE_JOB_NAME
            value: fluid-ctr
          - name: PADDLE_IS_LOCAL
            value: "0"
          - name: FLAGS_rpc_deadline
            value: "36000000"
          - name: PADDLE_PORT
            value: "30236"
          - name: PADDLE_PSERVERS_NUM
            value: "2"
          - name: PADDLE_TRAINERS_NUM
            value: "2"
          - name: PADDLE_TRAINING_ROLE
            value: TRAINER
          - name: TRAINING_ROLE
            value: TRAINER
          - name: LD_LIBRARY_PATH
            value: /usr/local/lib:/usr/local/nvidia/lib64:/usr/local/rdma/lib64:/usr/lib64/mlnx_ofed/valgrind
          - name: ENTRY
            value: cd /workspace/ctr && python train.py --is_local 0 --cloud_train 1
        restartPolicy: OnFailure

```

Deployment of job.

```
kubectl apply -f ctr-volcano.yaml
```

Query if the job is running properly.

```
kubectl get pods | grep ctr-volcano
```

You can select a PServer task to view the log.

```
kubectl logs ctr-volcano-pserver-0
```

Select a Tariner task to view the log.

```
kubectl logs ctr-volcano-trainer-0
```

### Uninstall

Simply
```shell
$ kubectl delete -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/crd.yaml -f https://raw.githubusercontent.com/PaddleFlow/paddle-operator/main/deploy/v1/operator.yaml
```
## Advanced usage

More configuration can be found in Makefile, clone this repo and enjoy it.
If you have any questions or concerns about the usage, please do not hesitate to contact us.

## More Information

Please refer to the
[中文文档](https://fleet-x.readthedocs.io/en/latest/paddle_fleet_rst/paddle_on_k8s.html) 
for more information about paddle configuration.
