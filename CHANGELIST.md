# CHANGELIST
* Migration from Godepts to GB vendor
* Updated github.com/oleksandr/bonjour package and its usage:
  - New method for shutdown
* Updated github.com/gorilla/mux package and it usage:
  -  New RegEx format for variable path depths
* Changed code.google.com/p/go-uuid/uuid to github.com/pborman/uuid 
  - Google Code no longer go gettable
* Added auth support
  - Custom HTTP client with optional AUTH support for service/resource catalog APIs
  - service/resource registration with optional AUTH
  - AUTH struct in config files for server configuration
  - Optional AUTH struct in config files for client authentication (service/resource registration)