import React from 'react';
import { Flex, FlexItem, PageSection, Title } from '@patternfly/react-core';

type ComplianceReportType = 'scan' | 'cluster' | 'profile';

type ComplianceReportsProps = {
    type: ComplianceReportType;
};

// TODO: remove disabled linter, type will eventually be used to slightly alter view
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ComplianceReports({ type }: ComplianceReportsProps) {
    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-pl-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Compliance Reports</Title>
                    </FlexItem>
                    <FlexItem>
                        Prioritize and manage scanned controls across profiles and clusters
                    </FlexItem>
                </Flex>
            </PageSection>
        </>
    );
}

export default ComplianceReports;
