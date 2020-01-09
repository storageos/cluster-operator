# Release Process

The operator release involves:
1. Creating an operator container image.
2. Pushing the container image to container image repositories.
3. Generating a metadata zip file for publishing to rhel marketplace.
4. Creating a PR to [community-operator](https://github.com/operator-framework/community-operators/)
    for publishing to operatorhub.io.

All the above steps have been automated and are run in TravisCI builds every
time a new git tag is created.

The release requires
`deploy/olm/csv-rhel/storageos.v<version>.clusterserviceversion.yaml` and
`deploy/olm/storageos/storageos.v<version>.clusterserviceversion.yaml` to be
checked-in before `version` is tagged. These files should have the container
image set to tag `version`.

NOTE: The CSV must have related images in the operator deployment as environment
variables. Unlike before, the default version of StorageOS node container image
must be set in the CSV file. When there's a new version of node container, or
any other container images, update release-gen.sh with the new node container
tag in the operator changes template, before running the following release
generator command. Image update should apply to both operatorhub and rhel CSVs.

To create the CSV files and update all the associated files with the new release
version, run `NEW_VERSION=<version> make release`. This will run
`release-gen.sh`, updating all the intermediate files(`*-changes.yaml`) that are
used to process and update the CSV files, update the CSV `createdAt` property to
the current UTC time, run `update-metadata-files.sh` to update all the OLM
metadata files, and copy the latest CSVs from community and rhel to create
`deploy/olm/csv-rhel/storageos.v<version>.clusterserviceversion.yaml` and
`deploy/olm/storageos/storageos.v<version>.clusterserviceversion.yaml` files.
These changes can then be checked into the repo and a new release can be tagged.

Once tagged, these containers are
published using `scripts/deploy.sh`. rhel build is triggered at the same time to
publish a new container in rhel container registry. The metadata zip for rhel
release is generated and attached to the github release automatically. This file
can then be submitted to the rhel metadata scanner for a new rhel operator
release.

`scripts/create-pr.sh` creates a new PR to update the community-operator with
the new release. Once the PR is merged, a new version is released at
operatorhub.io.


# Deployment Secrets

The TravisCI github releases deployment provider requires api_key to publish the
generated artifacts. This api_key is encrypted and added in the `.travis.yml`
file at `deploy.api_key.secure`. The key can be encrypted by running:
```
$ travis encrypt <key>
```
The github personal access token requires repo access permission only.

For creating PR to community-operator, the same token stored as TravisCI env var
is used.
