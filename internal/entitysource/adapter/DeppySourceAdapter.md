## DeppySourceAdapter design choices

The `DeppySourceAdapter` is meant to be a translation layer between an external source and deppy. It converts the bundle information from the external format into the `Entity` format identified by deppy.

An `Entity` consists of a unique `Id` string and a set of properties.

```go
type Entity struct {
	Id         EntityID          `json:"id,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}
```

The adapter has a single grpc endpoint: `ListEntities`, which returns a list of entities for the source the adapter is connected to. Implementing a grpc endpoint does make testing against the adapter more complex, but with a requirement for a grpc catalogsource querier, the overhead of maintaining another library for the adapter API seemed like it would add to the maintenance burden.

### CatalogSourceAdapter

The `CatalogSourceAdapter` is designed as a deployment that handles exactly one specified `CatalogSource`. This simplifies the adapter itself, but requires a separate controller to create the deployments as needed.

The `CatalogSourceAdapter` does not have any caching implemented and requires a `ListBundles` grpc call to the backing CatalogSource for every `ListEntities` call. The `Entity` list is therefore up-to-date if the backing `CatalogSource` does not have any caching issues.

Since the `EntityID` is meant to have the most important identifying information for a bundle, the `CatalogSourceAdapter` uses a marshalled JSON object as the ID.

```go
type BundleEntityID struct {
	CSVName string `json:"name,omitempty"`
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source,omitempty"`
	Path    string `json:"path,omitempty"`
}
```

The ID should contain everything needed to identify and install a bundle. An example entity ID might look like:
```json
{"id": "{'name':'myoperator-v0.1.0-beta1','package':'myoperator','version':'0.1.0-beta1','source':'mycatalog','path':'path/to/myoperator@sha256:01eebf7561d709082074d6131cac018f49cacfea55ca78290ba83ec77d578442'}"}
```

While this simplifies the resolution output a bundle needs to act on, the ID can become unnecessarily long. It can also cause duplicate bundles for a `\(Package,Version\)` tuple where the Path may be different for two otherwise identical bundles.

The properties themselves are marshalled lists of possible values for their identifying key. For instance, the `gvk` property might look like:

```json
{"olm.gvk": "[{'group':'mydomain.com','kind':'MyCRD','version':'v1'},{'group':'mydomain.com/v1','kind':'MyCRD','version':'v1'}]"}
```

This allows multiple values for a single property type, which is useful for properties like `olm.gvk.required`, which may have multiple entries.

The `olm.channel` property is also similarly aggregated, unlike the pre-existing `ListBundles` call. It lists all channels a bundle belongs to, along with any possible upgrade edges that bundle may have on that channel. An example of an `olm.channel` property is shown below:
``` json
{"olm.channel": "[{'channelName':'preview','priority':0,'skipRange':'>=4.0.0 <5.0.0'},{'channelName':'stable','priority':0,'skipRange':'>=4.0.0 <5.0.0'}]"}
```


With the current prototype of the catalogsource adapter, properties present only on the CSV like `minKubeVersion` and some information like the `DefaultChannel` is not provided by the ListBundles call. The adapter may be extended to support such omitted information as additional properties in the future.

The entity format itself is still not stable, so may be subject to future modifications. [One entity format proposed looks like](https://github.com/joelanford/deppy-client-go/blob/main/api/entity.go):

```go
type Entity struct {
	ID          string          `json:"id"`
	Data        json.RawMessage `json:"data,omitempty"`
	Properties  []TypeValue     `json:"properties,omitempty"`
	Constraints []TypeValue     `json:"constraints,omitempty"`
}

type TypeValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}
```
The addition of the `Data` field allows for providing useful metadata to the user. For instance, providing a `BundleImage` in `Data` separates it from the bundle identity and from the properties list needed for resolution. 

The separation of Properties and Constraints moves the onus of constraint creation to some exent to the adapter. This may make constraint generation easier for deppy.
