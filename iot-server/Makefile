APP_NAME=alphassh
IMAGE_REGISTRY=imrenagi
IMAGE_NAME=$(IMAGE_REGISTRY)/$(APP_NAME)
IMAGE_TAG=$(shell git rev-parse --short HEAD)

.PHONY: build test docker

build: 
	CGO_ENABLED=0 go build -o $(APP_NAME) main.go

test: 
	go test ./... -cover -vet -all


docker-build: 
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

docker-push: 
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):latest
