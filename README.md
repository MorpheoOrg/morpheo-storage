Morpheo-Compute: a container-oriented Machine-Learning job runner
=================================================================

This repository holds the code for the compute part of the Morpheo project. It
is essentially written in Golang, using the Iris microframework for the API
part and NSQ as a distributed broker.

TL;DR
-----
* `client`: Golang API client for the `orchestrator` and `storage`
* `common`: data structure definitions and common interfaces and types
  (container runtime backend, blob store backend, broker backend...)
* `compute-api` receives compute orders (trainings and predictions) from the
  `orchestrator` and puts them in a task queue
* `compute-worker` gets tasks from the above queue, fetches problem workflows,
  models and data from `storage` and executes learning/training orders.
* `storage-api` handle problem, algorithm and data uploads, storage and download
* `utils/dind-daemon` defines an alpine based docker image running the Docker
  daemon. The `compute-worker` runs its containers (problem workflow &
  algorithm) in this "Docker in Docker" container.

Local dev. environment
----------------------

### Requirements

* `docker`, `docker-compose` and `make` (we're using docker containers to build
  and run our Golang services)

### Building

* All services' binaries: `make all-bin`
* A given service binary: `make [compute-api|compute-worker|storage-api]`
  (binaries are put under `./<service-name>/target/<service-name>`)
* All `docker` images `make all-docker`
* A given image: `make [compute-api-docker|compute-worker-docker|storage-api-docker]`

### Build and run local dev. environment

```shell
make dev
```

This launches the `compute` API on port `8081`, the `storage` API on port `8082`
and the NSQ broker admin interface on port `8085`.

You can simply run this everytime you change some Go code and your dev. env.
should be automatically updated :)

## TODO

* Integration with the viewer (and analytics)
* [configuration management] Use Viper or Cobra ?
* Complete our mock suite and write an extensive unit test suite

Maintainers
-----------
* Étienne Lafarge <etienne@rythm.co>
