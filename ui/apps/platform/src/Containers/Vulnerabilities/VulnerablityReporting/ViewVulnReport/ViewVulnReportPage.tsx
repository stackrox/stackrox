import React, { ReactElement, useState } from 'react';
import { Link, useHistory, useParams, generatePath } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    AlertVariant,
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Spinner,
    Dropdown,
    DropdownToggle,
    DropdownItem,
    DropdownSeparator,
    Tabs,
    Tab,
    TabTitleText,
    TabTitleIcon,
    TabAction,
    Popover,
} from '@patternfly/react-core';
import { CaretDownIcon, ClipboardCheckIcon, HelpIcon, HomeIcon } from '@patternfly/react-icons';

import { vulnerabilityReportPath } from 'Containers/Vulnerabilities/VulnerablityReporting/pathsForVulnerabilityReporting';
import { systemConfigPath, vulnerabilityReportsPath } from 'routePaths';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import useFetchReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReport';
import useDeleteModal, {
    isErrorDeleteResult,
} from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteModal';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import usePermissions from 'hooks/usePermissions';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import DeleteModal from '../components/DeleteModal';
import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import ReportJobs from './ReportJobs';
import useRunReport from '../api/useRunReport';
import { useWatchLastSnapshotForReports } from '../api/useWatchLastSnapshotForReports';

export type TabTitleProps = {
    icon?: ReactElement;
    children: string;
};

function TabTitle({ icon, children }: TabTitleProps): ReactElement {
    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {icon && (
                <FlexItem>
                    <TabTitleIcon>{icon}</TabTitleIcon>
                </FlexItem>
            )}
            <FlexItem>
                <TabTitleText>{children}</TabTitleText>
            </FlexItem>
        </Flex>
    );
}

