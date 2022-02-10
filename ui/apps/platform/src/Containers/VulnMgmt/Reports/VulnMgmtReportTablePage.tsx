/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertVariant,
    Bullseye,
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

import ACSEmptyState from 'Components/ACSEmptyState';
import PageTitle from 'Components/PageTitle';
import ButtonLink from 'Components/PatternFly/ButtonLink';
import { searchCategories } from 'constants/entityTypes';
import usePermissions from 'hooks/usePermissions';
import useSearchOptions from 'hooks/useSearchOptions';
import useFetchReports from 'hooks/useFetchReports';
import useTableSort from 'hooks/useTableSort';
import { vulnManagementReportsPath } from 'routePaths';
import { deleteReport, runReport } from 'services/ReportsService';
import { SearchFilter } from 'types/search';
import { filterAllowedSearch } from 'utils/searchUtils';
import { getQueryString } from 'utils/queryStringUtils';
import VulnMgmtReportTablePanel from './VulnMgmtReportTablePanel';
import VulnMgmtReportTableColumnDescriptor from './VulnMgmtReportTableColumnDescriptor';
import { VulnMgmtReportQueryObject } from './VulnMgmtReport.utils';

type ReportTablePageProps = {
    query: VulnMgmtReportQueryObject;
};

function ReportTablePage({ query }: ReportTablePageProps): ReactElement {
    const history = useHistory();

    const { hasReadWriteAccess } = usePermissions();
    const hasVulnReportWriteAccess = hasReadWriteAccess('VulnerabilityReports');
    const hasAccessScopeWriteAccess = hasReadWriteAccess('AuthProvider');
    const hasNotifierIntegrationWriteAccess = hasReadWriteAccess('Notifier');
    const canWriteReports =
        hasVulnReportWriteAccess && hasAccessScopeWriteAccess && hasNotifierIntegrationWriteAccess;

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

    function setSearchFilter(filters: SearchFilter) {
        const newSearchString = getQueryString({
            s: filters,
        });

        history.push({
            search: newSearchString,
        });
    }

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
        return Promise.all(runPromises);
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
                        {canWriteReports && (
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <ButtonLink
                                    variant={ButtonVariant.primary}
                                    isInline
                                    to={`${vulnManagementReportsPath}?action=create`}
                                >
                                    Create report
                                </ButtonLink>
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
                    searchFilter={filteredSearch}
                    setSearchFilter={setSearchFilter}
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
