# The pinned digest must carry a Go toolchain >= the `go` directive in go.mod.
# These images set GOTOOLCHAIN=local, so the build cannot fetch a newer toolchain
# and a lagging digest fails with "go.mod requires go >= X (running go Y)".
# This digest is golang:1.27-alpine with GOLANG_VERSION=1.27.3.
FROM golang:1.27-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /screws-box ./cmd/screwsbox

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /screws-box /screws-box
EXPOSE 8080
ENTRYPOINT ["/screws-box"]
