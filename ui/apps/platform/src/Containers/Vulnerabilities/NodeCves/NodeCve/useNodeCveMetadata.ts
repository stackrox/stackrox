import { gql, useQuery } from '@apollo/client';

import { CveMetadata } from '../../components/CvePageHeader';

const metadataQuery = gql`
    query getNodeCVEMetadata($cve: String!) {
        nodeCVE(cve: $cve) {
            cve
            distroTuples {
                summary
                link
                operatingSystem
            }
            firstDiscoveredInSystem
        }
    }
`;

export default function useNodeCveMetadata(cveId: string) {
    const metadataRequest = useQuery<
        {
            nodeCVE?: CveMetadata;
        },
        { cve: string }
    >(metadataQuery, {
        variables: { cve: cveId },
    });

    const { data, previousData } = metadataRequest;
    const cveData = data?.nodeCVE ?? previousData?.nodeCVE;

    return {
        metadataRequest,
        cveData,
    };
}
