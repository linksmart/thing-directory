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
COPY sample_conf/* /conf/

ENV SC_DNSSDENABLED=false
ENV SC_STORAGE_TYPE=leveldb
ENV SC_STORAGE_DSN=/data

VOLUME /conf /data
EXPOSE 8081

ENTRYPOINT ["./thing-directory"]
CMD ["-conf", "/conf/thing-directory.json"]