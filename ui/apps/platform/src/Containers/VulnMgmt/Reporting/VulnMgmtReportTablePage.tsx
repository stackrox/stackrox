import React, { ReactElement } from 'react';
import { PageSection, PageSectionVariants, Text, TextContent, Title } from '@patternfly/react-core';

function ReportingTablePage(): ReactElement {
    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <TextContent>
                    <Title headingLevel="h1">Vulnerability reporting</Title>
                    <Text component="p">
                        Configure reports, define resource scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </Text>
                </TextContent>
            </PageSection>
            <PageSection variant={PageSectionVariants.light}>table goes here</PageSection>
        </>
    );
}

export default ReportingTablePage;
