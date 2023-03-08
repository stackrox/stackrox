import { gql, useQuery } from '@apollo/client';
import { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

export type ImageVulnerabilitiesVariables = {
    id: string;
    vulnQuery: string;
};

export type ImageVulnerabilitiesResponse = {
    image: {
        id: string;
        imageVulnerabilities: {
            severity: string;
            isFixable: boolean;
            cve: string;
            summary: string;
            discoveredAtImage: Date | null;
            imageComponents: {
                id: string;
                name: string;
                version: string;
                fixedIn: string;
                location: string;
                layerIndex: number | null;
            }[];
        }[];
    };
};

export const imageVulnerabilitiesQuery = gql`
    query getImageVulnerabilities($id: ID!, $vulnQuery: String!) {
        image(id: $id) {
            id
            imageVulnerabilities(query: $vulnQuery) {
                severity
                isFixable
                cve
                summary
                discoveredAtImage
                imageComponents {
                    id
                    name
                    version
                    fixedIn
                    location
                    layerIndex
                }
            }
        }
    }
`;

export default function useImageVulnerabilities(imageId: string, searchFilter: SearchFilter) {
    return useQuery<ImageVulnerabilitiesResponse, ImageVulnerabilitiesVariables>(
        imageVulnerabilitiesQuery,
        {
            variables: {
                id: imageId,
                vulnQuery: getRequestQueryStringForSearchFilter(searchFilter),
            },
        }
    );
}
