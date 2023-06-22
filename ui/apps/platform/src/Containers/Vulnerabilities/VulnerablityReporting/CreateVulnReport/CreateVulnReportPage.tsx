import React from 'react';
import { PageSection, Title, Divider, Flex, FlexItem } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function VulnReportsPage() {
    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Create report</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure reports, define report scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }} />
        </>
    );
}

export default VulnReportsPage;
