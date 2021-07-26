import React, { ReactElement } from 'react';
import { PageSection, Title, Breadcrumb, BreadcrumbItem, Divider } from '@patternfly/react-core';
import { integrationsPath } from 'routePaths';
import { getIntegrationLabel } from 'Containers/Integrations/utils/integrationUtils';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import usePageState from './hooks/usePageState';

export type IntegrationPageProps = {
    title: string;
    children: ReactElement | ReactElement[];
};

function IntegrationPage({ title, children }: IntegrationPageProps): ReactElement {
    const {
        params: { source, type },
    } = usePageState();
    const typeLabel = getIntegrationLabel(source, type);

    const integrationsListPath = `${integrationsPath}/${source}/${type}`;

    return (
        <>
            <PageTitle title={title} />
            <PageSection variant="light">
                <div className="pf-u-mb-sm">
                    <Breadcrumb>
                        <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                        <BreadcrumbItemLink to={integrationsListPath}>
                            {typeLabel}
                        </BreadcrumbItemLink>
                        <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                    </Breadcrumb>
                </div>
                <Title headingLevel="h1">Configure {typeLabel} Integration</Title>
            </PageSection>
            <Divider component="div" />
            {children}
        </>
    );
}

export default IntegrationPage;
