import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    PageSection,
    PageSectionVariants,
    Title,
} from '@patternfly/react-core';
import { connect } from 'react-redux';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import { actions as integrationsActions } from 'reducers/integrations';
import { actions as apitokensActions } from 'reducers/apitokens';
import { actions as machineAccessActions } from 'reducers/machineAccessConfigs';
import { actions as cloudSourcesActions } from 'reducers/cloudSources';
import { getTableUIState } from 'utils/getTableUIState';
import { integrationsPath } from 'routePaths';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';
import useIntegrations from '../hooks/useIntegrations';
import { getIntegrationLabel } from '../utils/integrationsList';
import {
    getIsAPIToken,
    getIsCloudSource,
    getIsMachineAccessConfig,
    getIsScannerV4,
    getIsSignatureIntegration,
} from '../utils/integrationUtils';
import type { IntegrationSource, IntegrationType } from '../utils/integrationUtils';

import {
    DeleteAPITokensConfirmationText,
    DeleteIntegrationsConfirmationText,
} from './ConfirmationTexts';
import IntegrationsTable from './IntegrationsTable';

export type IntegrationsListPageProps = {
    source: IntegrationSource;
    type: IntegrationType;
    // TODO replace actions and connect with service functions.
    deleteIntegrations: (source: IntegrationSource, type: IntegrationType, ids: string[]) => void;
    triggerBackup: () => void;
    revokeAPITokens: (ids: string[]) => void;
    deleteMachineAccessConfigs: (ids: string[]) => void;
    deleteCloudSources: (ids: string[]) => void;
};

function IntegrationsListPage({
    source,
    type,
    // TODO replace actions and connect with service functions.
    deleteIntegrations,
    triggerBackup,
    revokeAPITokens,
    deleteMachineAccessConfigs,
    deleteCloudSources,
}: IntegrationsListPageProps): ReactElement {
    const integrations = useIntegrations({ source, type });
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState([]);

    const tableState = getTableUIState({
        isLoading: false,
        data: integrations,
        error: undefined,
        searchFilter: {},
    });

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
                    tableState={tableState}
                    hasMultipleDelete
                    onDeleteIntegrations={onDeleteIntegrations}
                    onTriggerBackup={triggerBackup}
                    isReadOnly={isScannerV4}
                    source={source}
                    type={type}
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
