import { useCallback } from 'react';
import { gql, useQuery } from '@apollo/client';
import { Alert, Content, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchCVEDetail } from 'services/VulnMgmtService';
import { sortCveDistroList } from '../../utils/sortUtils';

const CVE_SUMMARY_QUERY = gql`
    query getCVESummary($cve: String) {
        imageCVE(cve: $cve) {
            cve
            distroTuples {
                summary
                operatingSystem
            }
        }
    }
`;

type CVESummaryContentProps = {
    cve: string;
    useREST?: boolean;
};

function CVESummaryContentGraphQL({ cve }: { cve: string }) {
    const { data, loading, error } = useQuery(CVE_SUMMARY_QUERY, {
        variables: { cve },
    });

    if (loading) {
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

    const distroTuples: { summary: string; operatingSystem: string }[] =
        data?.imageCVE?.distroTuples ?? [];
    const prioritized = sortCveDistroList(distroTuples);
    const summary = prioritized.length > 0 ? prioritized[0].summary : '';

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

function CVESummaryContentREST({ cve }: { cve: string }) {
    const requestFn = useCallback(() => fetchCVEDetail(cve), [cve]);
    const { data, isLoading, error } = useRestQuery(requestFn);

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

function CVESummaryContent({ cve, useREST = false }: CVESummaryContentProps) {
    if (useREST) {
        return <CVESummaryContentREST cve={cve} />;
    }
    return <CVESummaryContentGraphQL cve={cve} />;
}

export default CVESummaryContent;
