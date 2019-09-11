#!/usr/bin/env bash

# This script runs the operator-sdk scorecard tests against an existing operator
# installed via OLM. OLM and storageos operator can be installed using the
# helpers in deploy/olm/olm.sh. An example usage of the same is in test/e2e.sh.
#
# The scorecard test includes basic and OLM tests that are
# run against each of the custom resources one at a time. It creates the custom
# resources, passed as `cr-manifest`, and analyzes the resources.
# The test "Writing into CRs has an effect" requires a proxy container to be
# deployed along with the operator. This is added in
# deploy/storageos-operators.configmap.yaml. The proxy container is removed from
# CSV in scripts/metadata-checker/update-metadata-files.sh when generating CSVs
# for a release.

# TODO: test/.osdk-scorecard.yaml contains a scorecard configuration file which
# will be supported in the upcoming versions of operator-sdk (> v0.10.0). Move
# .osdk-scorecard.yaml to the root of the project or run:
# $ operator-sdk scorecard --config test/.osdk-scorecard.yaml
# to run the same tests using config file. This script will no longer be
# required once the sdk fully supports scorecard config file.

# NOTE: The scorecard test behavior in operator-sdk 0.8.0 is simpler compared
# to 0.10.0. When operator-sdk version is updated to 0.10.0 or above, the tests
# will take extra time, waiting for actual custom resources to be created and
# analyze the properties of the resources. In order to reduce the test time, the
# CR controllers can be improved to add status to the created resources as soon
# as possible and cause some effect when there's a write to the custom resource.

PROJECT_ROOT=$PWD
CR_DIR="$PROJECT_ROOT/deploy/crds/"
CSV_PATH="deploy/olm/storageos/storageos.clusterserviceversion.yaml"
NAMESPACE="olm"

# Iterate through all the CR files and run scorecard test against them.
for cr_file in $(find "$CR_DIR" -name "*_cr.yaml"); do
  echo -e "\n\nRunning operator-sdk scorecard with example "$(cat "$cr_file" | yq r - "kind")""
  ./build/operator-sdk scorecard \
    --cr-manifest "$cr_file" \
    --csv-path $CSV_PATH \
    --olm-deployed \
    --namespace $NAMESPACE \
    --verbose

  # If scorecard errors out, print out operator logs
  if [[ $? != 0 ]]; then
    echo -e "\nFAIL: Scorecard test errored out."
    operatorpod=$(kubectl -n $NAMESPACE get pods --template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep storageos-operator)
    kubectl -n $NAMESPACE logs $operatorpod -c storageos-operator
    exit 1
  else
    echo -e "\nPASS: Scorecard test passed"
  fi
done
