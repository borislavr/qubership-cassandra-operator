FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine AS builder

ENV GOSUMDB=off GOPRIVATE=github.com/Netcracker

RUN apk add --no-cache git
RUN --mount=type=secret,id=GH_ACCESS_TOKEN \
    git config --global url."https://$(cat /run/secrets/GH_ACCESS_TOKEN)@github.com/".insteadOf "https://github.com/"

COPY . /workspace

WORKDIR /workspace
RUN go mod tidy

# Build
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ./build/_output/bin/dbaas-cassandra-adapter \
    -gcflags all=-trimpath=${GOPATH} -asmflags all=-trimpath=${GOPATH} ./main.go

FROM alpine:3.23

#ENV WORKDIR=/opt/operator/

ENV OPERATOR=/usr/local/bin/dbaas-cassandra-adapter \
    USER_UID=1001 \
    USER_NAME=dbaas-cassandra-adapter

RUN echo 'https://dl-cdn.alpinelinux.org/alpine/edge/main/' > /etc/apk/repositories && \
    echo 'https://dl-cdn.alpinelinux.org/alpine/edge/community/' >> /etc/apk/repositories \
    && apk update \
    && apk add curl


COPY --from=builder /workspace/build/_output/bin/dbaas-cassandra-adapter ${OPERATOR}
COPY build/bin /usr/local/bin

RUN chmod +x /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/user_setup && /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
