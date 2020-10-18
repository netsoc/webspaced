# \ConsoleApi

All URIs are relative to *https://webspaced.netsoc.ie/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ClearLog**](ConsoleApi.md#ClearLog) | **Delete** /webspace/{username}/log | Clear webspace console log
[**Console**](ConsoleApi.md#Console) | **Get** /webspace/{username}/console | Attach to webspace console
[**Exec**](ConsoleApi.md#Exec) | **Post** /webspace/{username}/exec | Execute command non-interactively
[**ExecInteractive**](ConsoleApi.md#ExecInteractive) | **Get** /webspace/{username}/exec | Execute a command interactively
[**GetLog**](ConsoleApi.md#GetLog) | **Get** /webspace/{username}/log | Retrieve webspace console log



## ClearLog

> ClearLog(ctx, username)

Clear webspace console log

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


## Console

> Console(ctx, username)

Attach to webspace console

_IMPORTANT_: This endpoint uses a websocket. On connection, a single text message should be sent with integers for terminal `width` and `height` (as JSON, see `ResizeRequest` e.g. `{\"width\": 80, \"height\": 24}`). Following this, binary messages to and from the socket will be routed to the console TTY.  Any other text messages will also be treated as resize events (same format). 

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


## Exec

> ExecResponse Exec(ctx, username, execRequest)

Execute command non-interactively

Runs a command non-interactively (no TTY, waits for completion and returns complete stdout and stderr). 

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**execRequest** | [**ExecRequest**](ExecRequest.md)|  | 

### Return type

[**ExecResponse**](ExecResponse.md)

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ExecInteractive

> ExecInteractive(ctx, username)

Execute a command interactively

_IMPORTANT_: This endpoint uses a websocket. On connection, a single text message should be sent (as JSON), this message is of the form `ExecInteractiveRequest`. Following this, binary messages to and from the socket will be routed to the process PTY.  Any other text messages will be treated as `ExecInteractiveControl` messages. Pass a signal number to send a signal to the process, and non-zero values for `width` and `height` to resize.  Upon command completion, the close message will be the exit code of the process. 

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


## GetLog

> string GetLog(ctx, username)

Retrieve webspace console log

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

**string**

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: text/plain, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

