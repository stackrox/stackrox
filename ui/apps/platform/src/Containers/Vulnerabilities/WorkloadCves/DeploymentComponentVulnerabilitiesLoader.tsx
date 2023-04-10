import React from 'react';
import { gql, useQuery } from '@apollo/client';

import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ComponentVulnerabilitiesTable, {
    ComponentVulnerabilities,
    ImageMetadataContext,
    componentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './Tables/ComponentVulnerabilitiesTable';

const deploymentComponentVulnerabilitiesQuery = gql`
    query getDeploymentComponentVulnerabilities($deploymentId: ID!, $vulnCveQuery: String!) {
        deployment(id: $deploymentId) {
            images {
                ...ImageMetadataContext
                ...ComponentVulnerabilities
            }
        }
    }
    ${imageMetadataContextFragment}
    ${componentVulnerabilitiesFragment}
`;

export type DeploymentComponentVulnerabilitiesProps = {
    /** Whether to load the data or not when rendering, used to lazy load component vulns */
    isActive: boolean;
    cveId: string;
    deploymentId: string;
};

function ImageComponentVulnerabilitiesLoader({
    isActive,
    cveId,
    deploymentId,
}: DeploymentComponentVulnerabilitiesProps) {
    const { data, loading, error } = useQuery<
        { deployment: { images: (ImageMetadataContext & ComponentVulnerabilities)[] } },
        {
            deploymentId: string;
            vulnCveQuery: string;
        }
    >(deploymentComponentVulnerabilitiesQuery, {
        variables: {
            deploymentId,
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

    const images = data?.deployment?.images
        ? data.deployment.images.map((image) => ({
              context: image,
              componentVulnerabilities: image.imageComponents,
          }))
        : [];

    return <ComponentVulnerabilitiesTable showImage images={images} />;
}

export default ImageComponentVulnerabilitiesLoader;
