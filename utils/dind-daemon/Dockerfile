#
# A Dockerfile for a DinD docker daemon. To be run in privileged mode.
# A DinD docker container is attached to each consumer and is used to run the
# submission containers.
#

FROM alpine:edge

RUN apk add --update docker

EXPOSE 2376

ENTRYPOINT ["/usr/bin/dockerd"]
CMD ["--host", "tcp://0.0.0.0:2376"]
