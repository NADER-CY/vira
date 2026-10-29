# Builds every VIRA backend runtime into one small image; the compose service picks the command.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN mkdir /out && for c in api admin-api worker realtime media migrate devsim; do \
      CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/NADER-CY/vira/backend/internal/app/runtime.Version=${VERSION}" -o /out/$c ./cmd/$c; \
    done

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget && adduser -D -H -u 10001 vira
COPY --from=build /out/ /usr/local/bin/
USER vira
EXPOSE 8080
CMD ["api"]
