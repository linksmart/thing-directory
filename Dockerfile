FROM golang:1.14-alpine as builder

COPY . /home

WORKDIR /home
ENV CGO_ENABLED=0

ARG version
ARG buildnum
RUN go build -v -ldflags "-X main.Version=$version -X main.BuildNumber=$buildnum"

###########
FROM alpine

RUN apk --no-cache add ca-certificates

ARG version
ARG buildnum
LABEL NAME="LinkSmart Thing Directory"
LABEL VERSION=${version}
LABEL BUILD=${buildnum}

WORKDIR /home
COPY --from=builder /home/thing-directory .
COPY sample_conf/thing-directory.json /home/conf/
COPY wot/wot_td_schema.json /home/conf/

ENV TD_STORAGE_DSN=/data

VOLUME /data
EXPOSE 8081

ENTRYPOINT ["./thing-directory"]
# Note: this loads the default config files from /home/conf/. Use --help to to learn about CLI arguments.