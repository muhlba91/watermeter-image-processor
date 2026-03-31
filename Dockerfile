FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

ARG CI_COMMIT_TIMESTAMP
ARG CI_COMMIT_SHA
ARG CI_COMMIT_TAG

LABEL org.opencontainers.image.authors="Daniel Muehlbachler-Pietrzykowski <daniel.muehlbachler@niftyside.com>"
LABEL org.opencontainers.image.vendor="Daniel Muehlbachler-Pietrzykowski"
LABEL org.opencontainers.image.source="https://github.com/muhlba91/watermeter-image-processor"
LABEL org.opencontainers.image.created="${CI_COMMIT_TIMESTAMP}"
LABEL org.opencontainers.image.title="watermeter-image-processor"
LABEL org.opencontainers.image.description="Processor for watermeter images, which extracts the watermeter reading and sends it MQTT for Home Assistant."
LABEL org.opencontainers.image.revision="${CI_COMMIT_SHA}"
LABEL org.opencontainers.image.version="${CI_COMMIT_TAG}"

USER 20000:20000
COPY --chmod=555 watermeter-image-processor /opt/watermeter-image-processor

EXPOSE 8080/tcp

ENTRYPOINT ["/opt/watermeter-image-processor"]
