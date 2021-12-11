import React, { ReactElement } from 'react';
import { Title, Divider, Flex, FlexItem } from '@patternfly/react-core';

import PolicyMetadataFormSection from './PolicyMetadataFormSection';
import AttachNotifiersFormSection from './AttachNotifiersFormSection';

function PolicyDetailsForm(): ReactElement {
    return (
        <div>
            <Title headingLevel="h2">Policy details</Title>
            <div className="pf-u-mb-md pf-u-mt-sm">
                Describe general information about your policy.
            </div>
            <Divider component="div" />
            <Flex direction={{ default: 'row' }}>
                <FlexItem className="pf-u-mb-md" grow={{ default: 'grow' }}>
                    <PolicyMetadataFormSection />
                </FlexItem>
                <Divider component="div" isVertical />
                <FlexItem className="pf-u-w-33">
                    <AttachNotifiersFormSection />
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <FlexItem>mitre attack stuff will go here</FlexItem>
        </div>
    );
}

export default PolicyDetailsForm;
