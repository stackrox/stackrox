import { Flex, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function SecretsPage() {
    return (
        <>
            <PageTitle title="Risk - Secrets" />
            <PageSection>
                <Flex direction={{ default: 'column' }}>
                    <Title headingLevel="h1">Secrets</Title>
                </Flex>
            </PageSection>
        </>
    );
}

export default SecretsPage;
