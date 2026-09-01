FROM alpine:latest

RUN addgroup -S intruder && \
    adduser -S -G intruder intruder && \
    mkdir -p /host-sensitive

COPY read-decoy.sh /usr/local/bin/read-decoy.sh

RUN chmod 0555 /usr/local/bin/read-decoy.sh

USER intruder

ENTRYPOINT ["/usr/local/bin/read-decoy.sh"]