import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Button,
    ButtonVariant,
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
import { Traits } from 'types/traits.proto';
import { TraitsOriginLabel } from 'Containers/AccessControl/TraitsOriginLabel';
import { isUserResource } from 'Containers/AccessControl/traits';
import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';
import useFeatureFlags from 'hooks/useFeatureFlags';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const {
        pageState,
        params: { source, type, id },
    } = usePageState();
    const typeLabel = getIntegrationLabel(source, type);
    const isTechPreview = isFeatureFlagEnabled('ROX_SCANNER_V4') && type === 'scannerv4';

    const integrationsListPath = `${integrationsPath}/${source}/${type}`;
    const integrationEditPath = `${integrationsPath}/${source}/${type}/edit/${id as string}`;

    const editDisabledMessage = getEditDisabledMessage(type);

    const hasTraitsLabel =
        pageState !== 'CREATE' && pageState !== 'LIST' && (type === 'generic' || type === 'splunk');
    const hasEditButton =
        pageState === 'VIEW_DETAILS' && permissions[source].write && isUserResource(traits);
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
