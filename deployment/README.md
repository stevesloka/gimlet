# Deploying Gimlet

## Create Certs

1. First generate the certs using the helper `Makefile` task:
```bash
$ make certs
```

2. Create Kubernetes secret based upon the certs generated from previous step:
```bash
$ kubectl create secret generic gimletcerts --from-file=./rootCA.crt --from-file=server.crt --from-file=server.key -n gimlet 
```