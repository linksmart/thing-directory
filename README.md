# Web of Things (WoT) Thing Directory
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/td.svg)](https://hub.docker.com/r/linksmart/td/tags)
[![GitHub tag (latest pre-release)](https://img.shields.io/github/tag-pre/linksmart/thing-directory.svg?label=pre-release)](https://github.com/linksmart/thing-directory/tags)
[![Build Status](https://travis-ci.com/linksmart/thing-directory.svg?branch=master)](https://travis-ci.com/linksmart/thing-directory)
  
This is a candidate implementation for the W3C [Web of Things (WoT) Thing Directory](https://www.w3.org/TR/wot-architecture/#dfn-thing-directory) service, a catalog of [Thing Descriptions](https://www.w3.org/TR/wot-thing-description/).

The catalog currently supports XPath 3.0 and JSONPath as query languages.

This is currently under development.

## Getting Started
* [Deployment](https://github.com/linksmart/thing-directory/wiki/Deployment)
* [Configuration](https://github.com/linksmart/thing-directory/wiki/Configuration)
* [OpenAPI Specification](https://linksmart.github.io/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml)

Further documentation are available in the **[wiki](https://github.com/linksmart/thing-directory/wiki)**.

## Development
The dependencies of this package are managed by [Go Modules](https://github.com/golang/go/wiki/Modules).

To Compile from source:
```
git clone https://github.com/linksmart/thing-directory.git
cd thing-directory
go build
```

## Contributing
Contributions are welcome. 

Please fork, make your changes, and submit a pull request. For major changes, please open an issue first and discuss it with the other authors.
