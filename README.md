# LinkSmart Thing Directory
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/td.svg)](https://hub.docker.com/r/linksmart/td/tags)
[![GitHub tag (latest pre-release)](https://img.shields.io/github/tag-pre/linksmart/thing-directory.svg?label=pre-release)](https://github.com/linksmart/thing-directory/tags)
[![CICD](https://github.com/linksmart/thing-directory/workflows/CICD/badge.svg)](https://github.com/linksmart/thing-directory/actions?query=workflow:CICD)
[![DOI](https://joss.theoj.org/papers/10.21105/joss.03075/status.svg)](https://doi.org/10.21105/joss.03075)
  
This is an implementation of the [W3C WoT Thing Description Directory (TDD)](https://w3c.github.io/wot-discovery/), a registry of [Thing Descriptions](https://www.w3.org/TR/wot-thing-description/).

## Getting Started
Visit the following pages to get started:
* [Deployment](https://github.com/linksmart/thing-directory/wiki/Deployment): How to deploy the software, as Docker container, Debian package, or platform-specific binary distributions
* [Configuration](https://github.com/linksmart/thing-directory/wiki/Configuration): How to configure the server software with JSON files and environment variables
* [API Documentation](https://linksmart.github.io/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml): How to interact with the networking APIs

**Further documentation are available in the [wiki](https://github.com/linksmart/thing-directory/wiki)**.

## Features
* Service Discovery
  * [DNS-SD registration](https://github.com/linksmart/thing-directory/wiki/Discovery-with-DNS-SD)
  * [LinkSmart Service Catalog](https://github.com/linksmart/service-catalog) registration
* RESTful API
  * [HTTP API](https://linksmart.github.io/swagger-ui/dist/?url=https://raw.githubusercontent.com/linksmart/thing-directory/master/apidoc/openapi-spec.yml)
    * Thing Description (TD) CRUD, catalog, and validation
    * XPath 3.0 and JSONPath [query languages](https://github.com/linksmart/thing-directory/wiki/Query-Language)
    * TD validation with JSON Schema ([default](https://github.com/linksmart/thing-directory/blob/master/wot/wot_td_schema.json))
    * Request [authentication](https://github.com/linksmart/go-sec/wiki/Authentication) and [authorization](https://github.com/linksmart/go-sec/wiki/Authorization)
    * JSON-LD response format
* Persistent Storage
  * LevelDB
* CI/CD ([Github Actions](https://github.com/linksmart/thing-directory/actions?query=workflow:CICD))
  * Automated testing
  * Automated builds and releases ([Docker images](https://hub.docker.com/r/linksmart/td/tags?page=1&ordering=last_updated), [binaries](https://github.com/linksmart/thing-directory/releases))

## Development
The dependencies of this package are managed by [Go Modules](https://github.com/golang/go/wiki/Modules).

Clone this repo:
```bash
git clone https://github.com/linksmart/thing-directory.git
cd thing-directory
```

Compile from source:
```bash
go build
```
This will result in an executable named `thing-directory` (linux/macOS) or `thing-directory.exe` (windows).

Get the CLI arguments help (linux/macOS):
```bash
$ ./thing-directory -help
Usage of ./thing-directory:
  -conf string
        Configuration file path (default "conf/thing-directory.json")
  -version
        Print the API version
```

Run (linux/macOS):
```bash
$ ./thing-directory --conf=sample_conf/thing-directory.json
```

To build and run together:
```bash
go run . --conf=sample_conf/thing-directory.json
```

Test all packages (add `-v` flag for verbose results):
```bash
go test ./...
```


## Contributing
Contributions are welcome. 

Please fork, make your changes, and submit a pull request. For major changes, please open an issue first and discuss it with the other authors.
