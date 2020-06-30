.PHONY:	build push

PREFIX = pytimer
IMAGE = kube-events-watcher
TAG = 1.0.0

docker-build:
	docker build --tag ${PREFIX}/${IMAGE}:${TAG} .

push:
	docker push ${PREFIX}/${IMAGE}:${TAG}

build:
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -o ./kube-events-watcher main.go
