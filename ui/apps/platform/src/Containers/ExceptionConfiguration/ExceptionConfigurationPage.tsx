import React from 'react';
import { PageSection, Tab, Tabs, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import VulnerabilitiesConfiguration from './VulnerabilitiesConfiguration';

const exceptionConfigurationCategories = ['Vulnerabilities'] as const;

function ExceptionConfigurationPage() {
    const [category, setCategory] = useURLStringUnion('category', exceptionConfigurationCategories);

    return (
        <>
            <PageTitle title="Exception configuration" />
            <PageSection variant="light">
                <Title headingLevel="h1">Exception configuration</Title>
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

export default ExceptionConfigurationPage;
