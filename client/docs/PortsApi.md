# \PortsApi

All URIs are relative to *https://webspaced.netsoc.ie/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddPort**](PortsApi.md#AddPort) | **Post** /webspace/{username}/ports/{ePort}/{iPort} | Add port forward
[**AddRandomPort**](PortsApi.md#AddRandomPort) | **Post** /webspace/{username}/ports/{iPort} | Add random port forward
[**GetPorts**](PortsApi.md#GetPorts) | **Get** /webspace/{username}/ports | Retrieve webspace port forwards
[**RemovePort**](PortsApi.md#RemovePort) | **Delete** /webspace/{username}/ports/{ePort} | Delete port forward



## AddPort

> AddPort(ctx, username, ePort, iPort)

Add port forward

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**ePort** | **int32**|  | 
**iPort** | **int32**|  | 

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


## AddRandomPort

> AddRandomPortResponse AddRandomPort(ctx, username, iPort)

Add random port forward

Add port forward from random free port to internal port

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**iPort** | **int32**|  | 

### Return type

[**AddRandomPortResponse**](AddRandomPortResponse.md)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPorts

> map[string]int32 GetPorts(ctx, username)

Retrieve webspace port forwards

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

**map[string]int32**

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RemovePort

> RemovePort(ctx, username, ePort)

Delete port forward

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**ePort** | **int32**|  | 

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

