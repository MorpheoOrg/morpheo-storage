Dreemcare Common Golang Libraries
=================================

This repository contains Golang code common to all the Golang services of the
DreemCare platform:

 * **Blobstore**: blob storage abstraction (and its local disk and S3
   implementations)
 * **Broker**: broker abstration (and its NSQ implementation)
 * **Container Runtime**: container runtime abstraction (and its `docker`
   implementation).

In addition, a `MultiStringFlag` type has been defined and all the data
structures necessary for the project are defined in this folder
(`data_structures.go`).

Author
------

 * Étienne Lafarge <etienne_at_rythm.co>
