FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/vergeos-exporter /usr/local/bin/vergeos-exporter
EXPOSE 9888
ENTRYPOINT ["/usr/local/bin/vergeos-exporter"]
