Auto-scale temporal worker deployment based on task queue backlog size.

Install keda to k8s cluster
```shell
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace
```

Build and push the docker file using the following commands. Edit the dockerhub username.
```shell
docker build -t  prathyushpv/external-scaler:latest .
docker push prathyushpv/external-scaler:latest
```

Edit the image name in k8s/scaler_deployment.yaml file and run this command to deploy.
```shell
kubectl apply -f ./k8s
```
