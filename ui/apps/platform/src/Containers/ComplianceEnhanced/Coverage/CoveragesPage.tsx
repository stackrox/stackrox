import React, { useContext } from 'react';
import { Bullseye, PageSection, Spinner, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

import { ComplianceProfilesContext } from './ComplianceProfilesProvider';

function CoveragesPage({ children }: { children: React.ReactNode }) {
    const context = useContext(ComplianceProfilesContext);

    if (!context) {
        return null;
    }

    const { isLoading, error } = context;

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return <div>Error: {error.message}</div>;
    }

    return (
        <>
            <PageTitle title="Compliance coverage" />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Compliance coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </PageSection>
            <PageSection>{children}</PageSection>
        </>
    );
}

export default CoveragesPage;
