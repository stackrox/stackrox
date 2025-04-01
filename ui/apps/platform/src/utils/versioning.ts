function getVersionMajorMinor(version: string): string {
    if (version) {
        const results = /^(\d+)\.(\d+)/.exec(version);
        if (Array.isArray(results) && results.length === 3) {
            const [, versionMajor, versionMinor] = results;
            return `${versionMajor}.${versionMinor}`;
        }
    }
    return '';
}

const basePath =
    'https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes';

// we may have to consider the release build in the future, where the version may be ahead of documentation that has not yet been created
function getVersionedDocs(completeVersion: string, subPath?: string): string {
    const basePathWithVersion = `${basePath}/${getVersionMajorMinor(completeVersion)}`;
    return subPath ? `${basePathWithVersion}/html/${subPath}` : basePathWithVersion;
}

export { getVersionMajorMinor, getVersionedDocs };
