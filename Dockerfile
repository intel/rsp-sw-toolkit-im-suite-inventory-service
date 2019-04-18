FROM alpine

RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
RUN apk --no-cache add zeromq

ADD inventory-service /
HEALTHCHECK --interval=5s --timeout=3s CMD ["/inventory-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/inventory-service"]