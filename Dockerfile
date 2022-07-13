FROM golang:alpine as builder

# RUN mkdir /build
WORKDIR /build
# ADD . /build/
COPY . .

RUN apk add git
<<<<<<< HEAD
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -a -installsuffix cgo -ldflags '-extldflags "-static"' -o blocproxy ./cmd

FROM scratch

COPY --from=builder /build/blocproxy /app/
WORKDIR /app
EXPOSE 62081
CMD ["./blocproxy"]
