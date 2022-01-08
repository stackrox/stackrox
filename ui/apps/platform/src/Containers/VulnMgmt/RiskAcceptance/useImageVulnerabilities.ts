import { useApolloClient, useQuery } from '@apollo/client';
import { useEffect, useState } from 'react';
import { fetchVulnRequests } from 'services/VulnerabilityRequestsService';
import {
    GetImageVulnerabilitiesData,
    GetImageVulnerabilitiesVars,
    GET_IMAGE_VULNERABILITIES,
} from './imageVulnerabilities.graphql';
import { combineVulnsWithVulnRequests } from './utils/imageVulnerabilitiesUtils';

function useImageVulnerabilities({ imageId, vulnsQuery, pagination }) {
    const [lastTimeRefetched, setLastTimeRefetched] = useState(() => new Date());
    const [data, setData] = useState<GetImageVulnerabilitiesData>();

    const client = useApolloClient();
    const {
        loading: isLoading,
        data: vulnsData,
        error,
    } = useQuery<GetImageVulnerabilitiesData, GetImageVulnerabilitiesVars>(
        GET_IMAGE_VULNERABILITIES,
        {
            variables: {
                imageId,
                vulnsQuery,
                pagination,
            },
            fetchPolicy: 'network-only',
        }
    );

    useEffect(() => {
        if (vulnsData) {
            const cves = vulnsData.image.vulns.map((vuln) => vuln.cve).join(',');
            const vulnRequestsQuery = vulnsData.image.vulns.length ? `CVE:${cves}` : '';
            fetchVulnRequests({ query: vulnRequestsQuery })
                .then((vulnRequests) => {
                    const { vulns } = vulnsData.image;
                    const newVulns = combineVulnsWithVulnRequests(vulns, vulnRequests);
                    const newVulnsData: GetImageVulnerabilitiesData = {
                        image: {
                            name: vulnsData.image.name,
                            vulns: newVulns,
                            vulnCount: vulnsData.image.vulnCount,
                        },
                    };
                    setData(newVulnsData);
                })
                .catch(() => {
                    // @TODO: Handle error
                });
        }
        // since sometimes when we refetch, the vulns data won't change, we need another indicator for refecthing
    }, [vulnsData, lastTimeRefetched]);

    async function refetchQuery() {
        await client.refetchQueries({
            include: [GET_IMAGE_VULNERABILITIES],
        });
        setLastTimeRefetched(new Date());
    }

    return { isLoading, data, error, refetchQuery };
}

export default useImageVulnerabilities;
