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

// we may have to consider the release build in the future, where the version may be ahead of documentation that has not yet been created
function getVersionedDocs(completeVersion: string, subPath = 'welcome/index.html'): string {
    return `https://docs.openshift.com/acs/${getVersionMajorMinor(completeVersion)}/${subPath}`;
}

export { getVersionMajorMinor, getVersionedDocs };
