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
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PageTitle from 'Components/PageTitle';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useRestMutation from 'hooks/useRestMutation';
import useToasts from 'hooks/patternfly/useToasts';
import type { Toast } from 'hooks/patternfly/useToasts';
import { getTableUIState } from 'utils/getTableUIState';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { integrationsPath } from 'routePaths';
import { deleteIntegrations, isServiceIntegrationSource } from 'services/IntegrationsService';
import { revokeAPITokens } from 'services/APITokensService';
import { deleteMachineAccessConfigs } from 'services/MachineAccessService';
import { deleteCloudSources } from 'services/CloudSourceService';
import { triggerBackup } from 'services/BackupIntegrationsService';

import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';
import useIntegrations from '../hooks/useIntegrations';
import { getIntegrationLabel } from '../utils/integrationsList';
import {
    getIsAPIToken,
    getIsCloudSource,
    getIsGCR,
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
    const { integrations, isLoading, error, refetch } = useIntegrations({ source, type });
    const [deletingIntegrationIds, setDeletingIntegrationIds] = useState<string[]>([]);
    const { toasts, addToast, removeToast } = useToasts();

    const tableState = getTableUIState({
        isLoading,
        data: integrations,
        error,
        searchFilter: {},
    });

    const typeLabel = getIntegrationLabel(source, type);
    const isAPIToken = getIsAPIToken(source, type);
    const isMachineAccessConfig = getIsMachineAccessConfig(source, type);
    const isSignatureIntegration = getIsSignatureIntegration(source);
    const isScannerV4 = getIsScannerV4(source, type);
    const isGCR = getIsGCR(source, type);
    const isCloudSource = getIsCloudSource(source);

    // There is currently nothing relevant in Tech Preview.
    const isTechPreview = false;

    const deleteMutation = useRestMutation(
        (ids: string[]) => {
            if (isAPIToken) {
                return revokeAPITokens(ids);
            }
            if (isMachineAccessConfig) {
                return deleteMachineAccessConfigs(ids);
            }
            if (isCloudSource) {
                return deleteCloudSources(ids);
            }
            if (isServiceIntegrationSource(source)) {
                return deleteIntegrations(source, ids).then(() => undefined);
            }
            return Promise.reject(new Error('Invalid integration source'));
        },
        {
            onSuccess: () => {
                const count = deletingIntegrationIds.length;
                addToast(`Successfully deleted ${pluralize(count, 'integration')}`, 'success');
                setDeletingIntegrationIds([]);
                refetch();
            },
        }
    );

    const backupMutation = useRestMutation(triggerBackup, {
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
            {isGCR && (
                <PageSection>
                    <Alert title="Deprecation notice" component="p" variant="warning" isInline>
                        Google Container Registry has been deprecated by Google. New integrations
                        cannot be created. Use Google Artifact Registry instead. See the{' '}
                        <ExternalLink>
                            <a
                                href="https://cloud.google.com/container-registry/docs/deprecations/container-registry-deprecation"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Container Registry deprecation notice
                            </a>
                        </ExternalLink>{' '}
                        for more information.
                    </Alert>
                </PageSection>
            )}
            <PageSection>
                <IntegrationsTable
                    tableState={tableState}
                    hasMultipleDelete
                    onDeleteIntegrations={onDeleteIntegrations}
                    onTriggerBackup={onTriggerBackup}
                    isReadOnly={isScannerV4}
                    isCreationDisabled={isGCR}
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
