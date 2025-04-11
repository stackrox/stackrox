import React, { ReactElement, useEffect, useState } from 'react';
import {
    PageSection,
    PageSectionVariants,
    Title,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
} from '@patternfly/react-core';
import { useParams, useNavigate } from 'react-router-dom';
import { connect } from 'react-redux';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useCentralCapabilities from 'hooks/useCentralCapabilities';
import { actions as integrationsActions } from 'reducers/integrations';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as machineAccessActions } from 'reducers/machineAccessConfigs';
import { actions as cloudSourcesActions } from 'reducers/cloudSources';
import { integrationsPath } from 'routePaths';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';
import useIntegrations from '../hooks/useIntegrations';
import { getIntegrationLabel } from '../utils/integrationsList';
import {
    getIsAPIToken,
    getIsCloudSource,
    getIsMachineAccessConfig,
    getIsSignatureIntegration,
    getIsScannerV4,
    IntegrationSource,
    IntegrationType,
} from '../utils/integrationUtils';

import {
    DeleteAPITokensConfirmationText,
    DeleteIntegrationsConfirmationText,
} from './ConfirmationTexts';
import IntegrationsTable from './IntegrationsTable';

function IntegrationsListPage({
    deleteIntegrations,
    triggerBackup,
    revokeAPITokens,
    deleteMachineAccessConfigs,
    deleteCloudSources,
}): ReactElement {
    const { source, type } = useParams() as { source: IntegrationSource; type: IntegrationType };
    const integrations = useIntegrations({ source, type });
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState([]);

    const navigate = useNavigate();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );
    useEffect(() => {
        if (!canUseCloudBackupIntegrations && source === 'backups') {
            navigate(integrationsPath, { replace: true });
        }
    }, [canUseCloudBackupIntegrations, source, navigate]);

    const typeLabel = getIntegrationLabel(source, type);
    const isAPIToken = getIsAPIToken(source, type);
    const isMachineAccessConfig = getIsMachineAccessConfig(source, type);
    const isSignatureIntegration = getIsSignatureIntegration(source);
    const isScannerV4 = getIsScannerV4(source, type);
    const isCloudSource = getIsCloudSource(source);

    // There is currently nothing relevant in Tech Preview.
    const isTechPreview = false;

    function onDeleteIntegrations(ids) {
        setDeletingIntegrationIds(ids);
    }

    function onConfirmDeletingIntegrationIds() {
        if (isAPIToken) {
            revokeAPITokens(deletingIntegrationIds);
        } else if (isMachineAccessConfig) {
            deleteMachineAccessConfigs(deletingIntegrationIds);
        } else if (isCloudSource) {
            deleteCloudSources(deletingIntegrationIds);
        } else {
            deleteIntegrations(source, type, deletingIntegrationIds);
        }
        setDeletingIntegrationIds([]);
    }

    function onCancelDeleteIntegrationIds() {
        setDeletingIntegrationIds([]);
    }

    return (
        <>
            <PageTitle title={typeLabel} />
            <PageSection variant={PageSectionVariants.light} className="pf-v5-u-py-md">
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
                {!isSignatureIntegration && (
                    <Title headingLevel="h2">
                        <Flex
                            spaceItems={{ default: 'spaceItemsSm' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                        >
                            <span>{typeLabel}</span>
                            {isTechPreview && <TechPreviewLabel />}
                        </Flex>
                    </Title>
                )}
            </PageSection>
            <PageSection variant="default">
                <IntegrationsTable
                    integrations={integrations}
                    hasMultipleDelete
                    onDeleteIntegrations={onDeleteIntegrations}
                    onTriggerBackup={triggerBackup}
                    isReadOnly={isScannerV4}
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
            {!isAPIToken && (
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
    revokeAPITokens: apitokensActions.revokeAPITokens,
    deleteMachineAccessConfigs: machineAccessActions.deleteMachineAccessConfigs,
    deleteCloudSources: cloudSourcesActions.deleteCloudSources,
};

export default connect(null, mapDispatchToProps)(IntegrationsListPage);
