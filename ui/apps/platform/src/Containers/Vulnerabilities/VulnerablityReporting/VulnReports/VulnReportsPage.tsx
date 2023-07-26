import React, { useState } from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Button,
    Card,
    CardBody,
    Bullseye,
    Spinner,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateVariant,
    Text,
    Modal,
    Alert,
    AlertVariant,
} from '@patternfly/react-core';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link } from 'react-router-dom';
import { ExclamationCircleIcon, FileIcon } from '@patternfly/react-icons';

import useFetchReports from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReports';
import usePermissions from 'hooks/usePermissions';
import { vulnerabilityReportsPath } from 'routePaths';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { deleteReportConfiguration } from 'services/ReportsService';
import useModal from 'hooks/useModal';
import HelpIconTh from './HelpIconTh';
import LastRunStatusState from './LastRunStatusState';
import LastRunState from './LastRunState';
import { getErrorMessage } from '../errorUtils';

const CreateReportsButton = () => {
    return (
        <Link to={`${vulnerabilityReportsPath}?action=create`}>
            <Button variant="primary">Create report</Button>
        </Link>
    );
};

function VulnReportsPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWorkflowAdministrationWriteAccess = hasReadWriteAccess('WorkflowAdministration');
    const hasImageReadAccess = hasReadAccess('Image');
    const hasAccessScopeReadAccess = hasReadAccess('Access');
    const hasNotifierIntegrationReadAccess = hasReadAccess('Integration');
    const canCreateReports =
        hasWorkflowAdministrationWriteAccess &&
        hasImageReadAccess &&
        hasAccessScopeReadAccess &&
        hasNotifierIntegrationReadAccess;

    const { reports, isLoading, error: fetchError, fetchReports } = useFetchReports();

    const { isModalOpen, openModal, closeModal } = useModal();
    const [reportIdToDelete, setReportIdToDelete] = useState<string>('');
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState<string>('');

    function openDeleteModal(reportId: string) {
        openModal();
        setReportIdToDelete(reportId);
    }

    function closeDeleteModal() {
        closeModal();
        setReportIdToDelete('');
    }

    async function deleteReport(reportId: string) {
        setIsDeleting(true);
        try {
            await deleteReportConfiguration(reportId);
            setIsDeleting(false);
            closeDeleteModal();
            fetchReports();
        } catch (err) {
            setIsDeleting(false);
            setDeleteError(getErrorMessage(err));
        }
    }

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-py-lg pf-u-px-lg"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign delivery
                                destinations to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    {reports.length > 0 && canCreateReports && (
                        <FlexItem>
                            <CreateReportsButton />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody className="pf-u-p-0">
                            {isLoading && (
                                <div className="pf-u-p-md">
                                    <Bullseye>
                                        <Spinner isSVG />
                                    </Bullseye>
                                </div>
                            )}
                            {fetchError && (
                                <EmptyState variant={EmptyStateVariant.small}>
                                    <EmptyStateIcon
                                        icon={ExclamationCircleIcon}
                                        className="pf-u-danger-color-100"
                                    />
                                    <Title headingLevel="h2" size="lg">
                                        Unable to get vulnerability reports
                                    </Title>
                                    <EmptyStateBody>{fetchError}</EmptyStateBody>
                                </EmptyState>
                            )}
                            {!isLoading && !fetchError && (
                                <TableComposable borders={false}>
                                    <Thead noWrap>
                                        <Tr>
                                            <Th>Report</Th>
                                            <HelpIconTh tooltip="A set of user-configured rules for selecting deployments as part of the report scope">
                                                Collection
                                            </HelpIconTh>
                                            <Th>Last run status</Th>
                                            <HelpIconTh tooltip="The report that was last run by a schedule or an on-demand action including 'send report now' and 'generate a downloadable report'">
                                                Last run
                                            </HelpIconTh>
                                            <Td />
                                        </Tr>
                                    </Thead>
                                    {reports.length === 0 && (
                                        <Tbody>
                                            <Tr>
                                                <Td colSpan={4}>
                                                    <Bullseye>
                                                        <EmptyStateTemplate
                                                            title="No vulnerability reports yet"
                                                            headingLevel="h2"
                                                            icon={FileIcon}
                                                        >
                                                            {canCreateReports && (
                                                                <Flex
                                                                    direction={{
                                                                        default: 'column',
                                                                    }}
                                                                >
                                                                    <FlexItem>
                                                                        <Text>
                                                                            To get started, create a
                                                                            report
                                                                        </Text>
                                                                    </FlexItem>
                                                                    <FlexItem>
                                                                        <CreateReportsButton />
                                                                    </FlexItem>
                                                                </Flex>
                                                            )}
                                                        </EmptyStateTemplate>
                                                    </Bullseye>
                                                </Td>
                                            </Tr>
                                        </Tbody>
                                    )}
                                    {reports.map((report) => {
                                        const rowActions = [
                                            {
                                                title: 'Edit report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                },
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: 'Send report now',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                },
                                            },
                                            {
                                                title: 'Generate download',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                },
                                            },
                                            {
                                                title: 'Clone report',
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                },
                                            },
                                            {
                                                isSeparator: true,
                                            },
                                            {
                                                title: (
                                                    <span className="pf-u-danger-color-100">
                                                        Delete report
                                                    </span>
                                                ),
                                                onClick: (event) => {
                                                    event.preventDefault();
                                                    openDeleteModal(report.id);
                                                },
                                            },
                                        ];
                                        return (
                                            <Tbody
                                                key={report.id}
                                                style={{
                                                    borderBottom:
                                                        '1px solid var(--pf-c-table--BorderColor)',
                                                }}
                                            >
                                                <Tr>
                                                    <Td>{report.name}</Td>
                                                    <Td>
                                                        {
                                                            report.resourceScope.collectionScope
                                                                .collectionName
                                                        }
                                                    </Td>
                                                    <Td>
                                                        <LastRunStatusState
                                                            reportStatus={
                                                                report.reportLastRunStatus
                                                            }
                                                        />
                                                    </Td>
                                                    <Td>
                                                        <LastRunState
                                                            reportStatus={report.reportStatus}
                                                        />
                                                    </Td>
                                                    <Td isActionCell>
                                                        <ActionsColumn items={rowActions} />
                                                    </Td>
                                                </Tr>
                                            </Tbody>
                                        );
                                    })}
                                </TableComposable>
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
            {reportIdToDelete !== '' && (
                <Modal
                    variant="small"
                    title="Permanently delete report?"
                    isOpen={isModalOpen}
                    onClose={closeDeleteModal}
                    actions={[
                        <Button
                            key="confirm"
                            variant="danger"
                            isLoading={isDeleting}
                            isDisabled={isDeleting}
                            onClick={() => deleteReport(reportIdToDelete)}
                        >
                            Delete
                        </Button>,
                        <Button key="cancel" variant="secondary" onClick={closeDeleteModal}>
                            Cancel
                        </Button>,
                    ]}
                >
                    {deleteError && (
                        <Alert
                            isInline
                            variant={AlertVariant.danger}
                            title={deleteError}
                            className="pf-u-mb-sm"
                        />
                    )}
                    <p>
                        This report and any attached downloadable reports will be permanently
                        deleted. The action cannot be undone.
                    </p>
                </Modal>
            )}
        </>
    );
}

export default VulnReportsPage;
