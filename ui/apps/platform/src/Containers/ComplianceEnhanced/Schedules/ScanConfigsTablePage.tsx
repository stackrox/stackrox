/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useCallback } from 'react';
import { generatePath, Link, useHistory } from 'react-router-dom';
import pluralize from 'pluralize';

import {
    Alert,
    AlertActionCloseButton,
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
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import { complianceEnhancedCoveragePath, complianceEnhancedSchedulesPath } from 'routePaths';
import DeleteModal from 'Components/PatternFly/DeleteModal';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import PageTitle from 'Components/PageTitle';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import useAlert from 'hooks/useAlert';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import {
    ComplianceScanConfigurationStatus,
    deleteComplianceScanConfiguration,
    listComplianceScanConfigurations,
    runComplianceReport,
    runComplianceScanConfiguration,
} from 'services/ComplianceScanConfigurationService';
import { SortOption } from 'types/table';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { displayOnlyItemOrItemCount } from 'utils/textUtils';

import { DEFAULT_COMPLIANCE_PAGE_SIZE, SCAN_CONFIG_NAME_QUERY } from '../compliance.constants';
import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';
import {
    formatScanSchedule,
    getTimeWithHourMinuteFromISO8601,
} from './compliance.scanConfigs.utils';
import ScanConfigActionsColumn from './ScanConfigActionsColumn';

type ScanConfigsTablePageProps = {
    hasWriteAccessForCompliance: boolean;
};

const CreateScanConfigButton = () => {
    return (
        <Link to={`${complianceEnhancedSchedulesPath}?action=create`}>
            <Button variant="primary">Create scan schedule</Button>
        </Link>
    );
};

const sortFields = [SCAN_CONFIG_NAME_QUERY];
const defaultSortOption = {
    field: SCAN_CONFIG_NAME_QUERY,
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

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    const listQuery = useCallback(
        () => listComplianceScanConfigurations(sortOption, page - 1, perPage),
        [sortOption, page, perPage]
    );
    const { data: listData, loading: isLoading, error, refetch } = useRestQuery(listQuery);

    const { alertObj, setAlertObj, clearAlertObj } = useAlert();

    const colSpan = hasWriteAccessForCompliance ? 6 : 5;

    function openDeleteModal(scanConfigs) {
        setScanConfigsToDelete(scanConfigs);
    }

    function closeDeleteScanConfigModal() {
        setScanConfigDeletionErrors([]);
        setScanConfigsToDelete([]);
    }

    function onDeleteScanConfig() {
        const deletePromises = scanConfigsToDelete.map((scanConfig) =>
            deleteComplianceScanConfiguration(scanConfig.id)
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

    function handleDeleteScanConfig(scanConfigResponse: ComplianceScanConfigurationStatus) {
        openDeleteModal([scanConfigResponse]);
    }

    function handleRunScanConfig(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        runComplianceScanConfiguration(scanConfigResponse.id)
            .then(() => {
                setAlertObj({
                    type: 'success',
                    title: 'Successfully triggered a re-scan',
                });
                refetch(); // TODO verify is lastExecutedTime expected to change?
            })
            .catch((error) => {
                setAlertObj({
                    type: 'danger',
                    title: 'Could not trigger a re-scan',
                    children: getAxiosErrorMessage(error),
                });
            });
    }

    function handleSendReport(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        runComplianceReport(scanConfigResponse.id)
            .then(() => {
                setAlertObj({
                    type: 'success',
                    title: 'Successfully requested to send a report',
                });
            })
            .catch((error) => {
                setAlertObj({
                    type: 'danger',
                    title: 'Could not send a report',
                    children: getAxiosErrorMessage(error),
                });
            });
    }

    const renderTableContent = () => {
        return listData?.configurations?.map((scanSchedule) => {
            const { id, scanName, scanConfig, lastExecutedTime, clusterStatus } = scanSchedule;
            const scanConfigUrl = generatePath(scanConfigDetailsPath, {
                scanConfigId: id,
            });

            return (
                <Tr key={id}>
                    <Td dataLabel="Name">
                        <Link to={scanConfigUrl}>{scanName}</Link>
                    </Td>
                    <Td dataLabel="Schedule">{formatScanSchedule(scanConfig.scanSchedule)}</Td>
                    <Td dataLabel="Last run">
                        {lastExecutedTime
                            ? getTimeWithHourMinuteFromISO8601(lastExecutedTime)
                            : 'Scanning now'}
                    </Td>
                    <Td dataLabel="Clusters">
                        {displayOnlyItemOrItemCount(
                            clusterStatus.map((cluster) => cluster.clusterName),
                            'clusters'
                        )}
                    </Td>
                    <Td dataLabel="Profiles">
                        {displayOnlyItemOrItemCount(scanConfig.profiles, 'profiles')}
                    </Td>
                    {hasWriteAccessForCompliance && (
                        <Td isActionCell>
                            <ScanConfigActionsColumn
                                handleDeleteScanConfig={handleDeleteScanConfig}
                                handleRunScanConfig={handleRunScanConfig}
                                handleSendReport={handleSendReport}
                                scanConfigResponse={scanSchedule}
                            />
                        </Td>
                    )}
                </Tr>
            );
        });
    };

    const renderLoadingContent = () => (
        <Tr>
            <Td colSpan={colSpan}>
                <Bullseye>
                    <Spinner />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={colSpan}>
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
        if (Array.isArray(listData?.configurations) && listData.configurations.length > 0) {
            return renderTableContent();
        }
        return renderEmptyContent();
    };

    return (
        <>
            <PageTitle title="Compliance - Schedules" />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsCenter' }}>
                    <Flex direction={{ default: 'column' }}>
                        <Title headingLevel="h1">Schedules</Title>
                        <Text>
                            Configure scan schedules to run profile compliance checks on selected
                            clusters
                        </Text>
                    </Flex>
                    {hasWriteAccessForCompliance && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            <CreateScanConfigButton />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            {error ? (
                <PageSection variant="light" isFilled id="policies-table-error">
                    <Bullseye>
                        <Alert variant="danger" title={getAxiosErrorMessage(error)} />
                    </Bullseye>
                </PageSection>
            ) : (
                <PageSection>
                    {alertObj !== null && (
                        <Alert
                            title={alertObj.title}
                            variant={alertObj.type}
                            isInline
                            className="pf-v5-u-mb-lg"
                            component="h2"
                            actionClose={<AlertActionCloseButton onClose={clearAlertObj} />}
                        >
                            {alertObj.children}
                        </Alert>
                    )}

                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                                <Pagination
                                    itemCount={listData?.totalCount ?? 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>

                    <Table>
                        <Thead noWrap>
                            <Tr>
                                <Th sort={getSortParams('Compliance Scan Config Name')}>Name</Th>
                                <Th>Schedule</Th>
                                <Th>Last run</Th>
                                <Th>Clusters</Th>
                                <Th>Profiles</Th>
                                {hasWriteAccessForCompliance && <Td />}
                            </Tr>
                        </Thead>
                        <Tbody>{renderTableBodyContent()}</Tbody>
                    </Table>
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
                                            className="pf-v5-u-mb-sm"
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
