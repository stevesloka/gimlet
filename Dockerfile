FROM golang:1.14.1 AS build
WORKDIR /gimlet

ENV GOPROXY=https://proxy.golang.org
COPY go.mod go.sum /gimlet/
RUN go mod download

COPY client client
COPY server server
COPY Makefile Makefile

RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-ldflags=-w go build -o /go/bin/gimlet-client -ldflags=-s -v github.com/stevesloka/gimlet/client
RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-ldflags=-w go build -o /go/bin/gimlet-server -ldflags=-s -v github.com/stevesloka/gimlet/server

FROM scratch AS final
COPY --from=build /go/bin/gimlet-client /bin/gimlet-client
COPY --from=build /go/bin/gimlet-server /bin/gimlet-server
