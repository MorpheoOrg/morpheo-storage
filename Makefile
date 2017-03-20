default: all

all: iris-api-nsq-producer-docker

clean:
	@echo "Dropping the vendor folder"
	rm -rf ./vendor
	@echo "Deleting previous buisd targets"
	rm -rf ./target

deps: glide.yaml glide.lock
	@echo "Pulling dependencies with glide"
	glide install

iris-api-nsq-producer: deps iris-api/main.go
	@echo "Building storage (w/ full static linking) using a build container"
	mkdir -p ./target
	docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/DeepSee/dc-compute" -v $${PWD}:/usr/local/go/src/github.com/DeepSee/dc-compute:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor go build --ldflags '-extldflags \"-static\"' -o /target/dc-compute-producer ./iris-api/main.go"

iris-api-nsq-producer-docker: iris-api-nsq-producer
	@echo "Building the compute producer Docker image"
	docker build -t compute-producer .
