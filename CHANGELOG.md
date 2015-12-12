# CHANGELOG

* 0.2.0
    - Migration from Godepts to GB vendor
    - Updated github.com/oleksandr/bonjour package and its usage:
      + New method for shutdown
    - Updated github.com/gorilla/mux package and it usage:
      + New RegEx format for variable path depths
    - Changed code.google.com/p/go-uuid/uuid to github.com/pborman/uuid 
      + Google Code is no longer go gettable
    - Replaced PublicAddr with PublichEndpoint:
      + Allows to use custom <protocol>://<addr>:<port> for local endpoints when publishing to catalogs etc. E.g., can be used together with reverse proxy.
    - Added (optional) authentication and authorization support for HTTP APIs
      + optional AUTH struct in config files of services and clients
      + Server side (service/resource catalogs, device-gateway)
          * (optional) AUTH struct in config files for server configuration
      + Client side (service/resource catalogs clients, device-gateway)
          * remote catalogs clients: all HTTP requests through a custom HTTP client with AUTH support
      + Support for multiple AUTH providers (drivers)
    - Added LevelDB storage backend for service/resource catalogs
      + Configurable backend for service/resource catalogs clients, device-gateway
    - Added Proxy-based storage implementation to resource catalog
      + Implementation in resource/proxystorage.go
      + Added/Modified methods in resource/remoteclient.go
          * Added GetResource
          * GetDevices->GetMany, fixed a bug in devicesFromResponse func
          * Updated corresponding interfaces in resource/client.go
      + Added rc-proxy (resource-catalog proxy), a client for Proxy-based resource catalog
    - Minor API modifications
      + Error handling
          * All storage interfaces return errors (service/resource catalogs)
          * Handling errors in catalogapi.go (service/resource catalogs)
      + Optimized 'resources' filtering
          * CatalogStorage.pathFilterResources returns Devices instead
          * CatalogClient.FindResources returns Devices instead
      + Removed devicesFromResources() from storage interface methods (resource catalog)
      + Added Close() to storage interface (service/resource catalogs)
      + Changed Device.Expires/Service.Expires to pointer, set to null if TTL < 0 (resource/service catalog)
      + Path Filtering supports data types other than string (e.g. int, float, bool)