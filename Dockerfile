FROM golang:1.13.5 AS builder

WORKDIR /go/src/github.com/pytimer/kube-events-watcher
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -ldflags "-w" -o /kube-events-watcher main.go

FROM alpine

WORKDIR /
COPY --from=builder /kube-events-watcher /kube-events-watcher
ENTRYPOINT ["/kube-events-watcher"]
