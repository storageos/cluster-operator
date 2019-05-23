# community operator artifacts

This directory contains the Cluster Service Version (CSV), Custom Resource
Definition (CRD) and package files for
[community operator](https://operatorhub.io/) releases.
`storageos.clusterserviceversion.yaml` contains the latest CSV. CSVs for
releases have the release version in the file name
`storageos.<version>.clusterserviceversion.yaml`. A released CSV file must not
be modified because releases are versioned and can't be rereleased. For any
change, a new CSV with incremented version must be created.

## Release Instructions

Update `metadata.name` with new version number,
`metadata.annotations.containerImage` and
`spec.install.spec.deployments[0].spec.template.spec.containers[0].image` with
the new operator container image, and `spec.version` to the new release version
number in `deploy/olm/community-changes.yaml`. Run `make metadata-update` to
generate `storageos.clusterserviceversion.yaml`.

__NOTE__: The above step is incorporated in the `make release` command. All the
steps are automated to update and generate appropriate files.

Any other change in the CSV file must be made in
`deploy/storageos-operators.configmap.yaml` and regenerate all the metadata
files.

## Testing

Run `make olm-lint` from the project root to lint all the file in this
directory. A lint must be performed before submitting a new release to the
operatorhub.
