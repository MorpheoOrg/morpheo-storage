DreemCare - Data structures
===========================

Common tags
-----------

These tags could be useful to (automatically at some point?) assemble the data
that a given model deals with. For now, we have a one dimension value "problem"
but it may be relevant to leverage the use of tags for specifying a given
problem ("Let's work on Sleep Apnea using EEG data, let's work on ti using ECG
data). Since the beginning of the "big data" era, tags have proven to be of a
great help in most use cases.

 * `location`: the top-level domain of where data (records/algos) is allowed to
   travel to. If unset, it means virtually anywhere. It can be a comma separated
   list of domains (ex: `gdn.rythm.co`, `deepsee.polytechnique.edu`,
   `predict.aphp.fr`...)
 * `type`: `eeg`, `ecg`, `blood_pressure`...
 * `arrangement`: `continuous`, `sparse` ?

Stored in a (distributed ?) database
------------------------------------

This is just a draft about how the storage project could look like. These
metadata should be shared accross all storage clusters, however the underlying
data/models will be stored only where allowed to. The storage system will
release the encryption keys only if it agrees that the recepient is allowed to
get it (data is free to move/it is asked by a worker belonging to the cluster).

### Predictor

 * `uuid`: the `uuid` of the algorithm
 * `locations`: comma-separated list of top-level domains where to get it from
 * `tags`: all of the aforementionned tags

### Dataset (can be a RAW record, an H5, a CSV file...)

 * `uuid`: the `uuid` of the dataset
 * `locations`: comma-separated list of top-level domains where to get it from
 * `tags`: all of the aforementionned tags

In-transit
----------

### Learn-uplets

 * `uuid`: UUID of the learn-uplet
 * `problem`: UUID of the problem to work on
 * `train_data`: list of UUIDs of data to train on (sequentially!)
 * `test_data`: list of data to test a model against
 * `model`: the model to train
 * `global_perf`: a `float` the cumulated performance increase of the training
   (characterizes the quality of the model)
 * `individual_perf`: list of `float`s characterizing the individual impact of
   each piece of data on the training (characterize the relevance of each
   individual dataset and therefore its **contributivity**)
 * `status`: `TODO|RUNNING|DONE`
 * `individual_statuses`: list of `done|unschedulable|unavailable|failed`
   statuses of each individual train status.
 * Individual test statuses: same for the tests (because the perf. would not be
   representative if the test was performed on only a few datasets)

Do we wanna give a performance to the tests phases too (for data contributivity
only ofc) ?

### Pred-uplets

 * `uuid`: UUID of the pred-uplet
 * `problem`: UUID of the problem to work on
 * `data`: dataset (one record usually) to work on
 * `model`: the model to use for the prediction
 * `request_date`: timestamp at which the prediction was scheduled
 * `response_date`: timestamp of arrival of the prediction
