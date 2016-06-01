FROM golang
MAINTAINER Alexandr Krylovskiy "alexandr.krylovskiy@fit.fraunhofer.de"
ENV REFRESHED_AT 2016-01-06

# update system
RUN apt-get update
RUN apt-get install -y wget git

# install the fraunhofer certificate
RUN wget http://cdp1.pca.dfn.de/fraunhofer-ca/pub/cacert/cacert.pem -O /usr/local/share/ca-certificates/fhg.crt
RUN update-ca-certificates

# install go tools
RUN go get github.com/constabulary/gb/...

# setup local connect home
RUN mkdir /opt/lslc
ENV LSLC_HOME /opt/lslc
WORKDIR ${LSLC_HOME}

# copy code & build
COPY . ${LSLC_HOME}
RUN gb build all

VOLUME conf
VOLUME data

EXPOSE 8081
EXPOSE 8082
