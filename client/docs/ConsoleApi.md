# \ConsoleApi

All URIs are relative to *https://webspaced.netsoc.ie/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetLog**](ConsoleApi.md#GetLog) | **Get** /webspace/{username}/console | Retrieve webspace console log



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

