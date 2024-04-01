import React from 'react';
import { PageSection, Text } from '@patternfly/react-core';

// eslint-disable-next-line @typescript-eslint/ban-types
export type NodePageVulnerabilitiesProps = {};

// eslint-disable-next-line no-empty-pattern
function NodePageVulnerabilities({}: NodePageVulnerabilitiesProps) {
    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this node</Text>
            </PageSection>
            <PageSection isFilled className="pf-u-display-flex pf-u-flex-direction-column">
                <div className="pf-u-flex-grow-1 pf-u-background-color-100">vulns</div>
            </PageSection>
        </>
    );
}

export default NodePageVulnerabilities;
