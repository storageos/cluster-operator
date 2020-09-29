## crds/

This directory contains CRDs and their examples. The examples should contain
all the available configurations for reference needs.

The files in this directory are used by the operator-sdk e2e test.


## olm/

This directory contains OLM related files.

`olm/olm.sh` contains helper functions to setup OLM and install storageos
operator.

`olm/storageos` contains the files that are submitted to the operatorhub for
operatorhub release.

`../internal/olm-deploy/storageos-operators.olm.cr.yaml` contains a custom
resource for storageos cluster.
**NOTE**: This file is moved to `../internal/olm-deploy/` to prevent
`operator-sdk generate csv` from using the custom resource in the generated CSV
file as an example CR. Any CR manifest in `deploy/` and its subdirectories are
automatically used as example in the generated CSV.
