import React from 'react';
import { PageSection, Title, Divider, Flex, FlexItem, Button } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function VulnReportsPage() {
    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-py-lg pf-u-px-lg"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign distribution
                                lists to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    <FlexItem>
                        <Button variant="primary" onClick={() => {}}>
                            Create report
                        </Button>
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }} />
        </>
    );
}

export default VulnReportsPage;
