FROM scratch

ADD build/target /storage-api
ADD migrations /migrations

ENTRYPOINT ["/storage-api"]
