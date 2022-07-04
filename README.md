# Simple MiCoProxy Sidecar Proxy

The code is based off this [repo](https://github.com/ymedialabs/ReverseProxy) and this [talk](https://www.youtube.com/watch?v=tWSmUsYLiE4)

## Changing LoadBalancer Policy

Change the `LBPolicy` environment variable for the `micoproxy` container in the deployment yaml.

```yaml
env:
- name: LBPolicy
    value: "MLeastConn"
```

Other options for `LBPolicy` are `Random` and `LeastConn`.

## Building the Proxy

```bash
sudo docker build -t ratnadeepb/blocproxy:latest .
sudo docker push ratnadeepb/blocproxy:latest
```

## Running the BLOC tool

```bash
kubectl apply -f deployment.yaml
```

## Ports

All incoming requests to the pod are routed to port `62081` and all outgoing requests are sent to port `62082`.

## Note on commit formatting

We have now adopted the practice of adding a `+` for a commit that adds a feature add and a `-` for any commit that removes functionality. However, as of now, this is on a best effort basis.

