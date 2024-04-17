import React from 'react';
import { PageSection, Text } from '@patternfly/react-core';

// eslint-disable-next-line @typescript-eslint/ban-types
export type ClusterPageDetailsProps = {};

// eslint-disable-next-line no-empty-pattern
function ClusterPageDetails({}: ClusterPageDetailsProps) {
    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-u-px-xl">
                <Text>View details about this cluster</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100">Details</div>
            </PageSection>
        </>
    );
}

export default ClusterPageDetails;
