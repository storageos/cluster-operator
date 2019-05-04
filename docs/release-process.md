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
image set to tag `version`. Update `community-changes.yaml`, `rhel-changes.yaml`
and `package-changes.yaml` files with the new versions and run
`make metadata-update` from project root to automatically update all the CSV and
package files with new versions and images. Once tagged, these containers are
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
