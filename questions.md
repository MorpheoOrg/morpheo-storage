Questions and remarks for Camille and Mathieu
=============================================

Blocking
--------

* Learn or train ? Do we normalize the terms we use to refer to ML concepts ?

To meditate upon
----------------

* Do we offer the ability for machine learners to force the execution of their
  algorythm on a given cluster as well ? Or do we keep this notion of location
  only for datasets ? Even if we do so, machine learners are driven to allow
  execution of their code anywhere because learning on more data means getting
  better prediction scores and therefore, more money.

* What metric do we use for the compute workers' contributivity ? The sum of the
  sizes of the input dataset that were processed ? Note that the time spent
  computing would be a very bad metric: it encourages people to run workers on
  slow hardware to increase the computing time. What we want is the opposite:
  have people buy expensive GPU hardware to run computations faster and at a
  lower cost. Note that we must somehow make sure that these prices will be fair
  for people running their infrastructure in the cloud (like Rythm is gonna do
  :) ), we can probably align ourselves with prices on AWS.

* Retrieving chunks of the decryption keys from orchestrators: the orchestrator
  will have to verify that the target is allowed to decrypt that data by
  analyzing it's DB, the blockchain ultimately. How do we do that ?

* It is essential that we use HTTPs everywhere. In a nutshell, this is an
  out-of-the box way for a service A to verify the identity of another service B
  when it talks with it: the sender/recipient can't be faked, transmitted data
  can't be altered by a "man-in-the-middle". Traditionnaly, when internal
  services communicate together in the same cluster, no encryption occurs. We'll
  have to explain everyone how to do that, deploy an internal certificate
  authority on every cluster (Hashicorp's Vault seems to be a great option for
  that) for communications that need to be internal (between services and their
  DBs, their Brokers, themselves...). In general, when exposed services (the 4/5
  ones we already listed) talk together, it's better to just use the external
  HTTPs endpoint even if they are on the same cluster while making sure routes
  are configured so that this traffic doesn't go out of the cluster, because it
  would cost additionnal money.
