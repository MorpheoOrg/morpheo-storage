Morpheo: Storage API
====================

The Storage API for the [Morpheo](https://morpheoorg.github.io/morpheo/index.html)
platform *receives*, *stores* and *serves*:
 * **Algorithms** as `.tar.gz` files
 * **Datasets**
 * **Models**
 * **Predictions**
 * **Problems** as `.tar.gz` files


Key features
------------

* **RAM-friendly**: uses low-level Golang primitives to stream data directly
  from the request body, to the response body as it comes. Nothing is stored in
  RAM (if using disk storage, the cache memory will be used but that's not a
  problem at all)
* **Self contained**: Golang enables us to ship a statically linked binary (in a
  `FROM scratch` docker image for instance)
* **Simple & Low Level**: written in Golang, simple, and intented to stay so :)

API Endpoints
-------------
The GET requests are pretty simple:

**GET /** - List all the routes

**GET /health** - Service liveness probe

**GET /:resource** - List all the resources (replacing `:resource` by `algo`, `data`, `model`, `prediction` or `problem`)

**GET /:resource/:uuid** - Get a resource by uuid

**GET /:resource/:uuid/blob** - Get a resource blob by uuid



<br>

The POST Requests use a multipart form to send metadata. The last form field should be the BLOB data, because it is streamed directly from the request body. The content should be formatted according to the *multipart/form-data* content type [[RFC2388]](https://www.ietf.org/rfc/rfc2388.txt). You can find below the endpoints with the corresponding form fields:


**POST /algo** - Add a new algo

* `uuid` (optional): uuid of the algo
* `name`: name of the algo
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**POST /data** - Add a new data

* `uuid` (optional): uuid of the data
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**POST /model?:uuid** - Add a new model

A model is linked to an algo by its `:uuid`. Blobs are sent directly in the request body for `/model` (TO BE CHANGED).

<br>

**POST /prediction** - Add a new prediction

* `uuid` (optional): uuid of the prediction
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**POST /problem** - Add a new problem

* `uuid` (optional): uuid of the problem
* `name`: name of the problem
* `description`: description of the problem
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**PATCH /problem** - Patch a new problem

All the following fields are optional:
* `uuid`: uuid of the problem
* `name`: name of the problem
* `description`: description of the problem
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)


Usage: Uploading or retrieving data
-----------------------------------

#### Upload an algo, data, prediction or problem
Examples for a problem with `curl`, assuming storage is running on `localhost:8081`:
```shell
curl -X POST -u user:pass -F uuid=$(uuidgen) -F name=funky_problem -F description=great -F size=666 -F blob=@problem.tar.gz http://localhost:8081/problem
```

#### Upload a model
Examples with `curl`:
```shell
curl --data-binary "@/path/to/model" -u user:password http://localhost:8081/model
```

#### Retrieve a blob of algo, data, model, prediction or problem
Examples with `curl` to retrieve a data:
```shell
curl -u user:pass http://localhost:8081/data/1f01d777-c3f4-4bdd-9c4a-8388860e4c5e/blob > data.hdf5
```

CLI Arguments
-------------

```
Usage of ./storage-api/target/storage-api:

  -host string
      The hostname our server will be listening on (default "0.0.0.0")
  -port int
      The port our compute API will be listening on (default 8000)
  -cert string
      The TLS certs to serve to clients (leave blank for no TLS)
  -key string
      The TLS key used to encrypt connection (leave blank for no TLS)

  -user string
      The username for Basic Authentification
  -password string
      The password for Basic Authentification

  -db-host string
    	The hostname of the postgres database (default: postgres) (default "postgres")
  -db-port int
      The database port (default 5432)
  -db-name string
      The database name (default: morpheo_storage) (default "morpheo_storage")
  -db-user string
      The database user (default: storage) (default "storage")
  -db-pass string
      The database password to use (default: tooshort) (default "tooshort")
  -db-migrations-dir string
    	The database migrations directory (default: /migrations) (default "/migrations")
  -db-rollback
    	if true, rolls back the last migration (default: false)

  -blobstore string
      Storage service provider: 'gc' for Google Cloud Storage, 's3' for AWS S3, 'local' (default) and 'mock' supported"

  -data-dir string
      The directory to store blob data under (default: /data)
      Note that this only applies when using local storage (default "/data")
  -s3-bucket string
      The AWS Bucket for S3 Storage (default: empty string)
  -s3-region string
      The AWS Bucket region for S3 Storage (default: empty string)
  -gc-bucket
      The Google Cloud Storage Bucket (default: empty string)
```

Maintainers
-----------

* Étienne Lafarge <etienne_at_rythm.co>
* Max-Pol Le Brun <maxpol_at_morpheo.co>
