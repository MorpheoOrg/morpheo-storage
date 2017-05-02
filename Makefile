default: all

all: iris-api-docker consumer-docker

clean:
	@echo "Dropping the vendor folder"
	rm -rf ./vendor
	@echo "Deleting previous build targets"
	rm -rf ./iris-api/target

deps: glide.yaml glide.lock
	@echo "Pulling dependencies with glide"
	glide install

# TODO: factorize the API and worker builds
iris-api: deps iris-api/main.go
	@echo "Building the Iris compute API (w/ full static linking) using a build container"
	mkdir -p ./iris-api/target
	docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/DeepSee/dc-compute" -v $${PWD}:/usr/local/go/src/github.com/DeepSee/dc-compute:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/iris-api/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o /target/compute-api ./iris-api/main.go"

iris-api-docker: iris-api
	@echo "Building the compute producer Docker image"
	docker build -t compute-producer ./iris-api

consumer: deps consumer/main.go
	@echo "Building the NSQ compute worker (w/ full static linking) using a build container"
	mkdir -p ./consumer/target
	CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o consumer/target/compute-worker consumer/main.go
	# docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/DeepSee/dc-compute" -v $${PWD}:/usr/local/go/src/github.com/DeepSee/dc-compute:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/consumer/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o /target/compute-worker ./consumer/main.go"

consumer-docker: consumer
	@echo "Building the compute worker Docker image"
	docker build -t compute-consumer ./consumer
