FROM golang:alpine as builder

RUN mkdir /build
WORKDIR /build
ADD . /build/

RUN apk add git
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o micoproxy .

FROM scratch

COPY --from=builder /build/micoproxy /app/
WORKDIR /app
EXPOSE 62081
CMD ["./micoproxy"]