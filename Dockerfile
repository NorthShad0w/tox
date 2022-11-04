FROM golang:1.17.3-alpine as builder
WORKDIR /app

ARG APP_NAME
ENV APP_NAME ${APP_NAME}
ARG APP_VERSION
ENV APP_VERSION ${APP_VERSION}

COPY . .
RUN mkdir -p ./dist && GO111MODULE=on go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/isayme/tox/util.Name=${APP_NAME} \
  -X github.com/isayme/tox/util.Version=${APP_VERSION}" \
  -o ./dist/tox main.go

RUN GOOS="windows" GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/isayme/tox/util.Name=${APP_NAME} \
  -X github.com/isayme/tox/util.Version=${APP_VERSION}" \
  -o ./dist/tox.exe main.go

FROM alpine
WORKDIR /app

ARG APP_NAME
ENV APP_NAME ${APP_NAME}
ARG APP_VERSION
ENV APP_VERSION ${APP_VERSION}

# default config file
ENV CONF_FILE_PATH=/etc/tox.yaml

COPY --from=builder /app/dist/tox /app/tox
COPY --from=builder /app/dist/tox.exe /app/tox

CMD ["/app/tox", "-h"]
