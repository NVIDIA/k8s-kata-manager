# K8s-Kata-manager

## Introduction

The ``k8s-kata-manager`` project provides tools for configuring Kata runtime classes with OCI container runtimes on Kubernetes.

It allows administrators to declaratively define what runtime classes they would like to have available in their cluster.
The ``k8s-kata-manager``, running on a worker node,  will download artifacts associated with each runtime class, add the
runtime class to the container runtime configuration, and restart the runtime.

As an example, consider the following configuration file:

```
artifactsDir: /opt/nvidia/artifacts/runtimeclasses
runtimeClasses:
  - name: kata-qemu-nvidia-gpu
    artifacts:
      url: stg.nvcr.io/nvidia/cloud-native/kata-gpu-artifacts:ubuntu22.04-525
      pullSecret: my-pull-secret
```

A runtime class, named **kata-qemu-nvidia-gpu**, will be added to the OCI container runtime on the node. Artifacts
associated with this kata runtime class will be pulled from the specified URL and be placed on the local filesystem
under *artifactsDir*.

## Kubernetes Deployment

Below are instructions on how to build and test the k8s-kata-manager in Kubernetes. In the Kubernetes deployment,
the k8s-kata-manager will run as a Daemonset. It will read its configuration from a ConfigMap that is created
as a prerequisite.

Build the container image:

```bash
make build-image
```

[Optional] Create a secret needed to pull the Kata artifacts:

```bash
kubectl create secret docker-registry <my-k8s-secret> --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>
```

Modify the example ConfigMap as needed. Then create the ConfigMap:

```bash
kubectl create configmap kata-config --from-file=./example/config/configmap.yaml
```

Modify the example daemonset as needed. Then deploy k8s-kata-manager:

```bash
kubectl apply -f example/daemonset/
```

Create the Kubernetes RuntimeClass object:

```bash
kubectl apply -f example/runtimeclass/runtimeclass.yaml
```

Example output:
```bash
$ kubectl apply -f ./example/daemonset/
daemonset.apps/k8s-kata-manager created
serviceaccount/kata-manager-sa created
role.rbac.authorization.k8s.io/kata-manager-role created
rolebinding.rbac.authorization.k8s.io/kata-manager-role-binding created

$ kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
k8s-kata-manager-ncl7f   1/1     Running   0          12s

// kata artifacts associated with the runtime class get pulled
$ ls -ltr /opt/nvidia-gpu-operator/artifacts/runtimeclasses/kata-qemu-nvidia-gpu/
total 792924
-rw-r--r-- 1 root root   6636272 Jun  1 22:54 vmlinuz-nvidia-gpu.container
-rw-r--r-- 1 root root 805306368 Jun  1 22:54 kata-ubuntu-jammy-nvidia-gpu.image
-rw-r--r-- 1 root root      2464 Jun  1 22:54 configuration-nvidia-gpu-qemu.toml

// the following entry gets added to the containerd configuration file
$ cat /etc/containerd/config.toml
. . .
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-qemu-nvidia-gpu]
          runtime_type = "io.containerd.kata-qemu-nvidia-gpu.v2"
          privileged_without_host_devices = true
          pod_annotations = ["io.katacontainers.*"]

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-qemu-nvidia-gpu.options]
            ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration-qemu-nvidia-gpu.toml"
. . .
```

## Local testing

A CLI tool can be used to pull artifacts and configure runtime classes on a node locally.
This tool is still under development.

