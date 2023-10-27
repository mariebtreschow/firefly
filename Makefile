#!make
include .env

# Name of the Docker image
IMAGE_NAME=wordcounter

# Folder name
BINARY_NAME=word-count

# Docker tag for the image
TAG=1.0.0

# Docker build context
BUILD_CONTEXT=.

# Dockerfile location
DOCKERFILE_PATH=./Dockerfile

docker-build:
	@docker build -t $(IMAGE_NAME):$(TAG) -f $(DOCKERFILE_PATH) $(BUILD_CONTEXT)

docker-run:
	@docker run -it --rm $(IMAGE_NAME):$(TAG)

wordcounter:
	go run cmd/$(BINARY_NAME)/main.go

tests:
	go test -v ./...