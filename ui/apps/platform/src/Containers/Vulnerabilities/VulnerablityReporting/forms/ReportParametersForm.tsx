import { Divider, Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

function ReportParametersForm(): ReactElement {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Configure report parameters</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure report&apos;s name, CVE attributes, and setup schedule to send
                        reports on a recurring basis.
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default ReportParametersForm;
