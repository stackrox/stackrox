import { gql, useQuery } from '@apollo/client';
import { Alert, Content, Spinner } from '@patternfly/react-core';

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
};

function CVESummaryContent({ cve }: CVESummaryContentProps) {
    const { data, loading, error } = useQuery(CVE_SUMMARY_QUERY, {
        variables: { cve },
    });

    if (loading) {
        return <Spinner size="md" />;
    }

    if (error) {
        return <Alert component="p" variant="warning" isInline isPlain title="Unable to load CVE summary" />;
    }

    const distroTuples: { summary: string; operatingSystem: string }[] =
        data?.imageCVE?.distroTuples ?? [];
    const prioritized = sortCveDistroList(distroTuples);
    const summary = prioritized.length > 0 ? prioritized[0].summary : '';

    if (!summary) {
        return <Alert component="p" variant="info" isInline isPlain title="No summary available for this CVE" />;
    }

    return <Content component="p">{summary}</Content>;
}

export default CVESummaryContent;
