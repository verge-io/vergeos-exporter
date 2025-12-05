# Downloads the prebuilt VergeOS exporter and bakes it into a tiny image.
# Build args: VER (e.g., 1.1.9) and ARCH (x86_64 or arm64)

ARG VER=1.1.9
ARG ARCH=x86_64

FROM debian:bookworm-slim AS fetch
ARG VER ARCH
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl tar findutils && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /tmp
RUN set -eux; \
    ASSET="vergeos-exporter_${VER}_Linux_${ARCH}.tar.gz"; \
    BASE="https://github.com/verge-io/vergeos-exporter/releases/download/v${VER}"; \
    echo "Fetching ${ASSET}"; \
    curl -fL -o "${ASSET}" "${BASE}/${ASSET}"; \
    curl -fL -o checksums.txt "${BASE}/checksums.txt"; \
    # verify only our asset
    grep " ${ASSET}\$" checksums.txt > sums.txt; \
    sha256sum -c sums.txt; \
    # unpack and install
    mkdir -p /out; \
    tar -xzf "${ASSET}" -C /out; \
    BIN="$(find /out -type f -name vergeos-exporter | head -n1)"; \
    test -n "$BIN"; \
    install -m 0755 "$BIN" /usr/local/bin/vergeos-exporter

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=fetch /usr/local/bin/vergeos-exporter /usr/local/bin/vergeos-exporter
USER 65532:65532
EXPOSE 9888
ENTRYPOINT ["/usr/local/bin/vergeos-exporter"]
