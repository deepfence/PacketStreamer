# Using on Kubernetes

PacketStreamer can be deployed on Kubernetes using Helm:

```bash
kubectl apply -f ./contrib/kubernetes/namespace.yaml
helm install packetstreamer ./contrib/helm/ --namespace packetstreamer
```

By default, the Helm chart deploys a DaemonSet with sensor on all nodes and one
receiver instance. For the custom configuration values, please refer to the
[values.yaml file](/contrib/helm/values.yaml).
