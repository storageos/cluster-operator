# Licence

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ClusterID** | **string** | A unique identifier for a cluster. The format of this type is undefined and may change but the defined properties will not change.  | [optional] [readonly] 
**ExpiresAt** | [**time.Time**](time.Time.md) | The time after which a licence will no longer be valid This timestamp is set when the licence is created. String format is RFC3339.  | [optional] [readonly] 
**ClusterCapacityBytes** | **uint64** | The allowed provisioning capacity in bytes This value if for the cluster, if provisioning a volume brings the cluster&#39;s total provisioned capacity above it the request will fail  | [optional] 
**Kind** | **string** | Denotes which category the licence belongs to  | [optional] 
**CustomerName** | **string** | A user friendly reference to the customer  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


