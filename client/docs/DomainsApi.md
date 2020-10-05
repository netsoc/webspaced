# \DomainsApi

All URIs are relative to *https://webspaced.netsoc.ie/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddDomain**](DomainsApi.md#AddDomain) | **Post** /webspace/{username}/domains/{domain} | Add custom domain
[**GetDomains**](DomainsApi.md#GetDomains) | **Get** /webspace/{username}/domains | Retrieve webspace domains
[**RemoveDomain**](DomainsApi.md#RemoveDomain) | **Delete** /webspace/{username}/domains/{domain} | Delete custom domain



## AddDomain

> AddDomain(ctx, username, domain)

Add custom domain

Domain will be verified by looking for a `TXT` record of the format `webspace:id:<user id>` 

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**domain** | **string**|  | 

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


## GetDomains

> []string GetDomains(ctx, username)

Retrieve webspace domains

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 

### Return type

**[]string**

### Authorization

[jwt](../README.md#jwt), [jwt_admin](../README.md#jwt_admin)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RemoveDomain

> RemoveDomain(ctx, username, domain)

Delete custom domain

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**username** | **string**| User&#39;s username. Can be &#x60;self&#x60; to indicate the currently authenticated user.  | 
**domain** | **string**|  | 

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

