IMAGE_REPO=joyme/kubeovs-l2
IMAGE_TAG=dev

all: compile build push

.PHONY: compile
compile:
	mkdir -p bin/
	docker run \
	  -v $(PWD):/go/src/github.com/joyme123/kubeovs-l2 \
	  -w /go/src/github.com/joyme123/kubeovs-l2 \
	  golang:1.12 sh -c '\
	  CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -mod=vendor \
	  -o bin/kubeovs-l2 \
	  github.com/joyme123/kubeovs-l2/cmd/cni'
	docker run \
	  -v $(PWD):/go/src/github.com/joyme123/kubeovs-l2 \
	  -w /go/src/github.com/joyme123/kubeovs-l2 \
	  golang:1.12 sh -c '\
	  CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -mod=vendor \
	  -o bin/kubeovsd \
	  github.com/joyme123/kubeovs-l2/cmd/kubeovsd'
.PHONY: build
build:
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) .
	docker tag $(IMAGE_REPO):$(IMAGE_TAG) $(IMAGE_REPO):latest

.PHONY: push
push:
	docker push $(IMAGE_REPO):$(IMAGE_TAG)
	docker push $(IMAGE_REPO):latest

.PHONY: vagrant
vagrant:
	vagrant up