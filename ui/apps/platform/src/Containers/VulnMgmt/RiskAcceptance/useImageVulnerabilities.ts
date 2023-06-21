import { useApolloClient, useQuery } from '@apollo/client';
import {
    GetImageVulnerabilitiesData,
    GetImageVulnerabilitiesVars,
    GET_IMAGE_VULNERABILITIES,
} from './imageVulnerabilities.graphql';

function useImageVulnerabilities({ imageId, vulnsQuery, pagination }) {
    const client = useApolloClient();
    const queryToUse = GET_IMAGE_VULNERABILITIES;

    const {
        loading: isLoading,
        data,
        error,
    } = useQuery<GetImageVulnerabilitiesData, GetImageVulnerabilitiesVars>(queryToUse, {
        variables: {
            imageId,
            vulnsQuery,
            pagination,
        },
        fetchPolicy: 'network-only',
    });

    async function refetchQuery() {
        await client.refetchQueries({
            include: [GET_IMAGE_VULNERABILITIES],
        });
    }

    return { isLoading, data, error, refetchQuery };
}

export default useImageVulnerabilities;
