import React from 'react';
import { PageSection, Text, Divider } from '@patternfly/react-core';

function ImageSingleVulnerabilities() {
    return (
        <>
            <PageSection variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this image</Text>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default ImageSingleVulnerabilities;
