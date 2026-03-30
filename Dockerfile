FROM scratch
COPY vergeos-exporter /usr/local/bin/vergeos-exporter
EXPOSE 9888
ENTRYPOINT ["/usr/local/bin/vergeos-exporter"]
