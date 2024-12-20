FROM golang:1.23

ARG SERVICE

ENV SERVICE=${SERVICE}

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY internal/ping ./internal/ping/
COPY internal/shared ./internal/shared/

COPY cmd/${SERVICE}/ ./cmd/${SERVICE}/

WORKDIR /usr/src/app/cmd/${SERVICE}

RUN go mod download

COPY cmd/${SERVICE}/. .

RUN go build -o ${SERVICE} .

RUN chmod +x ./${SERVICE}

COPY docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
