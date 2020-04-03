# Examples

The [server](server/) and [client](client/) examples showcase federated service
discovery using the service mesh federation - resource discovery protocol.

## Usage

1\. Generate root, server, and client certificates:

```console
$ make certs
```

2\. Create kind cluster:

Save following as `kind.config.yaml`:

```console
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
# - role: worker
- role: control-plane
- role: worker
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 8001
    hostPort: 8001
    protocol: TCP
```

Create kind cluster: 

```console
$ kind create cluster --config=kind.config.yaml
```

3\. Generate certs/secrets:

```console
$ make certs
```

```console
$ kubectl create secret generic gimletcerts --from-file=./rootCA.crt --from-file=server.crt --from-file=server.key -n gimlet  
```

4\. Deploy server bits to cluster:

```console
$ kubectl apply -f deployment
$ kubectl apply -f deployment/server
```

4\. Start the client on local machine:

```console
$ make start-client
```

5\. Hit ^c (ctrl + c) to terminate the server/client.