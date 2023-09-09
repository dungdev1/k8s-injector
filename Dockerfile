FROM golang:1.16 AS build
ENV PROJECT k8s-injector
WORKDIR /src/$PROJECT
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /src/$PROJECT/cmd
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /go/bin/k8s-injector
RUN ls /go/bin

FROM alpine:3.8
COPY --from=build /go/bin/k8s-injector /usr/bin/k8s-injector
ENV TLS_PORT 8443
ENV TLS_CERTIFICATE_FILE=/var/lib/secrets/tls.crt
ENV TLS_KEY_FILE=/var/lib/secrets/tls.key
ENV CONFIGMAP_NAME=k8s-injector
CMD [ "/usr/bin/k8s-injector" ]