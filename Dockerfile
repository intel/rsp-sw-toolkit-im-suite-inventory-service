FROM alpine:3.7 as builder

RUN echo http://nl.alpinelinux.org/alpine/v3.7/main > /etc/apk/repositories; \
    echo http://nl.alpinelinux.org/alpine/v3.7/community >> /etc/apk/repositories
    
RUN apk --no-cache add zeromq util-linux bash


FROM busybox:1.30.1

# ZeroMQ libraries and dependencies
COPY --from=builder /lib/libc.musl-x86_64.so.1 /lib/
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/
COPY --from=builder /usr/lib/libzmq.so.5.1.5 /usr/lib/
COPY --from=builder /usr/lib/libzmq.so.5 /usr/lib/
COPY --from=builder /usr/lib/libsodium.so.23 /usr/lib/ 
COPY --from=builder /usr/lib/libstdc++.so.6 /usr/lib/
COPY --from=builder /usr/lib/libgcc_s.so.1 /usr/lib/
COPY --from=builder /usr/lib/libcrypto.so.42 /usr/lib/
COPY --from=builder /usr/lib/libcrypto.so.42.0.0 /usr/lib/

# Adding bash/lscpu (required by Probabilistic plugin)
COPY --from=builder /bin/bash /bin
COPY --from=builder /lib/libsmartcols.so.1 /lib
COPY --from=builder /usr/lib/libreadline.so.7 /usr/lib/
COPY --from=builder /usr/lib/libncursesw.so.6 /usr/lib/
COPY --from=builder /usr/lib/bash /usr/lib/
COPY --from=builder /usr/bin/lscpu /usr/bin/

ADD inventory-service /
HEALTHCHECK --interval=5s --timeout=3s CMD ["/inventory-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/inventory-service"]