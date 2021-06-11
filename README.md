# podinfo-operator

## Installing the operator

```bash
make deploy IMG="jkremser/podinfo-operator:v0.0.2"
```
This uses kustomize and deploys all the stuff necessary for the operator.
Its service account gets a cluster role assigned so that the operator can also
watch for events in other namespaces than it's deployed in.

# Usage:

```bash
# create cluster
cat <<EOF | kubectl apply -f -
apiVersion: info.podinfo-operator.io/v1alpha1
kind: Podinfo
metadata:
  name: podinfo-sample
spec:
  backend-replicas: 2
  frontend-replicas: 1
  message: "Hello Podinfo"
EOF
```

This should create two deployments with desired replicas and two services. Consequently, all the changes to the `podinfo-sample` cr should be reflected by the operator.


## Development

building and pushing the image:

```bash
make docker-build docker-push IMG="jkremser/podinfo-operator:v0.0.2"
```

listing the logs:

```bash
make logs
```


Demo:

https://drive.google.com/file/d/1Eb-PN82Z9yEzJCVx9GCPUDY4H9zgSOZ_/view?usp=sharing