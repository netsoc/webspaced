# \StateApi

All URIs are relative to *https://webspaced.netsoc.ie/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetState**](StateApi.md#GetState) | **Get** /webspace/{username}/state | Retrieve webspace state
[**Reboot**](StateApi.md#Reboot) | **Put** /webspace/{username}/state | Reboot webspace container
[**Shutdown**](StateApi.md#Shutdown) | **Delete** /webspace/{username}/state | Shut down webspace container
[**Start**](StateApi.md#Start) | **Post** /webspace/{username}/state | Start webspace container
[**Sync**](StateApi.md#Sync) | **Patch** /webspace/{username}/state | Re-generate webspace backend config



## GetState

> State GetState(ctx, username)

Retrieve webspace state

Retrieve webspace state 

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

[**State**](State.md)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Reboot

> Reboot(ctx, username)

Reboot webspace container

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

 (empty response body)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Shutdown

> Shutdown(ctx, username)

Shut down webspace container

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

 (empty response body)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Start

> Start(ctx, username)

Start webspace container

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

 (empty response body)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Sync

> Sync(ctx, username)

Re-generate webspace backend config

Forces a reload of reverse proxy and port forwarding configuration

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

 (empty response body)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

