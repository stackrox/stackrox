import React, { ReactElement, useState } from 'react';
import { useNavigate, useParams, generatePath } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Spinner,
    Tabs,
    Tab,
    TabTitleText,
    Card,
    CardBody,
} from '@patternfly/react-core';
import {
    Dropdown,
    DropdownToggle,
    DropdownItem,
    DropdownSeparator,
} from '@patternfly/react-core/deprecated';
import { CaretDownIcon } from '@patternfly/react-icons';

import { vulnerabilityConfigurationReportDetailsPath } from 'Containers/Vulnerabilities/VulnerablityReporting/pathsForVulnerabilityReporting';
import { vulnerabilityConfigurationReportsPath } from 'routePaths';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import useFetchReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReport';
import useDeleteModal, {
    isErrorDeleteResult,
} from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteModal';

import { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';
import DeleteModal from 'Components/PatternFly/DeleteModal';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import usePermissions from 'hooks/usePermissions';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';

import ReportJobsHelpAction from 'Components/ReportJob/ReportJobsHelpAction';
import { JobContextTab } from 'Components/ReportJob/types';
import { ensureJobContextTab } from 'Components/ReportJob/utils';
import EmailTemplatePreview from '../components/EmailTemplatePreview';
import ReportParametersDetails from '../components/ReportParametersDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import { defaultEmailBody, getDefaultEmailSubject } from '../forms/emailTemplateFormUtils';
import ReportJobs from './ReportJobs';
import useRunReport from '../api/useRunReport';
import { useWatchLastSnapshotForReports } from '../api/useWatchLastSnapshotForReports';

export type TabTitleProps = {
    icon?: ReactElement;
    children: string;
};

const configDetailsTabId = 'VulnReportsConfigDetails';
const allReportJobsTabId = 'VulnReportsConfigReportJobs';

const headingLevel = 'h2';

function ViewVulnReportPage() {
    const navigate = useNavigate();
    const { reportId } = useParams() as { reportId: string };
    const [isActionsDropdownOpen, setIsActionsDropdownOpen] = useState(false);
    const [selectedTab, setSelectedTab] = useState<JobContextTab>('CONFIGURATION_DETAILS');

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
            navigate(vulnerabilityConfigurationReportsPath);
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
                <Spinner />
            </Bullseye>
        );
    }

    if (fetchError || !reportConfiguration) {
        return (
            <NotFoundMessage
                title="Error fetching the report configuration"
                message={fetchError || 'No data available'}
                actionText="Go to reports"
                url={vulnerabilityConfigurationReportsPath}
            />
        );
    }

    const vulnReportPageURL = generatePath(vulnerabilityConfigurationReportDetailsPath, {
        reportId: reportConfiguration.id,
    });

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
                        component="p"
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
            {runError && <Alert variant="danger" isInline title={runError} component="p" />}
            <PageTitle title="View vulnerability report" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityConfigurationReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{reportConfiguration.name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
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
                                            navigate(`${vulnReportPageURL}?action=edit`);
                                        }}
                                        isDisabled={isReportStatusPending || isRunning}
                                    >
                                        Edit report
                                    </DropdownItem>,
                                    <DropdownSeparator key="separator" />,
                                    <DropdownItem
                                        key="Send report"
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
                                        Send report
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
                                            navigate(`${vulnReportPageURL}?action=clone`);
                                        }}
                                    >
                                        Clone report
                                    </DropdownItem>,
                                    <DropdownSeparator key="Separator" />,
                                    <DropdownItem
                                        key="Delete report"
                                        className="pf-v5-u-danger-color-100"
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
            <PageSection variant="light" className="pf-v5-u-py-0">
                <Tabs
                    activeKey={selectedTab}
                    onSelect={(_e, tab) => {
                        setSelectedTab(ensureJobContextTab(tab));
                    }}
                    aria-label="Report details tabs"
                >
                    <Tab
                        tabContentId={configDetailsTabId}
                        eventKey="CONFIGURATION_DETAILS"
                        title={<TabTitleText>Configuration details</TabTitleText>}
                    />
                    <Tab
                        tabContentId={allReportJobsTabId}
                        eventKey="ALL_REPORT_JOBS"
                        title={<TabTitleText>All report jobs</TabTitleText>}
                        actions={<ReportJobsHelpAction reportType="Vulnerability" />}
                    />
                </Tabs>
            </PageSection>
            {selectedTab === 'CONFIGURATION_DETAILS' && (
                <PageSection isCenterAligned id={configDetailsTabId}>
                    <Card>
                        <CardBody>
                            <ReportParametersDetails
                                headingLevel={headingLevel}
                                formValues={reportFormValues}
                            />
                            <Divider component="div" className="pf-v5-u-py-md" />
                            <NotifierConfigurationView
                                headingLevel={headingLevel}
                                customBodyDefault={defaultEmailBody}
                                customSubjectDefault={getDefaultEmailSubject(
                                    reportFormValues.reportParameters.reportName,
                                    reportFormValues.reportParameters.reportScope?.name
                                )}
                                notifierConfigurations={reportFormValues.deliveryDestinations}
                                renderTemplatePreview={({
                                    customBody,
                                    customSubject,
                                    customSubjectDefault,
                                }: TemplatePreviewArgs) => (
                                    <EmailTemplatePreview
                                        emailSubject={customSubject}
                                        emailBody={customBody}
                                        defaultEmailSubject={customSubjectDefault}
                                        reportParameters={reportFormValues.reportParameters}
                                    />
                                )}
                            />
                            <Divider component="div" className="pf-v5-u-py-md" />
                            <ScheduleDetails formValues={reportFormValues} />
                        </CardBody>
                    </Card>
                </PageSection>
            )}
            {selectedTab === 'ALL_REPORT_JOBS' && (
                <PageSection isCenterAligned id={allReportJobsTabId}>
                    <ReportJobs reportId={reportId} />
                </PageSection>
            )}
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
                                variant="danger"
                                title={`Failed to delete "${reportConfiguration.name}"`}
                                component="p"
                                className="pf-v5-u-mb-sm"
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
