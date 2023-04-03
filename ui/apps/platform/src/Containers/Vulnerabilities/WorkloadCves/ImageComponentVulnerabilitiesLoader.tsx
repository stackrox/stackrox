import React from 'react';
import { gql, useQuery } from '@apollo/client';

import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ComponentVulnerabilitiesTable, {
    ComponentVulnerabilities,
    componentVulnerabilitiesFragment,
    ImageMetadataContext,
} from './Tables/ComponentVulnerabilitiesTable';

const imageComponentVulnerabilitiesQuery = gql`
    query getImageComponentVulnerabilities($imageId: ID!, $vulnCveQuery: String!) {
        image(id: $imageId) {
            id
            ...ComponentVulnerabilities
        }
    }
    ${componentVulnerabilitiesFragment}
`;

export type ImageComponentVulnerabilitiesProps = {
    /** Whether to load the data or not when rendering, used to lazy load component vulns */
    isActive: boolean;
    cveId: string;
    image: ImageMetadataContext;
};

// TODO Need counterpart DeploymentComponentVulnerabilitiesLoader that loads multiple images
function ImageComponentVulnerabilitiesLoader({
    isActive,
    cveId,
    image,
}: ImageComponentVulnerabilitiesProps) {
    const { data, loading, error } = useQuery<
        { image: ComponentVulnerabilities },
        {
            imageId: string;
            vulnCveQuery: string;
        }
    >(imageComponentVulnerabilitiesQuery, {
        variables: {
            imageId: image.id,
            vulnCveQuery: getRequestQueryStringForSearchFilter({ CVE: [cveId] }),
        },
        skip: !isActive,
    });

    if (loading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert
                title="There was an error loading the component vulnerabilities for this CVE"
                isInline
                variant="danger"
            >
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    return (
        <ComponentVulnerabilitiesTable
            showImage={false}
            images={[
                {
                    context: image,
                    componentVulnerabilities: data?.image?.imageComponents ?? [],
                },
            ]}
        />
    );
}

export default ImageComponentVulnerabilitiesLoader;
