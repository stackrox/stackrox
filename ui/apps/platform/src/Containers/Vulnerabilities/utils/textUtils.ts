const urlToLinkTextTuples = [
    ['www.cve.org', 'View at cve.org'],
    ['security.alpinelinux.org', 'View in Alpine CVE database'],
    ['alas.aws.amazon.com', 'View in Amazon CVE database'],
    ['security-tracker.debian.org', 'View in Debian CVE database'],
    ['access.redhat.com', 'View in Red Hat CVE database'],
    ['cve.mitre.org', 'View in MITRE CVE database'],
    ['linux.oracle.com', 'View in Oracle CVE database'],
    ['ubuntu.com', 'View in Ubuntu CVE database'],
    ['osv.dev', 'View in OSV CVE database'],
];

export function getDistroLinkText({ link }: { link: string }): string {
    const [, vendorText] = urlToLinkTextTuples.find(([url]) => link.includes(url)) ?? [];

    if (vendorText) {
        return vendorText;
    }

    try {
        const url = new URL(link);
        return `View additional information at ${url.host}`;
    } catch {
        return 'View additional information';
    }
}
