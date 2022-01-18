/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useEffect, useState, ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@apollo/client';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Button,
    ButtonVariant,
    PageSection,
    PageSectionVariants,
    Spinner,
    Text,
    TextContent,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import useDeepCompareEffect from 'use-deep-compare-effect';

import ACSEmptyState from 'Components/ACSEmptyState';
import PageTitle from 'Components/PageTitle';
import { searchCategories } from 'constants/entityTypes';
import usePermissions from 'hooks/usePermissions';
import useSearchOptions from 'hooks/useSearchOptions';
import useFetchReports from 'hooks/useFetchReports';
import useTableSort from 'hooks/useTableSort';
import { vulnManagementReportsPath } from 'routePaths';
import { deleteReport, runReport } from 'services/ReportsService';
import { filterAllowedSearch } from 'utils/searchUtils';
import VulnMgmtReportTablePanel from './VulnMgmtReportTablePanel';
import VulnMgmtReportTableColumnDescriptor from './VulnMgmtReportTableColumnDescriptor';
import { VulnMgmtReportQueryObject } from './VulnMgmtReport.utils';

type ReportTablePageProps = {
    query: VulnMgmtReportQueryObject;
};

function ReportTablePage({ query }: ReportTablePageProps): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasVulnReportWriteAccess = hasReadWriteAccess('VulnerabilityReports');

    const searchOptions = useSearchOptions(searchCategories.REPORT_CONFIGURATIONS) || [];

    const pageSearch = query.s;
    const filteredSearch = filterAllowedSearch(searchOptions, pageSearch || {});

    // Handle changes in the current table page.
    const [currentPage, setCurrentPage] = useState(1);
    const [perPage, setPerPage] = useState(20);

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

    const { reports, reportCount, error, isLoading, triggerRefresh } = useFetchReports(
        filteredSearch,
        sortOption,
        currentPage,
        perPage
    );

    function onRunReports(reportIds) {
        const runPromises = reportIds.map((id) => runReport(id));

        // Note: errors are handled and displayed down at the call site,
        //       ui/apps/platform/src/Containers/VulnMgmt/Reports/VulnMgmtReportTablePage.tsx
        return Promise.all(runPromises).then(() => {
            triggerRefresh();
        });
    }

    function onDeleteReports(reportIds) {
        const deletePromises = reportIds.map((id) => deleteReport(id));

        // Note: errors are handled and displayed down at the call site,
        //       ui/apps/platform/src/Containers/VulnMgmt/Reports/VulnMgmtReportTablePage.tsx
        return Promise.all(deletePromises).then(() => {
            triggerRefresh();
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
            {!!error && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={error}
                    className="pf-u-mb-lg"
                />
            )}
            {isLoading && (
                <PageSection variant={PageSectionVariants.light} isFilled>
                    <Bullseye>
                        <Spinner isSVG size="lg" />
                    </Bullseye>
                </PageSection>
            )}
            {!isLoading && reports && reports?.length > 0 && (
                <VulnMgmtReportTablePanel
                    reports={reports || []}
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
            )}
            {!isLoading && !reports?.length && (
                <PageSection variant={PageSectionVariants.light} isFilled>
                    <ACSEmptyState title="No reports are currently configured." />
                </PageSection>
            )}
        </>
    );
}

export default ReportTablePage;
