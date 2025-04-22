import React from 'react';
import {
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    PageSectionVariants,
    Title,
} from '@patternfly/react-core';
import { ActionsColumn, Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { useParams, Link } from 'react-router-dom';
import pluralize from 'pluralize';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useTableSelection from 'hooks/useTableSelection';
import { allEnabled } from 'utils/featureFlagUtils';
import TableCellValue from 'Components/TableCellValue/TableCellValue';
import { isUserResource } from 'Containers/AccessControl/traits';
import useIntegrationPermissions from '../hooks/useIntegrationPermissions';
import usePageState from '../hooks/usePageState';
import {
    Integration,
    IntegrationSource,
    IntegrationType,
    getIsAPIToken,
} from '../utils/integrationUtils';
import tableColumnDescriptor from '../utils/tableColumnDescriptor';

function getNewButtonText(type) {
    if (type === 'apitoken') {
        return 'Generate token';
    }
    if (type === 'machineAccess') {
        return 'Create configuration';
    }
    return 'New integration';
}

type IntegrationsTableProps = {
    integrations: Integration[];
    hasMultipleDelete: boolean;
    onDeleteIntegrations: (integration) => void;
    onTriggerBackup: (integrationId) => void;
    isReadOnly?: boolean;
};

function IntegrationsTable({
    integrations,
    hasMultipleDelete,
    onDeleteIntegrations,
    onTriggerBackup,
    isReadOnly,
}: IntegrationsTableProps): React.ReactElement {
    const permissions = useIntegrationPermissions();
    const { source, type } = useParams() as { source: IntegrationSource; type: IntegrationType };
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
        const { featureFlagDependency } = integration;
        if (featureFlagDependency && featureFlagDependency.length > 0) {
            return allEnabled(featureFlagDependency)(isFeatureFlagEnabled);
        }
        return true;
    });

    const isAPIToken = getIsAPIToken(source, type);

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
                            {hasSelections &&
                                hasMultipleDelete &&
                                permissions[source].write &&
                                !isReadOnly && (
                                    <FlexItem>
                                        <Button
                                            variant="danger"
                                            onClick={onDeleteIntegrationHandler}
                                        >
                                            Delete {numSelected} selected{' '}
                                            {pluralize('integration', numSelected)}
                                        </Button>
                                    </FlexItem>
                                )}
                            {permissions[source].write && !isReadOnly && (
                                <FlexItem>
                                    <Button
                                        variant="primary"
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
                    <Table variant="compact" isStickyHeader>
                        <Thead>
                            <Tr>
                                {hasMultipleDelete && !isReadOnly && (
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
                                <Th>
                                    <span className="pf-v5-screen-reader">Row actions</span>
                                </Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {integrations.map((integration, rowIndex) => {
                                const { id } = integration;
                                const canTriggerBackup =
                                    integration.type === 's3' ||
                                    integration.type === 's3compatible' ||
                                    integration.type === 'gcs';
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
                                        isHidden: isAPIToken,
                                    },
                                    {
                                        title: (
                                            <div className="pf-v5-u-danger-color-100">
                                                Delete Integration
                                            </div>
                                        ),
                                        onClick: () => onDeleteIntegrations([integration.id]),
                                        isHidden: isReadOnly,
                                    },
                                ].filter((actionItem) => {
                                    return !actionItem?.isHidden;
                                });
                                return (
                                    <Tr key={integration.id}>
                                        {hasMultipleDelete && !isReadOnly && (
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
                                            if (
                                                column.Header === 'Name' ||
                                                (type === 'machineAccess' &&
                                                    column.Header === 'Configuration')
                                            ) {
                                                return (
                                                    <Td key="name" dataLabel={column.Header}>
                                                        <Link
                                                            to={getPathToViewDetails(
                                                                source,
                                                                type,
                                                                id
                                                            )}
                                                        >
                                                            <TableCellValue
                                                                row={integration}
                                                                column={column}
                                                            />
                                                        </Link>
                                                    </Td>
                                                );
                                            }
                                            return (
                                                <Td key={column.Header} dataLabel={column.Header}>
                                                    <TableCellValue
                                                        row={integration}
                                                        column={column}
                                                    />
                                                </Td>
                                            );
                                        })}
                                        <Td isActionCell>
                                            <ActionsColumn
                                                isDisabled={
                                                    !permissions[source].write ||
                                                    !isUserResource(integration.traits)
                                                }
                                                items={actionItems}
                                            />
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    </Table>
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
