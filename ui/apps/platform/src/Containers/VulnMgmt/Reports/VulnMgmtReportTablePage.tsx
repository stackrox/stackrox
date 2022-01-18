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
import useTableSort from 'hooks/useTableSort';
import { vulnManagementReportsPath } from 'routePaths';
import { fetchReports, fetchReportsCount, deleteReport, runReport } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';
import VulnMgmtReportTablePanel from './VulnMgmtReportTablePanel';
import VulnMgmtReportTableColumnDescriptor from './VulnMgmtReportTableColumnDescriptor';

function ReportTablePage(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasVulnReportWriteAccess = hasReadWriteAccess('VulnerabilityReports');

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(1);
    const [perPage, setPerPage] = useState(20);
    const [reportCount, setReportCount] = useState(0);

    // To handle sort options.
    const columns = VulnMgmtReportTableColumnDescriptor;
    const defaultSort = {
        field: 'Report Name',
        reversed: false,
    };
    const {
        activeSortIndex,
        setActiveSortIndex,
        activeSortDirection,
        setActiveSortDirection,
        sortOption,
    } = useTableSort(columns, defaultSort);

    const [reports, setReports] = useState<ReportConfiguration[]>([]);

    useEffect(() => {
        fetchReportsCount()
            .then((count) => {
                setReportCount(count);
            })
            .catch((error) => {
                // TODO
            });
    }, []);

    useEffect(() => {
        refreshReportList();
    }, [currentPage, perPage, sortOption]);

    function refreshReportList() {
        fetchReports([], sortOption, currentPage - 1, perPage)
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
                    reportCount={reportCount}
                    currentPage={currentPage}
                    setCurrentPage={setCurrentPage}
                    perPage={perPage}
                    setPerPage={setPerPage}
                    activeSortIndex={activeSortIndex}
                    setActiveSortIndex={setActiveSortIndex}
                    activeSortDirection={activeSortDirection}
                    setActiveSortDirection={setActiveSortDirection}
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
