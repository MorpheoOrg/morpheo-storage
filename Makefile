# (Containerized) build commands
BUILD_CONTAINER = \
  docker run -u $(shell id -u) -it --rm \
	  --workdir "/usr/local/go/src/github.com/MorpheoOrg/go-morpheo" \
	  -v $${PWD}:/usr/local/go/src/github.com/MorpheoOrg/go-morpheo:ro \
	  -v $${PWD}/vendor:/vendor/src \
	  -e GOPATH="/go:/vendor" \
	  -e CGO_ENABLED=0 \
	  -e GOOS=linux

GLIDE_CONTAINER = \
	docker run -it --rm \
	  --workdir "/usr/local/go/src/github.com/MorpheoOrg/go-morpheo" \
	  -v $${PWD}:/usr/local/go/src/github.com/MorpheoOrg/go-morpheo \
		$(BUILD_CONTAINER_IMAGE)

BUILD_CONTAINER_IMAGE = golang:1-onbuild

GOBUILD = go build --installsuffix cgo --ldflags '-extldflags \"-static\"'

# User defined variables (use env. variables to override)
DOCKER_REPO ?= registry.morpheo.io
DOCKER_TAG ?= $(shell git rev-parse --verify --short HEAD)

# Prerequisite variables & intermediate variables
LIBS = client common
LIBSOURCES = $(foreach LIB,$(LIBS),$(wildcard $(LIB)/*.go))

# Targets (files & phony targets)
PROJECTS = compute-api compute-worker storage-api
BIN_CLEAN_TARGETS = $(foreach PROJECT,$(PROJECTS),$(PROJECT)-clean)
DOCKER_IMAGES_TARGETS = $(foreach PROJECT,$(PROJECTS),$(PROJECT)-docker)
DOCKER_IMAGES_CLEAN_TARGETS = $(foreach PROJECT,$(PROJECTS),$(PROJECT)-docker-clean)

# Target configuration
.DEFAULT: all
.PHONY: all clean all-bin all-bin-clean all-docker all-docker-clean \
	      devenv-start devenv-clean vendor-clean \
				$(PROJECTS) $(BIN_CLEAN_TARGETS) \
	      $(DOCKER_IMAGES_TARGETS) $(DOCKER_IMAGES_CLEAN_TARGETS)

# Project wide targets
all: all-bin all-docker
clean: vendor-clean all-bin-clean all-docker-clean devenv-clean

## Binary targets
all-bin: $(PROJECTS)
all-bin-clean: $(BIN_CLEAN_TARGETS)

## Docker targets
all-docker: $(DOCKER_IMAGES_TARGETS)
all-docker-clean: $(DOCKER_IMAGES_CLEAN_TARGETS)

## Development environment build, launch & teardown targets
devenv-start: all-bin
	STORAGE_PORT=8081 COMPUTE_PORT=8082 ORCHESTRATOR_PORT=8083 NSQ_ADMIN_PORT=8085 docker-compose up -d --build
devenv-clean:
	STORAGE_PORT=8081 COMPUTE_PORT=8082 ORCHESTRATOR_PORT=8083 NSQ_ADMIN_PORT=8085 docker-compose down
devenv-logs:
	STORAGE_PORT=8081 COMPUTE_PORT=8082 ORCHESTRATOR_PORT=8083 NSQ_ADMIN_PORT=8085 docker-compose logs --follow storage compute compute-worker orchestrator dind-executor

# Dependency-related rules
vendor: glide.yaml
	@echo "Pulling dependencies with glide... in a build container too"
	mkdir ./vendor
	$(GLIDE_CONTAINER) bash -c \
		"go get github.com/Masterminds/glide && glide install && chown $(shell id -u):$(shell id -g) -R ./glide.lock ./vendor"
vendor-clean:
	@echo "Dropping the vendor folder"
	rm -rf ./vendor

# Binary build rules & phony aliases
%/build/target: %/*.go vendor $(LIBSOURCES)
	echo "$@ $^"
	mkdir -p $${PWD}/$(@D)
	$(BUILD_CONTAINER) -v $${PWD}/$(@D):/build:rw $(BUILD_CONTAINER_IMAGE) \
		$(GOBUILD) -o /build/target ./$(dir $<)

$(PROJECTS):
	@echo "Building $(@) binary"
	$(MAKE) $(@)/build/target

$(BIN_CLEAN_TARGETS):
	@echo "Removing $(subst -clean,,$(@))/build directory"
	rm -rf $(subst -clean,,$(@))/build

# Docker image build rules & phony aliases
$(DOCKER_IMAGES_TARGETS): %-docker: %/build/target
	@echo "Building the $(DOCKER_REPO)/$(subst -docker,,$(@)):$(DOCKER_TAG) Docker image"
	docker build -t $(DOCKER_REPO)/$(subst -docker,,$(@)):$(DOCKER_TAG) \
	  ./$(subst -docker,,$(@))

$(DOCKER_IMAGES_CLEAN_TARGETS):
	@echo "Deleting the $(DOCKER_REPO)/$(subst -docker,,$(@)):$(DOCKER_TAG) Docker image"
	docker rmi $(DOCKER_REPO)/$(subst -docker,,$(@)):$(DOCKER_TAG) || \
		echo "No docker image to remove"
