# CSV rhel

All the Cluster Service Version (CSV) files for rhel operator releases are kept
in this directory. `storageos.clusterserviceversion.yaml` contains the latest
CSV. CSVs for releases have the release version in the file name
`storageos.<version>.clusterserviceversion.yaml`. A released CSV file must not
be modified because releases are versioned and can't be rereleased. For any
change, a new CSV with incremented version must be created.

## Release Instructions

To create a new release:
1. Update `metadata.annotations.containerImage` and deployment image with the
new operator container image and `spec.version` to the new release version
number in `storageos.clusterserviceversion.yaml`.
2. Run `make metadata-zip` from the root of the project to generate a metadata
zip file at `/build/_output/storageos-olm-metadata.zip`. This file can be
directly uploaded to the rhel operator metadata scanner for a new release.
3. Copy the modified `storageos.clusterserviceversion.yaml` file into a new file
`storageos.<version>.clusterserviceversion.yaml` and check-in both the files in
the repo to keep a record of the releases.

## Testing

Run `make metadata-bundle-lint` from the root of the project to create a
metadata bundle and lint it before submitting a new release.
