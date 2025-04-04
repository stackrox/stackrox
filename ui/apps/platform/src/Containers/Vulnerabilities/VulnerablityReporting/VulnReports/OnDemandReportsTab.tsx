import React from 'react';
import { Divider, Flex, PageSection, Text } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function OnDemandReportsTab() {
    return (
        <>
            <PageTitle title="Vulnerability reporting - On-demand reports" />
            <PageSection variant="light">
                <Text>
                    Check job status and download on-demand reports in CSV format. Requests are
                    purged according to retention settings.
                </Text>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}></PageSection>
        </>
    );
}

export default OnDemandReportsTab;
