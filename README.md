Morpheo: Storage API
====================

The Storage API for the [Morpheo](https://morpheoorg.github.io/morpheo/index.html)
platform *receives*, *stores* and *serves*:
 * **Problems** as `.tar.gz` files
 * **Algorithms** as `.tar.gz` files
 * **Models**
 * **Datasets**

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

**GET /:resource** - List all the resources (replacing `:resource` by `problem`, `algo`, `model` or `data`)

**GET /:resource/:uuid** - Get a resource by uuid

**GET /:resource/:uuid/:blob** - Get a resource blob by uuid



<br>

For the POST Requests, a multipart form is used to send metadata. The last form field should be the BLOB data, because it is streamed directly from the request body. The content should be formatted according to the *multipart/form-data* content type [[RFC2388]](https://www.ietf.org/rfc/rfc2388.txt). You can find below the endpoints with the corresponding form fields: 


**POST /problem** - Add a new problem

* `uuid` (optional): uuid of the problem 
* `name`: name of the problem
* `description`: description of the problem
* `owner` (optional): uuid of the owner
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**POST /algo** - Add a new problem

* `uuid` (optional): uuid of the algo 
* `name`: name of the algo
* `owner` (optional): uuid of the owner
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**POST /model?:uuid** - Add a new model

A model is linked to an algo by its `:uuid`. Blobs are sent directly in the request body for `/model` (TO BE CHANGED).

<br>

**POST /data** - Add a new problem

* `uuid` (optional): uuid of the data 
* `owner` (optional): uuid of the owner
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)

<br>

**PATCH /problem** - Patch a new problem

All the following fields are optional:
* `uuid`: uuid of the problem 
* `name`: name of the problem
* `description`: description of the problem
* `owner`: uuid of the owner
* `size`: size of the blob file
* `blob`: blob file (must be the last form field)


Usage: Uploading or retrieving data
-----------------------------------

#### Upload a problem, algo or data
Examples for a problem with `curl`, assuming storage is running on `localhost:8081`:
```shell
curl -X POST -u user:pass -F uuid=$(uuidgen) -F name=funky_problem -F description=great -F size=666 -F blob=@problem.tar.gz http://localhost:8081/problem
```

#### Upload a model
Examples with `curl`:
```shell
curl --data-binary "@/path/to/model" -u user:password http://localhost:8081/model
```

#### Retrieve a blob of data, algo, model or problem
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

  -data-dir string
      The directory to store blob data under (default: /data).
      Note that this only applies when using local storage (default "/data")
  -s3-bucket string
      The AWS Bucket for S3 Storage (default: empty string)
  -s3-region string
      The AWS Bucket region for S3 Storage (default: empty string)
```

Maintainers
-----------

* Étienne Lafarge <etienne_at_rythm.co>
* Max-Pol Le Brun <maxpol_at_morpheo.co>
