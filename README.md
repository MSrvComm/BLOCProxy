# Simple MiCoProxy Sidecar Proxy

The code is based off this [repo](https://github.com/ymedialabs/ReverseProxy) and this [talk](https://www.youtube.com/watch?v=tWSmUsYLiE4)

## Building the Proxy

```bash
sudo docker build -t ratnadeepb/micoproxy:latest .
sudo docker push ratnadeepb/micoproxy:latest
```

## Building the init-container

```bash
sudo docker build -t ratnadeepb/micoinit -f Dockerfile.init .
sudo docker push ratnadeepb/micoinit:latest
```

## Running the MiCo tool

```bash
kubectl apply -f deployment.yaml
```

## Ports

All incoming requests to the pod are routed to port `62081` and all outgoing requests are sent to port `62082`.

## Note on commit formatting

We have now adopted the practice of adding a `+` for any commit that adds lines or functionality and a `-` for any commit that removes lines or functionality. However, as of now, this is on a best effort basis.
