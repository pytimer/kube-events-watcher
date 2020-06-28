.PHONY:	build push

PREFIX = pytimer
IMAGE = kube-events-watcher
TAG = 1.0.0

build:
	docker build --tag ${PREFIX}/${IMAGE}:${TAG} .

push:
	docker push ${PREFIX}/${IMAGE}:${TAG}