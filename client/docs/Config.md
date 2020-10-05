# Config

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**StartupDelay** | **float64** | How many seconds to delay incoming connections to a webspace while starting the container  | [optional] [default to 3.0]
**HttpPort** | **int32** | Incoming SSL-terminated HTTP requests (and SNI passthrough HTTPS connections) will be forwarded to this port  | [optional] [default to 80]
**SniPassthrough** | **bool** | If true, SSL termination will be disabled and HTTPS connections will forwarded directly  | [optional] [default to false]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


