import { useApolloClient, useQuery } from '@apollo/client';
import {
    GetImageVulnerabilitiesData,
    GetImageVulnerabilitiesVars,
    GET_IMAGE_VULNERABILITIES,
    GET_IMAGE_VULNERABILITIES_LEGACY,
} from './imageVulnerabilities.graphql';

function useImageVulnerabilities({ imageId, vulnsQuery, pagination, showVMUpdates = false }) {
    const client = useApolloClient();
    const queryToUse = showVMUpdates ? GET_IMAGE_VULNERABILITIES : GET_IMAGE_VULNERABILITIES_LEGACY;

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
        if (showVMUpdates) {
            await client.refetchQueries({
                include: [GET_IMAGE_VULNERABILITIES],
            });
        } else {
            await client.refetchQueries({
                include: [GET_IMAGE_VULNERABILITIES_LEGACY],
            });
        }
    }

    return { isLoading, data, error, refetchQuery };
}

export default useImageVulnerabilities;
