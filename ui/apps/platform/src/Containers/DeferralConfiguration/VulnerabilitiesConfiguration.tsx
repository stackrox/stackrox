import React from 'react';
import {
    Button,
    Divider,
    PageSection,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';

function VulnerabilitiesConfiguration() {
    return (
        <>
            <div className="pf-u-py-md pf-u-px-md pf-u-px-lg-on-xl">
                <Split className="pf-u-align-items-center">
                    <SplitItem isFilled>
                        <Text>Configure deferral behavior for vulnerabilities</Text>
                    </SplitItem>
                    <SplitItem>
                        <Button variant="primary">Save</Button>
                    </SplitItem>
                </Split>
            </div>
            <Divider component="div" />
            <PageSection variant="light">
                <Title headingLevel="h2">Configure deferral times</Title>
            </PageSection>
        </>
    );
}

export default VulnerabilitiesConfiguration;
