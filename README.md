# Web of Things (WoT) Thing Directory
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/td.svg)](https://hub.docker.com/r/linksmart/td/tags)
[![GitHub tag (latest pre-release)](https://img.shields.io/github/tag-pre/linksmart/thing-directory.svg?label=pre-release)](https://github.com/linksmart/thing-directory/tags)
[![Build Status](https://travis-ci.com/linksmart/thing-directory.svg?branch=master)](https://travis-ci.com/linksmart/thing-directory)
  
This is an implementation of the [Web of Things (WoT) Thing Directory](https://www.w3.org/TR/wot-architecture/#dfn-thing-directory) which provides a RESTful API to maintain [Thing Descriptions](https://www.w3.org/TR/wot-thing-description/).

This is currently under development.

## Getting Started
API Documentation: [OpenAPI Specification](https://linksmart.eu/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml)

## Deployment
### Docker
The following command runs the latest build of Thing Directory with the default configuration file ([/conf/thing-directory.json](https://github.com/linksmart/thing-directory/blob/master/sample_conf/thing-directory.json)):
```
docker run -p 8081:8081 linksmart/td
```
The index of the RESTful API should now be accessible at: http://localhost:8081

Please refer to the API Documentation to learn about the different endpoints.

## Development
The dependencies of this package are managed by [Go Modules](https://github.com/golang/go/wiki/Modules).

To Compile from source:
```
git clone https://github.com/linksmart/thing-directory.git
cd thing-directory
go build
```

