import React, { ReactElement, useState } from 'react';
import { PageSection, Title, Breadcrumb, BreadcrumbItem, Divider } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';
import { connect } from 'react-redux';

import { actions as integrationsActions } from 'reducers/integrations';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as clusterInitBundlesActions } from 'reducers/clusterInitBundles';
import { integrationsPath } from 'routePaths';
import {
    Integration,
    getIsAPIToken,
    getIsClusterInitBundle,
    getIntegrationLabel,
} from 'Containers/Integrations/utils/integrationUtils';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import IntegrationsTable from './IntegrationsTable';
import useIntegrations from '../hooks/useIntegrations';
import GenericIntegrationModal from '../GenericIntegrationModal';
import ConfirmationModal from './ConfirmationModal';
import {
    DeleteAPITokensConfirmationText,
    DeleteClusterInitBundlesConfirmationText,
    DeleteIntegrationsConfirmationText,
} from './ConfirmationTexts';

function IntegrationsListPage({
    deleteIntegrations,
    revokeAPITokens,
    revokeClusterInitBundles,
}): ReactElement {
    const { source, type } = useParams();
    const [selectedIntegration, setSelectedIntegration] = useState<
        Integration | Record<string, unknown> | null
    >(null);
    const integrations = useIntegrations({ source, type });
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState([]);

    const typeLabel = getIntegrationLabel(source, type);
    const isAPIToken = getIsAPIToken(source, type);
    const isClusterInitBundle = getIsClusterInitBundle(source, type);

    function closeModal() {
        setSelectedIntegration(null);
    }

    function onEditIntegration(integration) {
        setSelectedIntegration(integration);
    }

    function onViewIntegration(integration) {
        setSelectedIntegration(integration);
    }

    function onDeleteIntegrations(ids) {
        setDeletingIntegrationIds(ids);
    }

    function onConfirmDeletingIntegrationIds() {
        if (isAPIToken) {
            revokeAPITokens(deletingIntegrationIds);
        } else if (isClusterInitBundle) {
            revokeClusterInitBundles(deletingIntegrationIds);
        } else {
            deleteIntegrations(source, type, deletingIntegrationIds);
        }
        setDeletingIntegrationIds([]);
    }

    function onCancelDeleteIntegrationIds() {
        setDeletingIntegrationIds([]);
    }

    function onCreateIntegration() {
        setSelectedIntegration({});
    }

    return (
        <>
            <PageTitle title={typeLabel} />
            <PageSection variant="light">
                <div className="pf-u-mb-sm">
                    <Breadcrumb>
                        <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                        <BreadcrumbItem isActive>{typeLabel}</BreadcrumbItem>
                    </Breadcrumb>
                </div>
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <Divider component="div" />
            <IntegrationsTable
                title={typeLabel}
                integrations={integrations}
                onCreateIntegration={onCreateIntegration}
                onEditIntegration={onEditIntegration}
                onDeleteIntegrations={onDeleteIntegrations}
                onViewIntegration={
                    isClusterInitBundle || isAPIToken ? onViewIntegration : undefined
                }
            />
            {selectedIntegration && (
                <GenericIntegrationModal
                    integrations={integrations}
                    source={source}
                    type={type}
                    label={typeLabel}
                    onRequestClose={closeModal}
                    selectedIntegration={selectedIntegration}
                />
            )}
            {isAPIToken && (
                <ConfirmationModal
                    isOpen={deletingIntegrationIds.length !== 0}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                >
                    <DeleteAPITokensConfirmationText
                        numIntegrations={deletingIntegrationIds.length}
                    />
                </ConfirmationModal>
            )}
            {isClusterInitBundle && (
                <ConfirmationModal
                    isOpen={deletingIntegrationIds.length !== 0}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                >
                    <DeleteClusterInitBundlesConfirmationText
                        numIntegrations={deletingIntegrationIds.length}
                    />
                </ConfirmationModal>
            )}
            {!isAPIToken && !isClusterInitBundle && (
                <ConfirmationModal
                    isOpen={deletingIntegrationIds.length !== 0}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                >
                    <DeleteIntegrationsConfirmationText
                        numIntegrations={deletingIntegrationIds.length}
                    />
                </ConfirmationModal>
            )}
        </>
    );
}

const mapDispatchToProps = {
    deleteIntegrations: integrationsActions.deleteIntegrations,
    revokeAPITokens: apitokensActions.revokeAPITokens,
    revokeClusterInitBundles: clusterInitBundlesActions.revokeClusterInitBundles,
};

export default connect(null, mapDispatchToProps)(IntegrationsListPage);
