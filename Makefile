DOCKER_IMAGE_NAME ?= kubeadm-backup
DOCKER_REPO       ?= local
DOCKER_IMAGE_TAG  ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

build: build-amd64 build-armv7

build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags '-extldflags "-static"' -o bin/kubeadm-backup-linux-amd64 cmd/kubeadm-backup/main.go

build-armv7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GO111MODULE=on go build -ldflags '-extldflags "-static"' -o bin/kubeadm-backup-linux-armv7 cmd/kubeadm-backup/main.go

docker:
	docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-amd64:$(DOCKER_IMAGE_TAG)" --build-arg ARCH="amd64" --build-arg OS="linux" .
	docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-armv7:$(DOCKER_IMAGE_TAG)" --build-arg ARCH="armv7" --build-arg OS="linux" .

docker-push:
	docker push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-amd64:$(DOCKER_IMAGE_TAG)"
	docker push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-armv7:$(DOCKER_IMAGE_TAG)"

docker-manifest:
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create -a "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-amd64:$(DOCKER_IMAGE_TAG)" "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME)-linux-armv7:$(DOCKER_IMAGE_TAG)"
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
