default: all

all-bin: compute-api compute-worker storage-api

all-docker: compute-api-docker compute-worker-docker storage-api-docker

clean:
	@echo "Dropping the vendor folder"
	rm -rf ./vendor
	@echo "Deleting previous build targets"
	rm -rf ./compute-api/target

deps: glide.yaml
	@echo "Pulling dependencies with glide"
	glide install

# TODO: factorize the API and worker builds (and storage... eheheheh)
compute-api: deps compute-api/main.go
	@echo "Building the Iris compute API (w/ full static linking) using a build container"
	mkdir -p ./compute-api/target
	docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/MorpheoOrg/go-morpheo" -v $${PWD}:/usr/local/go/src/github.com/MorpheoOrg/go-morpheo:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/compute-api/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o /target/compute-api ./compute-api"

compute-api-docker: compute-api
	@echo "Building the compute producer Docker image"
	docker build -t compute-api ./compute-api

compute-worker: deps compute-worker/main.go
	@echo "Building the NSQ compute worker (w/ full static linking) using a build container"
	mkdir -p ./compute-worker/target
	# go build -o compute-worker/target/compute-worker compute-worker/main.go
	docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/MorpheoOrg/go-morpheo" -v $${PWD}:/usr/local/go/src/github.com/MorpheoOrg/go-morpheo:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/compute-worker/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o /target/compute-worker ./compute-worker"

compute-worker-docker: compute-worker
	@echo "Building the compute worker Docker image"
	docker build -t compute-worker ./compute-worker

storage-api: deps storage-api/main.go
	@echo "Building the Iris storage API (w/ full static linking) using a build container"
	mkdir -p ./storage-api/target
	docker run -u $$UID -it --rm --workdir "/usr/local/go/src/github.com/MorpheoOrg/go-morpheo" -v $${PWD}:/usr/local/go/src/github.com/MorpheoOrg/go-morpheo:ro -v $${PWD}/vendor:/vendor/src -v $${PWD}/storage-api/target:/target:rw golang:1-onbuild bash -c "GOPATH=$$GOPATH:/vendor CGO_ENABLED=0 GOOS=linux go build --installsuffix cgo --ldflags '-extldflags \"-static\"' -o /target/storage-api ./storage-api"

storage-api-docker: storage-api
	@echo "Building the storage producer Docker image"
	docker build -t storage-api ./storage-api
