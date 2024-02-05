/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useCallback } from 'react';
import { generatePath, Link, useHistory } from 'react-router-dom';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import {
    Alert,
    AlertGroup,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    List,
    ListItem,
    PageSection,
    Pagination,
    Spinner,
    Text,
    TextContent,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import {
    complianceEnhancedCoveragePath,
    complianceEnhancedScanConfigDetailPath,
    complianceEnhancedScanConfigsPath,
} from 'routePaths';
import DeleteModal from 'Components/PatternFly/DeleteModal';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import TabNavHeader from 'Components/TabNav/TabNavHeader';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import {
    getScanConfigs,
    getScanConfigsCount,
    deleteScanConfig,
    ComplianceScanConfigurationStatus,
} from 'services/ComplianceEnhancedService';
import { SortOption } from 'types/table';
import { displayOnlyItemOrItemCount } from 'utils/textUtils';

import { formatScanSchedule } from '../compliance.scanConfigs.utils';

type ScanConfigsTablePageProps = {
    hasWriteAccessForCompliance: boolean;
};

const CreateScanConfigButton = () => {
    return (
        <Link to={`${complianceEnhancedScanConfigsPath}?action=create`}>
            <Button variant="primary">Create scan schedule</Button>
        </Link>
    );
};

const sortFields = ['Compliance Scan Config Name'];
const defaultSortOption = {
    field: 'Compliance Scan Config Name',
    direction: 'asc',
} as SortOption;

function ScanConfigsTablePage({
    hasWriteAccessForCompliance,
}: ScanConfigsTablePageProps): React.ReactElement {
    const [scanConfigsToDelete, setScanConfigsToDelete] = useState<
        ComplianceScanConfigurationStatus[]
    >([]);
    const [scanConfigDeletionErrors, setScanConfigDeletionErrors] = useState<Error[]>([]);
    const [isDeletingScanConfigs, setIsDeletingScanConfigs] = useState(false);
    const history = useHistory();

    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    const listQuery = useCallback(
        () => getScanConfigs(sortOption, page - 1, perPage),
        [sortOption, page, perPage]
    );
    const { data: scanSchedules, loading: isLoading, error, refetch } = useRestQuery(listQuery);

    const countQuery = useCallback(() => getScanConfigsCount(), []);
    const { data: scanSchedulesCount } = useRestQuery(countQuery);

    function openDeleteModal(scanConfigs) {
        setScanConfigsToDelete(scanConfigs);
    }

    function closeDeleteScanConfigModal() {
        setScanConfigDeletionErrors([]);
        setScanConfigsToDelete([]);
    }

    function onDeleteScanConfig() {
        const deletePromises = scanConfigsToDelete.map((scanConfig) =>
            deleteScanConfig(scanConfig.id)
        );

        setScanConfigDeletionErrors([]);
        setIsDeletingScanConfigs(true);
        Promise.all(deletePromises)
            .then(() => {
                setScanConfigsToDelete([]);
                refetch();
            })
            .catch((errorResult) => {
                if (Array.isArray(errorResult)) {
                    errorResult.forEach((error) => {
                        setScanConfigDeletionErrors((prev) => [...prev, error as Error]);
                    });
                } else {
                    setScanConfigDeletionErrors([errorResult]);
                }
            })
            .finally(() => {
                setIsDeletingScanConfigs(false);
            });
    }

    const renderTableContent = () => {
        return scanSchedules?.map((scanSchedule) => {
            const { id, scanName, scanConfig, lastUpdatedTime, clusterStatus } = scanSchedule;
            const scanConfigUrl = generatePath(complianceEnhancedScanConfigDetailPath, {
                scanConfigId: id,
            });

            const rowActions = [
                {
                    title: 'Edit scan schedule',
                    onClick: (event) => {
                        event.preventDefault();
                        history.push({
                            pathname: scanConfigUrl,
                            search: 'action=edit',
                        });
                    },
                    isDisabled: !hasWriteAccessForCompliance,
                },
                {
                    title: 'Delete scan schedule',
                    onClick: (event) => {
                        event.preventDefault();
                        openDeleteModal([scanSchedule]);
                    },
                    isDisabled: !hasWriteAccessForCompliance,
                },
            ];

            return (
                <Tr key={id}>
                    <Td>
                        <Link to={scanConfigUrl}>{scanName}</Link>
                    </Td>
                    <Td>{formatScanSchedule(scanConfig.scanSchedule)}</Td>
                    <Td>{format(lastUpdatedTime, 'DD MMM YYYY, h:mm:ss A')}</Td>
                    <Td>
                        {displayOnlyItemOrItemCount(
                            clusterStatus.map((cluster) => cluster.clusterName),
                            'clusters'
                        )}
                    </Td>
                    <Td>{displayOnlyItemOrItemCount(scanConfig.profiles, 'profiles')}</Td>
                    <Td isActionCell>
                        <ActionsColumn menuAppendTo={() => document.body} items={rowActions} />
                    </Td>
                </Tr>
            );
        });
    };

    const renderLoadingContent = () => (
        <Tr>
            <Td colSpan={5}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={5}>
                <Bullseye>
                    <EmptyStateTemplate
                        title="No scan schedules"
                        headingLevel="h2"
                        icon={OutlinedClockIcon}
                    >
                        {hasWriteAccessForCompliance && (
                            <Flex direction={{ default: 'column' }}>
                                <FlexItem>
                                    <Text>Create one to get started</Text>
                                </FlexItem>
                                <FlexItem>
                                    <CreateScanConfigButton />
                                </FlexItem>
                            </Flex>
                        )}
                    </EmptyStateTemplate>
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderTableBodyContent = () => {
        if (isLoading) {
            return renderLoadingContent();
        }
        if (scanSchedules && scanSchedules.length > 0) {
            return renderTableContent();
        }
        return renderEmptyContent();
    };

    return (
        <>
            <TabNavHeader
                currentTabTitle="Schedules"
                tabLinks={[
                    { title: 'Coverage', href: complianceEnhancedCoveragePath },
                    { title: 'Schedules', href: complianceEnhancedScanConfigsPath },
                ]}
                pageTitle="Compliance - Cluster compliance"
                mainTitle="Cluster compliance"
            />
            <Divider component="div" />
            <TabNavSubHeader
                actions={hasWriteAccessForCompliance ? <CreateScanConfigButton /> : <></>}
                description="Configure a scan schedule to run profile compliance checks on selected clusters"
            />
            <Divider component="div" />
            {error ? (
                <PageSection variant="light" isFilled id="policies-table-error">
                    <Bullseye>
                        <Alert variant="danger" title={error} />
                    </Bullseye>
                </PageSection>
            ) : (
                <PageSection>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                                <Pagination
                                    itemCount={scanSchedulesCount ?? 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>

                    <TableComposable>
                        <Thead noWrap>
                            <Tr>
                                <Th sort={getSortParams('Compliance Scan Config Name')}>Name</Th>
                                <Th>Schedule</Th>
                                <Th>Last run</Th>
                                <Th>Clusters</Th>
                                <Th>Profiles</Th>
                                <Td />
                            </Tr>
                        </Thead>
                        <Tbody>{renderTableBodyContent()}</Tbody>
                    </TableComposable>
                    <DeleteModal
                        title={`Permanently delete scan (${scanConfigsToDelete.length}) ${pluralize(
                            'schedule',
                            scanConfigsToDelete.length
                        )}?`}
                        isOpen={scanConfigsToDelete.length > 0}
                        onClose={closeDeleteScanConfigModal}
                        isDeleting={isDeletingScanConfigs}
                        onDelete={onDeleteScanConfig}
                    >
                        {scanConfigDeletionErrors.length > 0 ? (
                            <AlertGroup>
                                {scanConfigDeletionErrors.map((deleteError) => {
                                    return (
                                        <Alert
                                            isInline
                                            variant="danger"
                                            title="Failed to delete"
                                            className="pf-u-mb-sm"
                                        >
                                            {deleteError.toString()}
                                        </Alert>
                                    );
                                })}
                            </AlertGroup>
                        ) : (
                            <></>
                        )}
                        <TextContent>
                            <Text>
                                The following scan{' '}
                                {`${pluralize('schedule', scanConfigsToDelete.length)}`} will be
                                deleted.
                            </Text>
                            <List>
                                {scanConfigsToDelete.map((scanConfig) => (
                                    <ListItem key={scanConfig.id}>{scanConfig.scanName}</ListItem>
                                ))}
                            </List>
                        </TextContent>
                    </DeleteModal>
                </PageSection>
            )}
        </>
    );
}

export default ScanConfigsTablePage;
