Morpheo: Storage API
====================

The `storage` API for the
[Morpheo](https://morpheoorg.github.io/morpheo/modules/introduction.html)
platform.

It receives problems and submission algorithms as `.tar.gz` files including a
`Dockerfile` and **streams** them to disk or to another storage backend (such as
Amazon S3). It also receives, stores and serves datasets that feed ML algorithms
hosted on the Morpheo platform.

Key features
------------

* **RAM-friendly**: uses low-level Golang primitives to stream data directly
  from the request body, to the response body as it comes. Nothing in stored in
  RAM (if using disk storage, the cache memory will be used but that's not a
  problem at all).
* **Self contained**: Golang enables us to ship a statically linked binary (in a
  `FROM scratch` docker image for instance)
* **Simple & Low Level**: written in Golang, simple, and intented to stay so :)

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

API Specification
-----------------

 * `GET /`: lists all the routes
 * `GET /health`: service liveness probe

 * `GET /problem`: problem list
 * `POST /problem`: post a problem
 * `GET /problem/:uuid`: get a problem object by id
 * `GET /problem/:uuid/blob`: get a problem data blob by id

 * `GET /algo`: algo list
 * `POST /algo`: post an algo
 * `GET /algo/:uuid`: get an algo object by id
 * `GET /algo/:uuid/blob`: get an algo data blob by id

 * `GET /data`: dataset list
 * `POST /data`: post a dataset
 * `GET /data/:uuid`: get a dataset object by id
 * `GET /data/:uuid/blob`: get a dataset data blob by id

Uploading or retrieving data
----------------------------

The problem/algo/data blobs are read directly from the request body (for now,
we're not even using multipart form uploads, it might be necessary at some point
though).

### Examples with `curl`

* Posting a problem (assuming storage is running on `localhost:8080`):
```shell
curl --data-binary "@/path/to/problem.tar.gz" -u user:password http://localhost:8080/data
```

* Retrieving a piece of data (assuming storage is running on `localhost:8080`):
```shell
curl -u user:password http://localhost:8080/data/1f01d777-c3f4-4bdd-9c4a-8388860e4c5e/blob > data.hdf5
```

Container Specification
-----------------------

`/problem` and `/algo` routes expect a `.tar.gz` archive, containing a
Dockerfile. Please refer to [the
documentation](https://morpheoorg.github.io/morpheo/) for more information.

Examples can be found [here](https://github.com/MorpheoOrg/hypnogram-wf).

Maintainers
-----------

* Étienne Lafarge <etienne@rythm.co>
* Max-Pol Le Brun <maxpol_at_morpheo.co