function ViewVulnReportPage() {
    const history = useHistory();
    const { reportId } = useParams();
    const [isActionsDropdownOpen, setIsActionsDropdownOpen] = useState(false);

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Image') && // for vulnerabilities
        hasReadAccess('Integration'); // for notifiers

    const { reportConfiguration, isLoading, error: fetchError } = useFetchReport(reportId);
    const { reportSnapshots } = useWatchLastSnapshotForReports(reportConfiguration);
    const reportSnapshot = reportSnapshots[reportId];

    const {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteResults,
    } = useDeleteModal({
        onCompleted: () => {
            history.push(vulnerabilityReportsPath);
        },
    });

    const { toasts, addToast, removeToast } = useToasts();

    const { isRunning, runError, runReport } = useRunReport({
        onCompleted: ({ reportNotificationMethod }) => {
            if (reportNotificationMethod === 'EMAIL') {
                addToast('The report has been sent to the configured email notifier', 'success');
            } else if (reportNotificationMethod === 'DOWNLOAD') {
                addToast(
                    'The report generation has started and will be available for download once complete',
                    'success'
                );
            }
        },
    });

    function onToggleActionsDropdown() {
        setIsActionsDropdownOpen((prevValue) => !prevValue);
    }

    function onSelectAction() {
        setIsActionsDropdownOpen(false);
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (fetchError || !reportConfiguration) {
        return (
            <NotFoundMessage
                title="Error fetching the report configuration"
                message={fetchError || 'No data available'}
                actionText="Go to reports"
                url={vulnerabilityReportsPath}
            />
        );
    }

    const vulnReportPageURL = generatePath(vulnerabilityReportPath, {
        reportId: reportConfiguration.id,
    }) as string;

    const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);

    const isReportStatusPending =
        reportSnapshot?.reportStatus.runState === 'PREPARING' ||
        reportSnapshot?.reportStatus.runState === 'WAITING';

    return (
        <>
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        timeout
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
            {runError && <Alert variant={AlertVariant.danger} isInline title={runError} />}
            <PageTitle title="View vulnerability report" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{reportConfiguration.name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'row' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">{reportConfiguration.name}</Title>
                    </FlexItem>
                    {hasWriteAccessForReport && (
                        <FlexItem>
                            <Dropdown
                                onSelect={onSelectAction}
                                position="right"
                                toggle={
                                    <DropdownToggle
                                        onToggle={onToggleActionsDropdown}
                                        toggleIndicator={CaretDownIcon}
                                    >
                                        Actions
                                    </DropdownToggle>
                                }
                                isOpen={isActionsDropdownOpen}
                                dropdownItems={[
                                    <DropdownItem
                                        key="Edit report"
                                        component="button"
                                        onClick={() => {
                                            history.push(`${vulnReportPageURL}?action=edit`);
                                        }}
                                        isDisabled={isReportStatusPending || isRunning}
                                    >
                                        Edit report
                                    </DropdownItem>,
                                    <DropdownSeparator key="separator" />,
                                    <DropdownItem
                                        key="Send report now"
                                        component="button"
                                        onClick={() => runReport(reportId, 'EMAIL')}
                                        isDisabled={
                                            isReportStatusPending ||
                                            isRunning ||
                                            reportConfiguration.notifiers.length === 0
                                        }
                                        description={
                                            reportConfiguration.notifiers.length === 0
                                                ? 'No delivery destinations set'
                                                : ''
                                        }
                                    >
                                        Send report now
                                    </DropdownItem>,
                                    <DropdownItem
                                        key="Generate download"
                                        component="button"
                                        onClick={() => runReport(reportId, 'DOWNLOAD')}
                                        isDisabled={isReportStatusPending || isRunning}
                                    >
                                        Generate download
                                    </DropdownItem>,
                                    <DropdownItem
                                        key="Clone report"
                                        component="button"
                                        onClick={() => {
                                            history.push(`${vulnReportPageURL}?action=clone`);
                                        }}
                                    >
                                        Clone report
                                    </DropdownItem>,
                                    <DropdownSeparator key="Separator" />,
                                    <DropdownItem
                                        key="Delete report"
                                        className="pf-u-danger-color-100"
                                        component="button"
                                        onClick={() => {
                                            openDeleteModal([reportConfiguration.id]);
                                        }}
                                        isDisabled={isReportStatusPending || isRunning}
                                    >
                                        Delete report
                                    </DropdownItem>,
                                ]}
                            />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <Tabs
                    className="pf-u-background-color-100"
                    defaultActiveKey={0}
                    aria-label="Report details tabs"
                    role="region"
                >
                    <Tab
                        eventKey={0}
                        title={<TabTitle icon={<HomeIcon />}>Configuration details</TabTitle>}
                        aria-label="Configuration details tab"
                    >
                        <PageSection
                            variant="light"
                            padding={{ default: 'noPadding' }}
                            className="pf-u-py-lg pf-u-px-lg"
                        >
                            <ReportParametersDetails formValues={reportFormValues} />
                            <Divider component="div" className="pf-u-py-md" />
                            <DeliveryDestinationsDetails formValues={reportFormValues} />
                            <Divider component="div" className="pf-u-py-md" />
                            <ScheduleDetails formValues={reportFormValues} />
                        </PageSection>
                    </Tab>
                    <Tab
                        eventKey={1}
                        title={<TabTitle icon={<ClipboardCheckIcon />}>All report jobs</TabTitle>}
                        aria-label="Report jobs tab"
                        actions={
                            <>
                                <Popover
                                    aria-label="All report jobs help text"
                                    headerContent={<div>All report jobs</div>}
                                    bodyContent={
                                        <div>
                                            This function displays the requested jobs from different
                                            users and includes their statuses accordingly. While the
                                            function provides the ability to monitor and audit your
                                            active and past requested jobs, we suggest configuring
                                            the{' '}
                                            <Link to={systemConfigPath}>
                                                Vulnerability report retention limit
                                            </Link>{' '}
                                            based on your needs in order to ensure optimal user
                                            experience. All the report jobs will be kept in your
                                            system until they exceed the limit set by you.
                                        </div>
                                    }
                                    enableFlip
                                    position="top"
                                >
                                    <TabAction aria-label="Help for report jobs tab">
                                        <HelpIcon />
                                    </TabAction>
                                </Popover>
                            </>
                        }
                    >
                        <PageSection
                            padding={{ default: 'noPadding' }}
                            className="pf-u-py-lg pf-u-px-lg"
                        >
                            <ReportJobs reportId={reportId} />
                        </PageSection>
                    </Tab>
                </Tabs>
            </PageSection>
            <DeleteModal
                title="Permanently delete report?"
                isOpen={isDeleteModalOpen}
                onClose={closeDeleteModal}
                isDeleting={isDeleting}
                onDelete={onDelete}
            >
                <AlertGroup>
                    {deleteResults?.filter(isErrorDeleteResult).map((deleteResult) => {
                        return (
                            <Alert
                                isInline
                                variant={AlertVariant.danger}
                                title={`Failed to delete "${reportConfiguration.name}"`}
                                className="pf-u-mb-sm"
                            >
                                {deleteResult.error}
                            </Alert>
                        );
                    })}
                </AlertGroup>
                <p>
                    The selected report and any attached downloadable reports will be permanently
                    deleted. The action cannot be undone.
                </p>
            </DeleteModal>
        </>
    );
}

export default ViewVulnReportPage;
