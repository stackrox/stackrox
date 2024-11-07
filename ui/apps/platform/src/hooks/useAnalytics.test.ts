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
        const originalSearchParams = ['s[Cluster][0]=control-cluster'].join('&');
        const redactedSearchParams = [`s[Cluster][0]=${redactedSearchReplacement}`].join('&');
        expect(
            getRedactedOriginProperties(
                `https://example.stackrox.com/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?${originalSearchParams}`
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?${redactedSearchParams}`,
            search: `?${redactedSearchParams}`,
            referrer: '',
        });
    });

    test('does not scrub allow-listed search keys', () => {
        const originalSearchParams = [
            's[Cluster][0]=control-cluster',
            's[Compliance%20Check%20Name][0]=apiserver',
        ].join('&');
        const redactedSearchParams = [
            `s[Cluster][0]=${redactedSearchReplacement}`,
            's[Compliance%20Check%20Name][0]=apiserver',
        ].join('&');
        expect(
            getRedactedOriginProperties(
                `https://example.stackrox.com/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?${originalSearchParams}`
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/coverage/profiles/ocp4-cis-node-1-5/checks?${redactedSearchParams}`,
            search: `?${redactedSearchParams}`,
            referrer: '',
        });
    });

    test('scrubs top level string search values', () => {
        const originalSearchParams = [
            's[groupBy]=CLUSTER',
            's[standard]=NIST%20SP%20800-53',
            's[bogus]=bogus',
        ].join('&');
        const redactedSearchParams = [
            's[groupBy]=CLUSTER',
            's[standard]=NIST%20SP%20800-53',
            `s[bogus]=${redactedSearchReplacement}`,
        ].join('&');
        expect(
            getRedactedOriginProperties(
                `https://example.stackrox.com/main/compliance/controls?${originalSearchParams}`
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/controls?${redactedSearchParams}`,
            search: `?${redactedSearchParams}`,
            referrer: '',
        });
    });

    test('scrubs only s and s2 search keys', () => {
        const originalSearchParams = [
            's[bogus%20standard]=NIST%20SP%20800-190',
            's2[bogus%20standard]=NIST%20SP%20800-190',
            's3[bogus%20standard]=NIST%20SP%20800-190',
        ].join('&');
        const redactedSearchParams = [
            `s[bogus%20standard]=${redactedSearchReplacement}`,
            `s2[bogus%20standard]=${redactedSearchReplacement}`,
            's3[bogus%20standard]=NIST%20SP%20800-190',
        ].join('&');
        expect(
            getRedactedOriginProperties(
                `https://example.stackrox.com/main/compliance/controls?${originalSearchParams}`
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/compliance/controls?${redactedSearchParams}`,
            search: `?${redactedSearchParams}`,
            referrer: '',
        });
    });

    test('scrubs mixed search value structures', () => {
        const originalSearchParams = [
            's[External%20Hostname]=test',
            's[Cluster]=control-cluster',
            's[Namespace][0]=cert-manager',
            's[Namespace][1]=kube-public',
            's[Deployment][0]=cert-manager',
            's[Deployment][1]=cert-manager',
            'simulation=networkPolicy',
        ].join('&');
        const redactedSearchParams = [
            `s[External%20Hostname]=${redactedSearchReplacement}`,
            `s[Cluster]=${redactedSearchReplacement}`,
            `s[Namespace][0]=${redactedSearchReplacement}`,
            `s[Namespace][1]=${redactedSearchReplacement}`,
            `s[Deployment][0]=${redactedSearchReplacement}`,
            `s[Deployment][1]=${redactedSearchReplacement}`,
            'simulation=networkPolicy',
        ].join('&');
        expect(
            getRedactedOriginProperties(
                `https://example.stackrox.com:443/main/network-graph?${originalSearchParams}`
            )
        ).toEqual({
            url: `https://${redactedHostReplacement}/main/network-graph?${redactedSearchParams}`,
            search: `?${redactedSearchParams}`,
            referrer: '',
        });
    });
});
