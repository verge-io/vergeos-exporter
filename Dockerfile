FROM alpine:3.23.3
RUN apk add --no-cache ca-certificates
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/vergeos-exporter /usr/local/bin/vergeos-exporter
EXPOSE 9888
ENTRYPOINT ["/usr/local/bin/vergeos-exporter"]
