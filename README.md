# Web of Things (WoT) Thing Directory
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/td.svg)](https://hub.docker.com/r/linksmart/td/tags)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/linksmart/thing-directory)](https://github.com/linksmart/thing-directory/releases)
[![Build Status](https://travis-ci.com/linksmart/thing-directory.svg?branch=master)](https://travis-ci.com/linksmart/thing-directory)
  
This is an implementation of the [Web of Things (WoT) Thing Directory](https://www.w3.org/TR/wot-architecture/#dfn-thing-directory) which provides a RESTful API to maintain [Thing Descriptions](https://www.w3.org/TR/wot-thing-description/).

This is currently under development.

## Getting Started
API Documentation: [OpenAPI Specification](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml)

## Development
The dependencies of this package are managed by [Go Modules](https://github.com/golang/go/wiki/Modules).

To Compile from source:
```
git clone https://github.com/linksmart/thing-directory.git
cd thing-directory
go build
```

