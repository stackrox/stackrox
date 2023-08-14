import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertProps,
    AlertVariant,
    Breadcrumb,
    BreadcrumbItem,
    Card,
    CardBody,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownToggle,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import DateTimeField from 'Components/DateTimeField';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import FixabilityLabelsList from 'Components/PatternFly/FixabilityLabelsList';
import SeverityLabelsList from 'Components/PatternFly/SeverityLabelsList';
import NotifierName from 'Containers/VulnMgmt/Reports/Components/NotifierName';
import ScheduleText from 'Containers/VulnMgmt/Reports/Components/ScheduleText';
import ScopeName from 'Containers/VulnMgmt/Reports/Components/ScopeName';
import usePermissions from 'hooks/usePermissions';
import { vulnManagementReportsPath } from 'routePaths';
import { deleteReport, runReport } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ReportScope } from 'hooks/useFetchReport';

import { getWriteAccessForReport } from '../VulnMgmtReport.utils';

type VulnMgmtReportDetailProps = {
    report: ReportConfiguration;
    reportScope: ReportScope | null;
};

function VulnMgmtReportDetail({ report, reportScope }: VulnMgmtReportDetailProps): ReactElement {
    const history = useHistory();

    const [alert, setAlert] = useState<AlertProps | null>(null);
    const [deleteConfirmationText, setDeleteConfirmationText] = useState('');
    const [isActionsOpen, setIsActionsOpen] = useState(false);

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForReport = getWriteAccessForReport({ hasReadAccess, hasReadWriteAccess });

    const dropdownItems: ReactElement[] = [];
    if (hasWriteAccessForReport) {
        dropdownItems.push(
            <DropdownItem key="Edit report" component="button" onClick={onEditReport}>
                Edit report
            </DropdownItem>
        );
        dropdownItems.push(
            <DropdownItem key="Run report now" component="button" onClick={onRunReport}>
                Run report now
            </DropdownItem>
        );
        dropdownItems.push(
            <DropdownItem key="Delete report" component="button" onClick={initiateDeleteReport}>
                Delete report
            </DropdownItem>
        );
    }

    const { id, name } = report;

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onToggleActions(isOpen) {
        setIsActionsOpen(isOpen);
    }

    function onEditReport() {
        history.push({
            pathname: `${vulnManagementReportsPath}/${id}`,
            search: 'action=edit',
        });
    }

    function onRunReport() {
        setAlert(null);

        runReport(report.id)
            .then(() => {
                setAlert({
                    variant: AlertVariant.success,
                    title: 'The report has been queued to run.',
                    timeout: 6000,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                setAlert({
                    title: 'Could not run report:',
                    children: message || 'An unknown error occurred while triggering a report run',
                    variant: AlertVariant.danger,
                });
            });
    }

    function initiateDeleteReport() {
        setAlert(null);

        setDeleteConfirmationText(`Are you sure you want to delete the report ${report.name}?`);
    }

    function onConfirmDeleteReport() {
        deleteReport(report.id)
            .then(() => {
                history.replace({
                    pathname: vulnManagementReportsPath,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                setAlert({
                    title: 'Could not delete report',
                    children: message || 'An unknown error occurred while deleting',
                    variant: AlertVariant.danger,
                });
            })
            .finally(() => {
                // close modal on both success or failure
                setDeleteConfirmationText('');
            });
    }

    return (
        <>
            <PageSection id="report-page" variant="light">
                <Breadcrumb className="pf-u-mb-md">
                    <BreadcrumbItemLink to={vulnManagementReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{name}</BreadcrumbItem>
                </Breadcrumb>
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarItem>
                            <Title headingLevel="h1">{name}</Title>
                        </ToolbarItem>
                        {dropdownItems.length > 0 && (
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <Dropdown
                                    onSelect={onSelectActions}
                                    position="right"
                                    toggle={
                                        <DropdownToggle
                                            isPrimary
                                            onToggle={onToggleActions}
                                            toggleIndicator={CaretDownIcon}
                                        >
                                            Actions
                                        </DropdownToggle>
                                    }
                                    isOpen={isActionsOpen}
                                    dropdownItems={dropdownItems}
                                />
                            </ToolbarItem>
                        )}
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <Card>
                    <CardBody>
                        {!!alert && (
                            <Alert
                                isInline
                                variant={alert.variant}
                                title={alert.title}
                                className="pf-u-mb-lg"
                                timeout={alert.timeout}
                                onTimeout={() => setAlert(null)}
                            />
                        )}
                        <DescriptionList
                            columnModifier={{
                                default: '2Col',
                            }}
                        >
                            <DescriptionListGroup>
                                <DescriptionListTerm>Description</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {report.description || <em>No description</em>}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Last run</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <DateTimeField date={report?.runStatus?.lastTimeRun || ''} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>CVE fixability type</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <FixabilityLabelsList
                                        fixability={report?.vulnReportFilters?.fixability}
                                    />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Reporting schedule</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ScheduleText schedule={report?.schedule} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>CVE severities</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <SeverityLabelsList
                                        severities={report?.vulnReportFilters?.severities}
                                    />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Notification method</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <NotifierName notifierId={report?.emailConfig?.notifierId} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Report scope</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ScopeName reportScope={reportScope} />
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Distribution list</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {report?.emailConfig?.mailingLists.join(', ') || (
                                        <em>
                                            No distribution list specified. Default recipient for
                                            notifier will be used.
                                        </em>
                                    )}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </CardBody>
                </Card>
                <ConfirmationModal
                    ariaLabel="Confirm deleting reports"
                    confirmText="Delete"
                    isOpen={!!deleteConfirmationText}
                    onConfirm={onConfirmDeleteReport}
                    onCancel={() => {
                        setDeleteConfirmationText('');
                    }}
                >
                    {deleteConfirmationText}
                </ConfirmationModal>
            </PageSection>
        </>
    );
}

export default VulnMgmtReportDetail;
