# Simple MiCoProxy Sidecar Proxy

The code is based off this [repo](https://github.com/ymedialabs/ReverseProxy) and this [talk](https://www.youtube.com/watch?v=tWSmUsYLiE4)

## MiCoProxy versions

The `latest` version of the MiCoProxy docker now stands abandoned. The algorithms are separately deployed as their own algorithms:
- `ratnadeepb/micoproxy:leasttime`
- `ratnadeepb/micoproxy:leastconn`
- `ratnadeepb/micoproxy:random`
- `ratnadeepb/micoproxy:roundrobin`
- `ratnadeepb/micoproxy:rangehash`
- `ratnadeepb/micoproxy:rangehashrounds`

## Building the Proxy

```bash
sudo docker build -t ratnadeepb/micoproxy:p2cgloballb .
sudo docker push ratnadeepb/micoproxy:p2cgloballb
```

## Running the MiCo tool

```bash
kubectl apply -f deployment.yaml
```

## Ports

All incoming requests to the pod are routed to port `62081` and all outgoing requests are sent to port `62082`.

## Install the Redis cluster

Detailed description found [here](https://www.containiq.com/post/deploy-redis-cluster-on-kubernetes)

## Note on commit formatting

We have now adopted the practice of adding a `+` for any commit that adds lines or functionality and a `-` for any commit that removes lines or functionality. However, as of now, this is on a best effort basis.

## Major TODO

TODO: remove all atomics and sync.maps by using svcEndpoints struct to replace Svc2BackendSrvMap
