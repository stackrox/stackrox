export type ExampleReportCSVData = {
    cluster: string;
    namespace: string;
    deployment: string;
    image: string;
    component: string;
    cve: string;
    fixable: string;
    componentUpgrade: string;
    severity: string;
    discoveredAt: string;
    reference: string;
};

const exampleReportsCSVData: ExampleReportCSVData[] = [
    {
        cluster: 'production',
        namespace: 'openshift-config-operator',
        deployment: 'openshift-config-operator',
        image: 'quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:a93ae9f08c38eb25ccd70aa44c08624199fe3f2d38efcb5d6627e83c8d165088',
        component: 'bzip2-libs',
        cve: 'CVE-2019-12900',
        fixable: 'false',
        componentUpgrade: '',
        severity: 'LOW',
        discoveredAt: 'January 26, 2022',
        reference: 'https://access.redhat.com/security/cve/CVE-2019-12900',
    },
    {
        cluster: 'production',
        namespace: 'openshift-config-operator',
        deployment: 'openshift-config-operator',
        image: 'quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:a93ae9f08c38eb25ccd70aa44c08624199fe3f2d38efcb5d6627e83c8d165088',
        component: 'cryptsetup-libs',
        cve: 'CVE-2021-4122',
        fixable: 'false',
        componentUpgrade: '',
        severity: 'MODERATE',
        discoveredAt: 'January 26, 2022',
        reference: 'https://access.redhat.com/security/cve/CVE-2021-4122',
    },
    {
        cluster: 'production',
        namespace: 'openshift-config-operator',
        deployment: 'openshift-config-operator',
        image: 'quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:a93ae9f08c38eb25ccd70aa44c08624199fe3f2d38efcb5d6627e83c8d165088',
        component: 'curl',
        cve: 'RHSA-2021:4511',
        fixable: 'true',
        componentUpgrade: '0:7.61.1-22.el8',
        severity: 'MODERATE',
        discoveredAt: 'January 26, 2022',
        reference: 'https://access.redhat.com/errata/RHSA-2021:4511',
    },
];

export default exampleReportsCSVData;
