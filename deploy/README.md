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


## examples.json

This file contains json based examples of all the CRD resources. This exists
here to make it easier to edit the examples in a readable format. These examples
are added to the `alm-examples` field in an OLM ClusterServiceVersion(CSV)
object and are used as template for the resources. All the white spaces in this
file can be removed by using https://www.browserling.com/tools/remove-all-whitespace
and the result can be added as a single line value for `alm-examples`.
