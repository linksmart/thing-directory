FROM golang:1.8-alpine

# copy default config file and code
COPY sample_conf/* /conf/
COPY . /home/src/code.linksmart.eu/sc/service-catalog

# build the code
ENV GOPATH /home
RUN go install code.linksmart.eu/sc/service-catalog

WORKDIR /home

VOLUME /conf /data
EXPOSE 8082

ENTRYPOINT ["./bin/service-catalog"]
CMD ["-conf", "/conf/docker.json"]
