FROM alpine:3.7

RUN echo http://nl.alpinelinux.org/alpine/v3.7/main > /etc/apk/repositories; \
    echo http://nl.alpinelinux.org/alpine/v3.7/community >> /etc/apk/repositories
    
RUN apk --no-cache add zeromq util-linux bash

ADD inventory-service /
HEALTHCHECK --interval=5s --timeout=3s CMD ["/inventory-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/inventory-service"]