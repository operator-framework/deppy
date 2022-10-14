# Active Resolution

The ActiveResolution API and its controller will manage the state of ActiveResolution resources. It executes the main algorithm of producing solver constraints from constraint definitions and entities, triggers resolution, converts results to something consumable by users and writes results out to the particular ActiveResolution resource.

## API definition

Though not the final definition for this resource, the following provides an example of what it might look like:

```yaml
apiVersion: deppy.io/v1alpha1
kind: ActiveResolution
metadata:
  name: example-resolution
spec:
  sources: # optional definition of specific sources
  - "example-operators"
  constraints: # define constraints necassary for selections
  - required("first-example-operator", ">= 1.0.0", "stable")
  # or
  - required:
      name: "second-example-operator"
      versionRange: ">= 1.0.0 < 1.2.0"
      # optionally
      channel: "stable"
status:
  conditions: # status conditions
  ...
  selections: # items selected in the form of source:name:version:channel
  - example-operators:first-example-operator:1.2.0:stable
  - example-operators:second-example-operator:1.1.0:stable
  - ...
```

## Functionality

For consumers of `Deppy`, they will be expected to provide the `sources` to reach out to as well as the `constraints` necessary for the desired resolution. Using this information, `Deppy` will then reach out to the appropriate cluster resources to generate a resolution. **`Deppy` will not be responsible for installing this content onto the cluster**, instead it is up to the consumer to determine how to act on a `ActiveResolution`'s status.

### With the Operator API

Its important to illustrate how the `Operator API` use case can utilize `Deppy` to understand the ultimate contract between the services for Phase 0 of `Deppy`. 

> **Note**: Keep in mind that the `Operator API` has not been completely designed. For the sake of discussing how the `ActiveResolution` component works within the larger OLM V1 system, lets assume that this `Operator API` has been designed. It takes has a single crd called `Operator` which can take in a package to install.

Say that we have defined an example `Operator` like so:

```yaml
apiVersion: platform.openshift.io/v1alpha1
kind: Operator
metadata:
  name: foo-operator
spec:
  package:
    name: foo-package
```

Upon applying this resource, there are two API's that will now be at play - Deppy and RukPak. To start the process, the `Operator` controller will create an `ActiveResolution` for `Deppy` to reconcile. 

```yaml
apiVersion: deppy.io/v1alpha1
kind: ActiveResolution
metadata:
  name: foo-resolution
spec:
  constraints:
  - required:
      name: "foo-package"
      channel: "platform"
...
```

For the sake of example, lets say that `foo-package` is reliant on another package called `bar-package`. In this case, `Deppy` will understand and convey this dependency to the status.

```yaml
...
status:
  conditions: # status conditions
  ...
  selections: # items selected in the form of source:name:version:channel
  - redhat-operators:foo-package:1.2.0:platform
  - redhat-operators:bar-package:1.1.0:platform
```

Looking at the status, we can see that there is a constructed query that tells us where our resolved entitiy is located at. The `Operator` controller could read this and perform another query to get the images; However, it may be pertitnent for a new component to be introduced into Deppy - the `ResolutionActivator` (name TBD). 

### Resolution Activator

The responsibility for the `ResolutionActivator` is to be able to read `ActiveResolution`s and parse out the installable units of work. This could follow an adapter pattern, like Deppy itself does,  where the adapters are able to retrieve installable content. In the use case of `Operator`s this would result in a collections of images being made available. To mark a `ActiveResolution` with the appropriate adapter as well as if it should be activated, we can use annotations.

```yaml
apiVersion: deppy.io/v1alpha1
kind: ActiveResolution
metadata:
  annotations:
    deppy.io/v1alpha1/activator-class: "Operator"
  name: foo-resolution
...
```

This would result in the status being updated to include a new section - entities.

```yaml
...
status:
  conditions: # status conditions
  ...
  selections: # items selected in the form of source:name:version:channel
  - redhat-operators:foo-package:1.2.0:platform
  - redhat-operators:bar-package:1.1.0:platform
  entities: # entities that were made installable via the activator-class annotation
  - package: foo-package
    image: quay.io/redhat-operators/foo-package:v1.2.0
  - package: bar-package
    image: quay.io/redhat-operators/bar-package:v1.1.0
```

From here, the `Operator` controller can read the resolved `entities` and install them accordingly onto the cluster via `RukPak`. However, it is pertinent to allow `Rukpak` to manage these installables as a group of installed content instead of individuals. This is where the `ResolveSet` bundle format comes into play. 

A `ResolveSet` bundle is, at its core, just a `plain+v0` bundle of `BundleDeployments`. What this means is that when the `ResolveSet` bundle gets installed the `plain` provisioner will work to reconcile that content as though it were arbitrary manifests, essentially having a wrapper `BundleDeployment` for `BundleDeployments`. Generation of the `ResolveSet` bundle is trivial and can be done on the fly by the `Operator` controller. If we were to look at a `ResolveSet` bundle's manifests for the above example, it would look something like this:

```console
manifests/
|- foo-package-bd.yaml
|- bar-package-bd.yaml
```

From this, the `Operator` controller is able to maintain a single `BundleDeployment` for all of the resolved content.
