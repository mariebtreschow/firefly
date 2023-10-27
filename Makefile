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

.PHONY: docker-build

docker-build:
	@docker build -t $(IMAGE_NAME):$(TAG) -f $(DOCKERFILE_PATH) $(BUILD_CONTEXT)

wordcounter:
	@env $(shell grep -v '^#' .env | xargs) go run cmd/$(BINARY_NAME)/main.go