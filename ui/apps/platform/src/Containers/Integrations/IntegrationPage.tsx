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
import {
    getEditDisabledMessage,
    getIntegrationLabel,
} from 'Containers/Integrations/utils/integrationUtils';
import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useIntegrationPermissions from './hooks/useIntegrationPermissions';
import usePageState from './hooks/usePageState';
import { Traits } from '../../types/traits.proto';
import { TraitsOriginLabel } from '../AccessControl/TraitsOriginLabel';
import { isUserResource } from '../AccessControl/traits';

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
                        <Title headingLevel="h1">{name}</Title>
                    </FlexItem>
                    {pageState !== 'CREATE' &&
                        pageState !== 'LIST' &&
                        (type === 'generic' || type === 'splunk') && (
                            <TraitsOriginLabel traits={traits} />
                        )}
                    {pageState === 'VIEW_DETAILS' &&
                        permissions[source].write &&
                        isUserResource(traits) && (
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
