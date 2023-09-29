import React, { ReactElement, ReactNode } from 'react';
import { Gallery, PageSection, Title } from '@patternfly/react-core';

type IntegrationsSectionProps = {
    children: ReactNode;
    headerName: string;
    id: string;
};

const IntegrationsSection = ({
    children,
    headerName,
    id,
}: IntegrationsSectionProps): ReactElement => {
    return (
        <PageSection variant="light" id={id} className="pf-u-mb-xl">
            <Title headingLevel="h2" className="pf-u-mb-md">
                {headerName}
            </Title>
            <Gallery hasGutter>{children}</Gallery>
        </PageSection>
    );
};

export default IntegrationsSection;
