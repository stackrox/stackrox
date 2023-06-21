import React, { ReactElement, useState } from 'react';
import {
    PageSection,
    PageSectionVariants,
    Title,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
} from '@patternfly/react-core';
import { useParams, useHistory } from 'react-router-dom';
import { connect } from 'react-redux';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useCentralCapabilities from 'hooks/useCentralCapabilities';
import { actions as integrationsActions } from 'reducers/integrations';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as clusterInitBundlesActions } from 'reducers/clusterInitBundles';
import { integrationsPath } from 'routePaths';
import { ClusterInitBundle } from 'services/ClustersService';
import {
    getIsAPIToken,
    getIsClusterInitBundle,
    getIntegrationLabel,
    getIsSignatureIntegration,
} from 'Containers/Integrations/utils/integrationUtils';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import IntegrationsTable from './IntegrationsTable';
import useIntegrations from '../hooks/useIntegrations';
import {
    DeleteAPITokensConfirmationText,
    DeleteIntegrationsConfirmationText,
} from './ConfirmationTexts';
import DeleteClusterInitBundleConfirmationModal from './DeleteClusterInitBundleConfirmationModal';

function IntegrationsListPage({
    deleteIntegrations,
    triggerBackup,
    fetchClusterInitBundles,
    revokeAPITokens,
}): ReactElement {
    const { source, type } = useParams();
    const integrations = useIntegrations({ source, type });
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState([]);

    const history = useHistory();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );
    if (!canUseCloudBackupIntegrations && source === 'backups') {
        history.replace(integrationsPath);
    }

    const typeLabel = getIntegrationLabel(source, type);
    const isAPIToken = getIsAPIToken(source, type);
    const isClusterInitBundle = getIsClusterInitBundle(source, type);
    const isSignatureIntegration = getIsSignatureIntegration(source);

    function onDeleteIntegrations(ids) {
        setDeletingIntegrationIds(ids);
    }

    function onConfirmDeletingIntegrationIds() {
        if (isAPIToken) {
            revokeAPITokens(deletingIntegrationIds);
        } else {
            deleteIntegrations(source, type, deletingIntegrationIds);
        }
        setDeletingIntegrationIds([]);
    }

    function onCancelDeleteIntegrationIds() {
        setDeletingIntegrationIds([]);
    }

    /*
     * Instead of using bundleId arg to delete bundle from integrations in local state,
     * use Redux fetch action to indirectly update integrations and re-render the list,
     * because confirmation modal has already made the revokeClusterInitBundles request.
     */
    function handleDeleteClusterInitBundle() {
        setDeletingIntegrationIds([]);
        fetchClusterInitBundles();
    }

    return (
        <>
            <PageTitle title={typeLabel} />
            <PageSection variant={PageSectionVariants.light} className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{typeLabel}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Title headingLevel="h1">
                    {isSignatureIntegration ? 'Signature' : ''} Integrations
                </Title>
                {!isSignatureIntegration && <Title headingLevel="h2">{typeLabel}</Title>}
            </PageSection>
            <PageSection variant="default">
                <IntegrationsTable
                    integrations={integrations}
                    hasMultipleDelete={!isClusterInitBundle}
                    onDeleteIntegrations={onDeleteIntegrations}
                    onTriggerBackup={triggerBackup}
                />
            </PageSection>
            {isAPIToken && (
                <ConfirmationModal
                    ariaLabel="Confirm delete"
                    confirmText="Delete"
                    isOpen={deletingIntegrationIds.length !== 0}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                    title="Delete API token"
                >
                    <DeleteAPITokensConfirmationText
                        numIntegrations={deletingIntegrationIds.length}
                    />
                </ConfirmationModal>
            )}
            {isClusterInitBundle && (
                <DeleteClusterInitBundleConfirmationModal
                    bundle={
                        deletingIntegrationIds.length === 1
                            ? (integrations.find(
                                  (integration) => integration.id === deletingIntegrationIds[0]
                              ) as unknown as ClusterInitBundle)
                            : undefined
                    }
                    handleCancel={onCancelDeleteIntegrationIds}
                    handleDelete={handleDeleteClusterInitBundle}
                />
            )}
            {!isAPIToken && !isClusterInitBundle && (
                <ConfirmationModal
                    ariaLabel="Confirm delete"
                    confirmText="Delete"
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
    triggerBackup: integrationsActions.triggerBackup,
    fetchClusterInitBundles: clusterInitBundlesActions.fetchClusterInitBundles.request,
    revokeAPITokens: apitokensActions.revokeAPITokens,
};

export default connect(null, mapDispatchToProps)(IntegrationsListPage);
