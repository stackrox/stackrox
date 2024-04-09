import React from 'react';
import { PageSection, Text } from '@patternfly/react-core';

// eslint-disable-next-line @typescript-eslint/ban-types
export type NodePageDetailsProps = {};

// eslint-disable-next-line no-empty-pattern
function NodePageDetails({}: NodePageDetailsProps) {
    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>View details about this node</Text>
            </PageSection>
            <PageSection isFilled className="pf-u-display-flex pf-u-flex-direction-column">
                <div className="pf-u-flex-grow-1 pf-u-background-color-100">Details</div>
            </PageSection>
        </>
    );
}

export default NodePageDetails;
