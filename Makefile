# Project name
PROJECT_NAME := poker

# Docker image name
DOCKER_IMAGE_NAME := $(PROJECT_NAME)

# Docker tag
DOCKER_TAG := latest

# Docker image full name
DOCKER_IMAGE := registry.germainleignel.com/personal/poker:latest

# Kubernetes deployment name
K8S_DEPLOYMENT_NAME := poker-deployment

.PHONY: all build push deploy restart

all: build push deploy restart

build:
	docker build -t $(DOCKER_IMAGE_NAME) .

push:
	docker tag $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) $(DOCKER_IMAGE)
	docker push $(DOCKER_IMAGE)

deploy:
	kubectl apply -f k8s/certificate.yaml -n poker
	kubectl apply -f k8s/deployment.yaml -n poker
	kubectl apply -f k8s/service.yaml -n poker
	kubectl apply -f k8s/ingress.yaml -n poker

restart:
	kubectl rollout restart deployment $(K8S_DEPLOYMENT_NAME) -n poker
