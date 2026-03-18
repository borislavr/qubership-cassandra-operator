FROM --platform=$BUILDPLATFORM golang:1.25.8-alpine3.23 AS builder

ENV GOSUMDB=off GOPRIVATE=github.com/Netcracker

RUN apk add --no-cache git
RUN --mount=type=secret,id=GH_ACCESS_TOKEN git config --global url."https://$(cat /run/secrets/GH_ACCESS_TOKEN)@github.com/".insteadOf "https://github.com/"

COPY . /opt/operator

WORKDIR /opt/operator

RUN go mod tidy

# Build
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ./bin/cassandra-operator \
    -gcflags all=-trimpath=${GOPATH} -asmflags all=-trimpath=${GOPATH} ./main.go

FROM alpine:3.23

ENV OPERATOR=/usr/local/bin/cassandra-operator \
    USER_UID=1001 \
    USER_NAME=cassandra-operator

RUN echo 'https://dl-cdn.alpinelinux.org/alpine/edge/main/' > /etc/apk/repositories \
    && apk add --no-cache openssl curl \
    && apk update \
    && apk upgrade

COPY --from=builder /opt/operator/bin/cassandra-operator ${OPERATOR}
COPY build/bin /usr/local/bin
#COPY bin/gocql /usr/local/bin/

RUN chmod +x /usr/local/bin/entrypoint
RUN  chmod +x /usr/local/bin/user_setup && /usr/local/bin/user_setup

RUN mkdir -p deployments/charts/cassandra-operator

COPY /charts/helm/cassandra-operator/ deployments/charts/cassandra-operator/
COPY /charts/helm/cassandra-operator/deployment-configuration.json deployments/deployment-configuration.json

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
