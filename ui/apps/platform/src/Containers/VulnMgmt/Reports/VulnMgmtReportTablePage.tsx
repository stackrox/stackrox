/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useEffect, useState, ReactElement } from 'react';
import { Link } from 'react-router-dom';
import {
    Button,
    ButtonVariant,
    PageSection,
    PageSectionVariants,
    Text,
    TextContent,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import ACSEmptyState from 'Components/ACSEmptyState';
import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import { vulnManagementReportsPath } from 'routePaths';
import { fetchReports, deleteReport, runReport } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';
import VulnMgmtReportTablePanel from './VulnMgmtReportTablePanel';
import VulnMgmtReportTableColumnDescriptor from './VulnMgmtReportTableColumnDescriptor';

function ReportTablePage(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasVulnReportWriteAccess = hasReadWriteAccess('VulnerabilityReports');

    const [reports, setReports] = useState<ReportConfiguration[]>([]);
    const columns = VulnMgmtReportTableColumnDescriptor;

    useEffect(() => {
        refreshReportList();
    }, []);

    function refreshReportList() {
        fetchReports()
            .then((reportsResponse) => {
                setReports(reportsResponse);
            })
            .catch(() => {
                // TODO: show error message on failure
            });
    }

    function onRunReports(reportIds) {
        const runPromises = reportIds.map((id) => runReport(id));

        // Note: errors are handled and displayed down at the call site,
        //       ui/apps/platform/src/Containers/VulnMgmt/Reports/VulnMgmtReportTablePage.tsx
        return Promise.all(runPromises).then(() => {
            refreshReportList();
        });
    }

    function onDeleteReports(reportIds) {
        const deletePromises = reportIds.map((id) => deleteReport(id));

        // Note: errors are handled and displayed down at the call site,
        //       ui/apps/platform/src/Containers/VulnMgmt/Reports/VulnMgmtReportTablePage.tsx
        return Promise.all(deletePromises).then(() => {
            refreshReportList();
        });
    }

    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <PageTitle title="Vulnerability Management - Reports" />
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarItem>
                            <TextContent>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                                <Text component="p">
                                    Configure reports, define resource scopes, and assign
                                    distribution lists to report on vulnerabilities across the
                                    organization.
                                </Text>
                            </TextContent>
                        </ToolbarItem>
                        {hasVulnReportWriteAccess && (
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <Button
                                    variant={ButtonVariant.primary}
                                    isInline
                                    component={(props) => (
                                        <Link
                                            {...props}
                                            to={`${vulnManagementReportsPath}?action=create`}
                                        />
                                    )}
                                >
                                    Create report
                                </Button>
                            </ToolbarItem>
                        )}
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            {reports.length > 0 ? (
                <VulnMgmtReportTablePanel
                    reports={reports}
                    reportCount={0}
                    currentPage={0}
                    setCurrentPage={function (page: number): void {
                        throw new Error('Function not implemented.');
                    }}
                    perPage={0}
                    setPerPage={function (perPage: number): void {
                        throw new Error('Function not implemented.');
                    }}
                    activeSortIndex={0}
                    setActiveSortIndex={function (idx: number): void {
                        throw new Error('Function not implemented.');
                    }}
                    activeSortDirection="desc"
                    setActiveSortDirection={function (dir: string): void {
                        throw new Error('Function not implemented.');
                    }}
                    columns={columns}
                    onRunReports={onRunReports}
                    onDeleteReports={onDeleteReports}
                />
            ) : (
                <PageSection variant={PageSectionVariants.light} isFilled>
                    <ACSEmptyState title="No reports are currently configured." />
                </PageSection>
            )}
        </>
    );
}

export default ReportTablePage;
