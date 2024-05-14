import React, { ReactElement, useState } from 'react';
import {
    PageSection,
    PageSectionVariants,
    Title,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
} from '@patternfly/react-core';
import { useParams, useHistory } from 'react-router-dom';
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
import useFeatureFlags from 'hooks/useFeatureFlags';
import useIntegrations from '../hooks/useIntegrations';
import { getIntegrationLabel } from '../utils/integrationsList';
import {
    getIsAPIToken,
    getIsCloudSource,
    getIsClusterInitBundle,
    getIsMachineAccessConfig,
    getIsSignatureIntegration,
    getIsScannerV4,
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
    const { source, type } = useParams();
    const integrations = useIntegrations({ source, type });
    const { isFeatureFlagEnabled } = useFeatureFlags();
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
    const isMachineAccessConfig = getIsMachineAccessConfig(source, type);
    const isSignatureIntegration = getIsSignatureIntegration(source);
    const isScannerV4 = getIsScannerV4(source, type);
    const isCloudSource = getIsCloudSource(source);

    const isTechPreview = isFeatureFlagEnabled('ROX_SCANNER_V4') && type === 'scannerv4';

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
                    hasMultipleDelete={!isClusterInitBundle}
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
    revokeAPITokens: apitokensActions.revokeAPITokens,
    deleteMachineAccessConfigs: machineAccessActions.deleteMachineAccessConfigs,
    deleteCloudSources: cloudSourcesActions.deleteCloudSources,
};

export default connect(null, mapDispatchToProps)(IntegrationsListPage);
