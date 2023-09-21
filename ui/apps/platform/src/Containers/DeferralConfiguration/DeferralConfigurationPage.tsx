import React from 'react';
import { PageSection, Tab, Tabs, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import VulnerabilitiesConfiguration from './VulnerabilitiesConfiguration';

const deferralConfigurationCategories = ['Vulnerabilities'] as const;

function DeferralConfigurationPage() {
    const [category, setCategory] = useURLStringUnion('category', deferralConfigurationCategories);

    return (
        <>
            <PageTitle title="Deferral configuration" />
            <PageSection variant="light">
                <Title headingLevel="h1">Deferral configuration</Title>
            </PageSection>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={category}
                    onSelect={(e, value) => setCategory(value)}
                    usePageInsets
                    mountOnEnter
                >
                    <Tab eventKey="Vulnerabilities" title="Vulnerabilities">
                        <VulnerabilitiesConfiguration />
                    </Tab>
                </Tabs>
            </PageSection>
        </>
    );
}

export default DeferralConfigurationPage;
