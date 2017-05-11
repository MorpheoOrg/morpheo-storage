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

### Launching locally after a build, using `docker-compose`

```shell
make all-bin && STORAGE_PORT=8081 COMPUTE_PORT=8082 docker-compose up -d --build
```

Key features
------------
* Distributed by design: the API (producer)
* Cloud Native: even though running `dc-compute` on classical architectures is
  possible and easy too, it is really aimed at being ran in a Cloud-Native
  environment. DC-Compute's task producers and consumers are **stateless** and
  can therefore be scaled horizontally with no side effects. The broker between
  them however, isn't (since its purpose is to store the system's state that
  would be tricky to achieve :) ).
* Broker-agnostic: even though only RabbitMQ is supported for now, the borker
  layer has been completely abstracted.
* Computation-backend agnostic: three backend are planned for now: Kubernetes,
  Docker and Rkt (directly on the worker's host). As long as consumers can run
  shell commands and put the predictor/training data as well as retrieve the
  learning/prediction artifact file on it, a new computation backend is
  possible.
* Secure: The ML jobs are, for the training part at least, code we don't
  necessarily have access to. Therefore, it is run in containers that have a
  very limited access to the host machine (strong `seccomp` restrictions are
  applied on the running container, it has no network interface...).

## TODO

* Integration with the viewer (and analytics)
* [configuration management] Use Viper or Cobra ?
* Retry policies for our tasks depending on the source of the error

Maintainers
-----------
* Étienne Lafarge <etienne@rythm.co>
