# IBM midstream/downstream CICD workflow
This doc outlines the details and context pertaining to the CI/CD approach used by IBM Power and Z platforms.

## Context
 When the process of enabling Stackrox CI for `ppc64le` and `s390x` platforms was started, there were issues with building `central` and `scanner` upstream due to which we resorted to using the already available midstream and downstream builds for `ACS`. As it currently stands, only `scanner v2` build is pending upstream for these platforms.

## Details
Stackrox components are deployed on Openshift Clusters using RH ACS operator. The operator is deployed in the cluster using Index Image Build (IIB). For example, the build info is obtained from [here](#http://external-ci-coldstorage.datahub.redhat.com/cvp/cvp-redhat-operator-bundle-image-validation-test/rhacs-operator-bundle-container-4.4.0-13/5b908d5c-7406-4f05-8cba-081933de2b24/index_images.yml).

The current build info is present in [iib.json](iib.json)

The stackrox deployment and test execution entrypoint is defined in https://github.com/stackrox/stackrox/blob/master/qa-tests-backend/scripts/run-custom-pz.sh


## Contacts
For any issues or failures in the IBM CI please reach out to,

* ppc64le: mdafsan.hossain@ibm.com or prathamm@us.ibm.com

* s390x: joseph.dao@ibm.com