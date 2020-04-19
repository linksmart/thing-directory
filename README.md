# Web of Things (WoT) Thing Directory
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/td.svg)](https://hub.docker.com/r/linksmart/td/tags)
[![GitHub tag (latest pre-release)](https://img.shields.io/github/tag-pre/linksmart/thing-directory.svg?label=pre-release)](https://github.com/linksmart/thing-directory/tags)
[![Build Status](https://travis-ci.com/linksmart/thing-directory.svg?branch=master)](https://travis-ci.com/linksmart/thing-directory)
  
This is an implementation of the W3C [Web of Things (WoT) Thing Directory](https://www.w3.org/TR/wot-architecture/#dfn-thing-directory), a catalog of [Thing Descriptions](https://www.w3.org/TR/wot-thing-description/).

This is currently under development.

## Getting Started
API Documentation: [OpenAPI Specification](https://linksmart.eu/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml)

## Installation
### Binary Distribution
1. Download the binary distribution and configuration file from [releases](https://github.com/linksmart/thing-directory/releases)
2. Download the WoT Thing Description JSON Schema document. E.g. [wot_td_schema.json](https://raw.githubusercontent.com/linksmart/thing-directory/master/wot/wot_td_schema.json)
3. Run, e.g. in Linux/AMD64:
```
./thing-directory-linux-amd64 --conf ./thing-directory.json --schema ./wot_td_schema.json
```
For more information about the CLI arguments, set `--help` flag.

### Docker
Run the latest build of Thing Directory with the default configuration file ([/conf/thing-directory.json](https://github.com/linksmart/thing-directory/blob/master/sample_conf/thing-directory.json)):
```
docker run -p 8081:8081 linksmart/td
```
The index of the RESTful API should now be accessible at: http://localhost:8081

The configurations can be changes by mounting a directory and providing the paths in CLI arguments. For more information about the CLI arguments, set `--help` flag.

Please refer to the API Documentation to learn about the different endpoints.

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
