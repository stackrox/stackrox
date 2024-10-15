import {
    getRedactedOriginProperties,
    redactedHostReplacement,
    redactedSearchReplacement,
} from './useAnalytics';

describe('getRedactedOriginProperties', () => {
    test('scrubs installation-specific host value', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com/main/compliance/coverage/profiles/ocp4-pci-dss-4-0/checks'
            ).url
        ).toEqual(
            `https://${redactedHostReplacement}/main/compliance/coverage/profiles/ocp4-pci-dss-4-0/checks`
        );
    });

    test('scrubs installation-specific search value', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?s[Cluster][0]=control-cluster'
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?s[Cluster][0]=${redactedSearchReplacement}`,
            search: `?s[Cluster][0]=${redactedSearchReplacement}`,
            referrer: '',
        });
    });

    test('does not scrub allow-listed search keys', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?s[Cluster][0]=control-cluster&s[Compliance%20Check%20Name][0]=apiserver'
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?s[Cluster][0]=${redactedSearchReplacement}&s[Compliance%20Check%20Name][0]=apiserver`,
            search: `?s[Cluster][0]=${redactedSearchReplacement}&s[Compliance%20Check%20Name][0]=apiserver`,
            referrer: '',
        });
    });

    test('scrubs top level string search values', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com/main/compliance/controls?s[groupBy]=CLUSTER&s[standard]=NIST%20SP%20800-53&s[bogus]=bogus'
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/controls?s[groupBy]=CLUSTER&s[standard]=NIST%20SP%20800-53&s[bogus]=${redactedSearchReplacement}`,
            search: `?s[groupBy]=CLUSTER&s[standard]=NIST%20SP%20800-53&s[bogus]=${redactedSearchReplacement}`,
            referrer: '',
        });
    });

    test('scrubs only s and s2 search keys', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com/main/compliance/controls?s[bogus%20standard]=NIST%20SP%20800-190&s2[bogus%20standard]=NIST%20SP%20800-190&s3[bogus%20standard]=NIST%20SP%20800-190'
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/controls?s[bogus%20standard]=${redactedSearchReplacement}&s2[bogus%20standard]=${redactedSearchReplacement}&s3[bogus%20standard]=NIST%20SP%20800-190`,
            search: `?s[bogus%20standard]=${redactedSearchReplacement}&s2[bogus%20standard]=${redactedSearchReplacement}&s3[bogus%20standard]=NIST%20SP%20800-190`,
            referrer: '',
        });
    });

    test('scrubs mixed search value structures', () => {
        expect(
            getRedactedOriginProperties(
                'https://example.stackrox.com:443/main/network-graph?s[External Hostname]=test&s[Cluster]=control-cluster&s[Namespace][0]=cert-manager&s[Namespace][1]=kube-public&s[Deployment][0]=cert-manager&s[Deployment][1]=cert-manager&simulation=networkPolicy'
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/network-graph?s[External%20Hostname]=${redactedSearchReplacement}&s[Cluster]=${redactedSearchReplacement}&s[Namespace][0]=${redactedSearchReplacement}&s[Namespace][1]=${redactedSearchReplacement}&s[Deployment][0]=${redactedSearchReplacement}&s[Deployment][1]=${redactedSearchReplacement}&simulation=networkPolicy`,
            search: `?s[External%20Hostname]=${redactedSearchReplacement}&s[Cluster]=${redactedSearchReplacement}&s[Namespace][0]=${redactedSearchReplacement}&s[Namespace][1]=${redactedSearchReplacement}&s[Deployment][0]=${redactedSearchReplacement}&s[Deployment][1]=${redactedSearchReplacement}&simulation=networkPolicy`,
            referrer: '',
        });
    });
});
