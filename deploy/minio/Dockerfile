# Local-development MinIO (S3-compatible object storage), built from the upstream source release because
# MinIO no longer publishes container images. AGPL-3.0; used only in the local/CI stack, never in production.
FROM golang:1.25-alpine AS build
ARG MINIO_RELEASE=RELEASE.2025-10-15T17-29-55Z
RUN apk add --no-cache git
RUN git clone --depth 1 --branch "${MINIO_RELEASE}" https://github.com/minio/minio /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/minio .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates curl && adduser -D -H -u 10002 minio && mkdir /data && chown minio /data
COPY --from=build /out/minio /usr/local/bin/minio
USER minio
EXPOSE 9000 9001
ENTRYPOINT ["minio"]
