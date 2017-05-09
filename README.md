Morpheo-Compute: a container-oriented Machine-Learning job runner
=================================================================

This repository holds the code for the compute part of the Morpheo project. It
is essentially written in Golang, using the Iris microframework for the API
part and NSQ as a distributed broker.

TL;DR
-----
`compute` receives compute orders (trainings and predictions) from the
`orchestrator`, fetches problem workflows, models and data from `storage` and
executes the orders.

API Spec
--------

The API is dead simple. It consists in 4 routes, two of them being completely
trivial:
 * `GET /`: lists all the routes
 * `GET /health`: service liveness probe
 * `POST /pred`: post a preduplet to this route
 * `POST /learn`: post a learnuplet to this route

The API expects the pred/learn uplets to be posted as JSON strings. Their
structure is described [here](https://morpheoorg.github.io/morpheo-orchestrator/modules/collections.html).

### Example, using HTTPie, assuming the API is reachable under localhost:8085

```shell
http POST http://localhost:8085/learn data=$(uuidgen) id=$(uuidgen) problem=$(uuidgen) algo=$(uuidgen) model=$(uuidgen) status=todo train_data:='["a7f75232-696a-4f8f-bc46-21b4406b903e", "a7f75232-696a-4f8f-bc46-21b4406b903e"]' test_data:='["a7f75232-696a-4f8f-bc46-21b4406b903e", "a7f75232-696a-4f8f-bc46-21b4406b903e"]'
```

Setup locally
-------------

You'll need Golang 1.8 (or newer) to be able to build the project. In addition,
Docker and `doker-compose` (preferably installed via the official Docker repos
if you're running on a distribution whose Docker packages lag behind) will be
required (we're planning to ship a Rkt dev. env. in the coming weeks).

### Mocking storage, the orchestrator and the viewer

 1. Upload a valid problem workflow on storage
 2. Upload a valid model on storage

-----------

* Learnuplets
* Learn Tasks
* Test Tasks
* Preduplets

Key features
------------
* Distributed by design: when a learn-uplet/pred-uplet hits a producer, a list
  of learning tasks (one per chosen learning dataset) is generated.
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


Interfaces
----------

### Inputs

* `learn-uplets/pred-uplets`: DeepSee defines (and stores in the blockchain)
  learn-uplets for learning/training tasks and pred-uplets for predictions.
  These entries are logged into the project's underlying blockchain and -
  whenever a job's done - `dc-compute` notifies (with retrial in case of
  failure) the orchestrator so that it can log the event (along with metadata
  such as the training score). Note that `dc-compute` never writes to or reads
  from the blockchain directly. These things always go through the orchestrator.

* `learn-tasks`: since learning tasks are performed on datasets containing
  themselves multiple records (whatever a record is, as far as `dc-compute`
  knows it could be a timeseries or simply a CSV with data on which learning can
  be performed) and since these records might be available only on specific
  storage clusters, a learning task might start on a cluster named A and then be
  continued on another one, call it B and the end of the learning might be
  performed on A.
  For that reason, a learn-uplet is split into a list of tasks. When task n is
  done, the updated predictor is sent to storage, the task is removed from the
  list and the rest of the list (`[n+1..s]` where `s` is the size of the initial
  list of learning tasks) is sent to the storage cluster that must
  execute the following task. For this list to remain unaltered, a signing
  mechanism will need to be used, probably on the orchestrator's end.

* `dc-compute` retrieves its data from a storage node and gets the symmetric
  decryption keys from `n` different orchestrators. This list of orchestrators
  is sent by `dc-storage` along with the encrypted file. Multi-side HTTPs
  encryption can be used.

### Outputs

* The updated learnuplet/preduplet, sent back by a consumer to the orchestrator.
* The list of remaining tasks, sent to a pool of consumer of a potentially
  different storage cluster. It must be signed (probably by the orchestrator,
  whose public key will be available on the blockchain).

## TODO

* Split the common package into a "core" package and an "api/client" one
* Integration with the viewer (and analytics)
* [configuration management] Use Viper or Cobra ?
* Keep track of the number of retries for each task and enforce it
* Rename this repository to `gomorpheo` and reorganize it a bit ?
