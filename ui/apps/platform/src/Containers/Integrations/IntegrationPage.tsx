import React from 'react';
import type { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
    Tooltip,
} from '@patternfly/react-core';

import { integrationsPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import type { Traits } from 'types/traits.proto';
import TraitsOriginLabel from 'Components/TraitsOriginLabel';
import { isUserResource } from 'utils/traits.utils';
import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';
import { getIntegrationLabel } from './utils/integrationsList';
import { getEditDisabledMessage, getIsMachineAccessConfig } from './utils/integrationUtils';
import usePageState from './hooks/usePageState';
import useIntegrationPermissions from './hooks/useIntegrationPermissions';

export type IntegrationPageProps = {
    title: string;
    name: string;
    traits?: Traits;
    children: ReactElement | ReactElement[];
};

function IntegrationPage({ title, name, traits, children }: IntegrationPageProps): ReactElement {
    const permissions = useIntegrationPermissions();
    const {
        pageState,
        params: { source, type, id },
    } = usePageState();
    const typeLabel = getIntegrationLabel(source, type);
    // There is currently nothing relevant in Tech Preview.
    const isTechPreview = false;

    const integrationsListPath = `${integrationsPath}/${source}/${type}`;
    const integrationEditPath = `${integrationsPath}/${source}/${type}/edit/${id as string}`;

    const editDisabledMessage = getEditDisabledMessage(type);

    const hasTraitsLabel =
        pageState !== 'CREATE' &&
        pageState !== 'LIST' &&
        (type === 'generic' || type === 'splunk' || type === 'machineAccess');
    const hasEditButton =
        pageState === 'VIEW_DETAILS' && permissions[source].write && isUserResource(traits);
    return (
        <>
            <PageTitle title={title} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                    <BreadcrumbItemLink to={integrationsListPath}>{typeLabel}</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    {(name || getIsMachineAccessConfig(source, type)) && (
                        <FlexItem>
                            <Title headingLevel="h1">{name || 'Manage configuration'}</Title>
                        </FlexItem>
                    )}
                    {isTechPreview && (
                        <FlexItem>
                            <TechPreviewLabel />
                        </FlexItem>
                    )}
                    {hasTraitsLabel && <TraitsOriginLabel traits={traits} />}
                    {hasEditButton && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            {editDisabledMessage ? (
                                <Tooltip content={editDisabledMessage}>
                                    <Button
                                        variant="secondary"
                                        component={LinkShim}
                                        href={integrationEditPath}
                                        isAriaDisabled={!!editDisabledMessage}
                                    >
                                        Edit
                                    </Button>
                                </Tooltip>
                            ) : (
                                <Button
                                    variant="secondary"
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
