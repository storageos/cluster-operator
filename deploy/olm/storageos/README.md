# community operator artifacts

This directory contains the Cluster Service Version (CSV), Custom Resource
Definition (CRD) and package files for
[community operator](https://operatorhub.io/) releases.
`storageos.clusterserviceversion.yaml` contains the latest CSV. CSVs for
releases have the release version in the file name
`storageos.<version>.clusterserviceversion.yaml`. A released CSV file must not
be modified because releases are versioned and can't be rereleased. For any
change, a new CSV with incremented version must be created.

## Testing

Run `make olm-lint` from the project root to lint all the file in this
directory. A lint must be performed before submitting a new release to the
operatorhub.

For e2e tests, ensure that any change in `storageos.clusterserviceversion.yaml`,
`storageos.package.yaml` and all the crd.yaml files are copied to
[`/deploy/storageos-operators.configmap`](/deploy/storageos-operators.configmap).
The configmap is used as a catalog source in the OLM e2e tests.

__NOTE__: With more tooling, we should be able to generate a gRPC catalog source
using the files in this directory directly, without maintaining a separate
configmap.
