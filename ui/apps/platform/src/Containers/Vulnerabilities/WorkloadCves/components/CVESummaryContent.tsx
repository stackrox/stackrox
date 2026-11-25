import { useCallback, useEffect } from 'react';
import { Alert, Content, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchCVEDetail } from 'services/VulnMgmtService';
import { useVulnMgmtCacheVersion } from '../Overview/useVulnMgmtCacheVersion';

export type CVESummaryContentProps = {
    cve: string;
    query?: string;
};

function CVESummaryContent({ cve, query }: CVESummaryContentProps) {
    const cacheVersion = useVulnMgmtCacheVersion();
    const requestFn = useCallback(() => fetchCVEDetail(cve, query), [cve, query]);
    const { data, isLoading, error, refetch } = useRestQuery(requestFn);

    // Refetch when a mutation invalidates the REST cache.
    useEffect(() => {
        if (cacheVersion > 0) {
            refetch();
        }
    }, [cacheVersion, refetch]);

    if (isLoading) {
        return <Spinner size="md" />;
    }

    if (error) {
        return (
            <Alert
                component="p"
                variant="warning"
                isInline
                isPlain
                title="Unable to load CVE summary"
            />
        );
    }

    const summary = data?.distroDetails?.[0]?.summary ?? '';
    if (!summary) {
        return (
            <Alert
                component="p"
                variant="info"
                isInline
                isPlain
                title="No summary available for this CVE"
            />
        );
    }

    return <Content component="p">{summary}</Content>;
}

export default CVESummaryContent;
