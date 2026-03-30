# Standalone Dockerfile for running a specific release without GHCR.
# For most users, the published image is simpler:
#   docker pull ghcr.io/verge-io/vergeos-exporter:latest
#
# Build args: VER (e.g., 2.0.0) and ARCH (x86_64 or arm64)

ARG VER=2.0.0
ARG ARCH=x86_64

FROM alpine:3.23.3 AS fetch
ARG VER ARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
RUN set -eux; \
    ASSET="vergeos-exporter_${VER}_Linux_${ARCH}.tar.gz"; \
    BASE="https://github.com/verge-io/vergeos-exporter/releases/download/v${VER}"; \
    curl -fL -o "${ASSET}" "${BASE}/${ASSET}"; \
    curl -fL -o checksums.txt "${BASE}/checksums.txt"; \
    grep " ${ASSET}$" checksums.txt | sha256sum -c -; \
    mkdir -p /out && tar -xzf "${ASSET}" -C /out; \
    install -m 0755 "$(find /out -type f -name vergeos-exporter)" /vergeos-exporter

FROM scratch
COPY --from=fetch /vergeos-exporter /usr/local/bin/vergeos-exporter
EXPOSE 9888
ENTRYPOINT ["/usr/local/bin/vergeos-exporter"]
