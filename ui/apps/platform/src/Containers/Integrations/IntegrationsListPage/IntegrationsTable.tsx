import React from 'react';
import {
    Button,
    ButtonVariant,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    PageSectionVariants,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { useParams, Link } from 'react-router-dom';
import pluralize from 'pluralize';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useTableSelection from 'hooks/useTableSelection';
import TableCellValue from 'Components/TableCellValue/TableCellValue';
import { isUserResource } from 'Containers/AccessControl/traits';
import useIntegrationPermissions from '../hooks/useIntegrationPermissions';
import usePageState from '../hooks/usePageState';
import { Integration, getIsAPIToken, getIsClusterInitBundle } from '../utils/integrationUtils';
import tableColumnDescriptor from '../utils/tableColumnDescriptor';
import DownloadCAConfigBundle from './DownloadCAConfigBundle';

function getNewButtonText(type) {
    if (type === 'apitoken') {
        return 'Generate token';
    }
    if (type === 'clusterInitBundle') {
        return 'Generate bundle';
    }
    return 'New integration';
}

type IntegrationsTableProps = {
    integrations: Integration[];
    hasMultipleDelete: boolean;
    onDeleteIntegrations: (integration) => void;
    onTriggerBackup: (integrationId) => void;
};

function IntegrationsTable({
    integrations,
    hasMultipleDelete,
    onDeleteIntegrations,
    onTriggerBackup,
}: IntegrationsTableProps): React.ReactElement {
    const permissions = useIntegrationPermissions();
    const { source, type } = useParams();
    const { getPathToCreate, getPathToEdit, getPathToViewDetails } = usePageState();
    const {
        selected,
        allRowsSelected,
        numSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        getSelectedIds,
    } = useTableSelection<Integration>(integrations);
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const columns = tableColumnDescriptor[source][type].filter((integration) => {
        if (typeof integration.featureFlagDependency === 'string') {
            return isFeatureFlagEnabled(integration.featureFlagDependency);
        }
        return true;
    });

    const isAPIToken = getIsAPIToken(source, type);
    const isClusterInitBundle = getIsClusterInitBundle(source, type);

    function onDeleteIntegrationHandler() {
        const ids = getSelectedIds();
        onDeleteIntegrations(ids);
    }

    const newButtonText = getNewButtonText(type);

    return (
        <>
            <PageSection variant="light">
                <Flex>
                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                        <Title headingLevel="h2">
                            {integrations.length} {pluralize('results', integrations.length)} found
                        </Title>
                    </FlexItem>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Flex>
                            {hasSelections && hasMultipleDelete && permissions[source].write && (
                                <FlexItem>
                                    <Button variant="danger" onClick={onDeleteIntegrationHandler}>
                                        Delete {numSelected} selected{' '}
                                        {pluralize('integration', numSelected)}
                                    </Button>
                                </FlexItem>
                            )}
                            {isClusterInitBundle && (
                                <FlexItem>
                                    <DownloadCAConfigBundle />
                                </FlexItem>
                            )}
                            {permissions[source].write && (
                                <FlexItem>
                                    <Button
                                        variant={ButtonVariant.primary}
                                        component={LinkShim}
                                        href={getPathToCreate(source, type)}
                                        data-testid="add-integration"
                                    >
                                        {newButtonText}
                                    </Button>
                                </FlexItem>
                            )}
                        </Flex>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection
                isFilled
                padding={{ default: 'noPadding' }}
                variant={PageSectionVariants.light}
            >
                {integrations.length > 0 ? (
                    <TableComposable variant="compact" isStickyHeader>
                        <Thead>
                            <Tr>
                                {hasMultipleDelete && (
                                    <Th
                                        select={{
                                            onSelect: onSelectAll,
                                            isSelected: allRowsSelected,
                                        }}
                                    />
                                )}
                                {columns.map((column) => {
                                    return (
                                        <Th key={column.Header} modifier="wrap">
                                            {column.Header}
                                        </Th>
                                    );
                                })}
                                <Td />
                            </Tr>
                        </Thead>
                        <Tbody>
                            {integrations.map((integration, rowIndex) => {
                                const { id } = integration;
                                const canTriggerBackup =
                                    integration.type === 's3' || integration.type === 'gcs';
                                const actionItems = [
                                    {
                                        title: 'Trigger backup',
                                        onClick: () => onTriggerBackup(integration.id),
                                        isHidden: !canTriggerBackup,
                                    },
                                    {
                                        title: (
                                            <Link to={getPathToEdit(source, type, id)}>
                                                Edit integration
                                            </Link>
                                        ),
                                        isHidden: isAPIToken || isClusterInitBundle,
                                    },
                                    {
                                        title: (
                                            <div className="pf-u-danger-color-100">
                                                Delete Integration
                                            </div>
                                        ),
                                        onClick: () => onDeleteIntegrations([integration.id]),
                                    },
                                ].filter((actionItem) => {
                                    return !actionItem?.isHidden;
                                });
                                return (
                                    <Tr key={integration.id}>
                                        {hasMultipleDelete && (
                                            <Td
                                                key={integration.id}
                                                select={{
                                                    rowIndex,
                                                    onSelect,
                                                    isSelected: selected[rowIndex],
                                                }}
                                            />
                                        )}
                                        {columns.map((column) => {
                                            if (column.Header === 'Name') {
                                                return (
                                                    <Td key="name">
                                                        <Button
                                                            variant={ButtonVariant.link}
                                                            isInline
                                                            component={LinkShim}
                                                            href={getPathToViewDetails(
                                                                source,
                                                                type,
                                                                id
                                                            )}
                                                        >
                                                            <TableCellValue
                                                                row={integration}
                                                                column={column}
                                                            />
                                                        </Button>
                                                    </Td>
                                                );
                                            }
                                            return (
                                                <Td key={column.Header}>
                                                    <TableCellValue
                                                        row={integration}
                                                        column={column}
                                                    />
                                                </Td>
                                            );
                                        })}
                                        <Td
                                            actions={{
                                                items: actionItems,
                                                disable:
                                                    !permissions[source].write ||
                                                    !isUserResource(integration.traits),
                                            }}
                                            className="pf-u-text-align-right"
                                        />
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    </TableComposable>
                ) : (
                    <EmptyStateTemplate
                        title="No integrations of this type are currently configured."
                        headingLevel="h3"
                    />
                )}
            </PageSection>
        </>
    );
}

export default IntegrationsTable;
