FROM scratch

ADD build/target /storage
ADD migrations /migrations

ENTRYPOINT ["/storage"]
