import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Breadcrumb,
    BreadcrumbItem,
    Flex,
    PageSection,
    Title,
    pluralize,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useRestMutation from 'hooks/useRestMutation';
import useToasts from 'hooks/patternfly/useToasts';
import type { Toast } from 'hooks/patternfly/useToasts';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { integrationsPath } from 'routePaths';
import {
    deleteIntegrations as serviceDeleteIntegrations,
    isServiceIntegrationSource,
} from 'services/IntegrationsService';
import { revokeAPITokens as serviceRevokeAPITokens } from 'services/APITokensService';
import { deleteMachineAccessConfigs as serviceDeleteMachineAccessConfigs } from 'services/MachineAccessService';
import { deleteCloudSources as serviceDeleteCloudSources } from 'services/CloudSourceService';
import { triggerBackup as serviceTriggerBackup } from 'services/BackupIntegrationsService';

import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import useIntegrations from '../hooks/useIntegrations';
import useFetchIntegrations from '../hooks/useFetchIntegrations';
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
};

function IntegrationsListPage({ source, type }: IntegrationsListPageProps): ReactElement {
    const integrations = useIntegrations({ source, type });
    const fetchIntegrations = useFetchIntegrations(source);
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState<string[]>([]);
    const { toasts, addToast, removeToast } = useToasts();

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

    const deleteMutation = useRestMutation(
        (ids: string[]) => {
            if (isAPIToken) {
                return serviceRevokeAPITokens(ids);
            }
            if (isMachineAccessConfig) {
                return serviceDeleteMachineAccessConfigs(ids);
            }
            if (isCloudSource) {
                return serviceDeleteCloudSources(ids);
            }
            if (isServiceIntegrationSource(source)) {
                return serviceDeleteIntegrations(source, ids);
            }
            return Promise.reject(new Error('Invalid integration source'));
        },
        {
            onSuccess: () => {
                const count = deletingIntegrationIds.length;
                addToast(
                    `Successfully deleted ${count} ${pluralize(count, 'integration')}`,
                    'success'
                );
                setDeletingIntegrationIds([]);
                fetchIntegrations();
            },
        }
    );

    const backupMutation = useRestMutation(serviceTriggerBackup, {
        onSuccess: () => addToast('Backup was successful', 'success'),
        onError: (error) => addToast(`Backup failed: ${getAxiosErrorMessage(error)}`, 'danger'),
    });

    function onDeleteIntegrations(ids: string[]) {
        setDeletingIntegrationIds(ids);
    }

    function onConfirmDeletingIntegrationIds() {
        deleteMutation.mutate(deletingIntegrationIds);
    }

    function onCancelDeleteIntegrationIds() {
        deleteMutation.reset();
        setDeletingIntegrationIds([]);
    }

    function onTriggerBackup(id: string) {
        backupMutation.mutate(id);
    }

    return (
        <>
            <PageTitle title={typeLabel} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={integrationsPath}>Integrations</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{typeLabel}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
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
                            {isTechPreview && <TechnologyPreviewLabel />}
                        </Flex>
                    </Title>
                )}
            </PageSection>
            <PageSection>
                <IntegrationsTable
                    tableState={tableState}
                    hasMultipleDelete
                    onDeleteIntegrations={onDeleteIntegrations}
                    onTriggerBackup={onTriggerBackup}
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
                    isLoading={deleteMutation.isLoading}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                    title="Delete API token"
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        {deleteMutation.isError && (
                            <Alert variant="danger" isInline title="Failed to delete" component="p">
                                {getAxiosErrorMessage(deleteMutation.error)}
                            </Alert>
                        )}
                        <DeleteAPITokensConfirmationText
                            numIntegrations={deletingIntegrationIds.length}
                        />
                    </Flex>
                </ConfirmationModal>
            )}
            {!isAPIToken && (
                <ConfirmationModal
                    ariaLabel="Confirm delete"
                    confirmText="Delete"
                    isOpen={deletingIntegrationIds.length !== 0}
                    isLoading={deleteMutation.isLoading}
                    onConfirm={onConfirmDeletingIntegrationIds}
                    onCancel={onCancelDeleteIntegrationIds}
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        {deleteMutation.isError && (
                            <Alert variant="danger" isInline title="Failed to delete" component="p">
                                {getAxiosErrorMessage(deleteMutation.error)}
                            </Alert>
                        )}
                        <DeleteIntegrationsConfirmationText
                            numIntegrations={deletingIntegrationIds.length}
                        />
                    </Flex>
                </ConfirmationModal>
            )}
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        variant={variant}
                        title={title}
                        component="p"
                        timeout={4000}
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={`${variant} alert`}
                                onClose={() => removeToast(key)}
                            />
                        }
                        key={key}
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default IntegrationsListPage;
