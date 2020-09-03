## csv-rhel/

This directory contains all the CSV files for rhel releases.

## storageos/

This directory contains all the CSV files for community operator releases. It
also contains CRD and package files. This directory is also used to create a
diff and submit a PR to the community operator repo for publishing new
release. rhel releases uses the CRD and package files from this directory.

## olm.sh

This script contains helper functions to setup OLM in a cluster, install
the storageos operator and install storageos. Also includes scripts to
uninstallation everything it installs.

## community-changes.yaml, rhel-changes.yaml, community-package-changes.yaml, rhel-package-changes.yaml

These files are update scripts for the yaml files used by yq yaml processor.
