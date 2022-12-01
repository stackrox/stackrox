import React, { ReactElement } from 'react';
import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    PageSection,
    Title,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Tooltip,
} from '@patternfly/react-core';

import { integrationsPath } from 'routePaths';
import {
    getEditDisabledMessage,
    getIntegrationLabel,
} from 'Containers/Integrations/utils/integrationUtils';
import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useIntegrationPermissions from './hooks/useIntegrationPermissions';
import usePageState from './hooks/usePageState';

export type IntegrationPageProps = {
    title: string;
    children: ReactElement | ReactElement[];
};

function IntegrationPage({ title, children }: IntegrationPageProps): ReactElement {
    const permissions = useIntegrationPermissions();
    const {
        pageState,
        params: { source, type, id },
    } = usePageState();
    const typeLabel = getIntegrationLabel(source, type);

    const integrationsListPath = `${integrationsPath}/${source}/${type}`;
    const integrationEditPath = `${integrationsPath}/${source}/${type}/edit/${id as string}`;

    const editDisabledMessage = getEditDisabledMessage(type);

    return (
        <>
            <PageTitle title={title} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                    <BreadcrumbItemLink to={integrationsListPath}>{typeLabel}</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex>
                    <FlexItem>
                        <Title headingLevel="h1">{`${
                            pageState === 'VIEW_DETAILS' ? '' : 'Configure'
                        } ${typeLabel} Integration`}</Title>
                    </FlexItem>
                    {pageState === 'VIEW_DETAILS' && permissions[source].write && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            {editDisabledMessage ? (
                                <Tooltip content={editDisabledMessage}>
                                    <Button
                                        variant={ButtonVariant.secondary}
                                        component={LinkShim}
                                        href={integrationEditPath}
                                        isAriaDisabled={!!editDisabledMessage}
                                    >
                                        Edit
                                    </Button>
                                </Tooltip>
                            ) : (
                                <Button
                                    variant={ButtonVariant.secondary}
                                    component={LinkShim}
                                    href={integrationEditPath}
                                >
                                    Edit
                                </Button>
                            )}
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            {children}
        </>
    );
}

export default IntegrationPage;
