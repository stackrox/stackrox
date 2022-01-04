import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Breadcrumb,
    BreadcrumbItem,
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
import FixabilityLabelsList from 'Components/PatternFly/FixabilityLabelsList';
import SeverityLabelsList from 'Components/PatternFly/SeverityLabelsList';
import NotifierName from 'Containers/VulnMgmt/Reports/Components/NotifierName';
import ScheduleText from 'Containers/VulnMgmt/Reports/Components/ScheduleText';
import ScopeName from 'Containers/VulnMgmt/Reports/Components/ScopeName';
import { vulnManagementReportsPath } from 'routePaths';
import { ReportConfiguration } from 'types/report.proto';

// import ReportOverview from './ReportOverview';

type VulnMgmtReportDetailProps = {
    report: ReportConfiguration;
};

function VulnMgmtReportDetail({ report }: VulnMgmtReportDetailProps): ReactElement {
    const history = useHistory();

    const [isActionsOpen, setIsActionsOpen] = useState(false);

    const { id, name } = report;

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onToggleActions(isOpen) {
        setIsActionsOpen(isOpen);
    }

    function onEditReport() {
        history.replace({
            pathname: `${vulnManagementReportsPath}/${id}`,
            search: 'action=edit',
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
                                dropdownItems={[
                                    <DropdownItem
                                        key="Edit report"
                                        component="button"
                                        onClick={onEditReport}
                                    >
                                        Edit report
                                    </DropdownItem>,
                                ]}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
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
                                <DescriptionListTerm>Resource source</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <ScopeName scopeId={report?.scopeId} />
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
            </PageSection>
        </>
    );
}

export default VulnMgmtReportDetail;
