Morpheo: Storage API
====================

The Storage API for the [Morpheo](https://morpheoorg.github.io/morpheo/modules/introduction.html)
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

API Specification
----------------- 

Replacing `object` by `problem`, `algo`, `model` or `data`, the API endpoints are: 

| Method | URI endpoints       |     Action                             |
|:------:| ------------------- | -------------------------------------- |
|  `GET` | /                   | Lists all the routes                   |
|  `GET` | /health             | Service liveness probe                 |
|  `GET` | /:object            | List all the objects                   |
| `POST` | /:object            | Post a new object                      |
|  `GET` | /:object/:uuid      | Get an object by uuid                  |
|  `GET` | /:object/:uuid/blob | Get an object blob by uuid             |


Usage: Uploading or retrieving data
-----------------------------------

#### Upload a data, model or problem
Blobs are sent directly in the request body for `/data`, `/model` and `/problem`.

Examples with `curl`, assuming storage is running on `localhost:8081`:
```shell
curl --data-binary "@/path/to/data.hdf5" -u user:password http://localhost:8081/data
```

#### Upload an algo
Uploading an algo is a bit different, as a multipart upload request is used to send metadata. The request body should be formatted according to the *multipart/form-data* content type [[RFC2388]](https://www.ietf.org/rfc/rfc2388.txt).

The form fields are the following:
 * `name`: name of the algo, should be a non-empty string
 * `size`: size of the blob file in *bytes*
 * `blob`: blob file. Must come **after name and size** in the request body, otherwise it returns an error 400

Examples with `curl`:
```shell
curl -X POST -H "Content-Type: multipart/form-data" -u user:password -F name=funky_algo -F size=120 -F blob=@algo.tar.gz http://localhost:8081/algo
```

#### Retrieve a blob of data, algo, model or problem
Examples with `curl` to retrieve a data:
```shell
curl -u user:password http://localhost:8081/data/1f01d777-c3f4-4bdd-9c4a-8388860e4c5e/blob > data.hdf5
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
