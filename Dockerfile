FROM scratch

COPY /target/dc-compute-producer  /dc-compute-producer
ENTRYPOINT ["/dc-compute-producer"]
