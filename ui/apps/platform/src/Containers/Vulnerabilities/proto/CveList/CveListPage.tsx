import { useState } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    Pagination,
    PageSection,
    SearchInput,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { vulnerabilitiesPrototypeCvePath } from 'routePaths';

import ProtoNav from '../ProtoNav';
import { usePagination } from '../usePagination';
import { useSort } from '../useSort';
import { useCveList } from './useCveList';
import type { ProtoCVEListItem } from './useCveList';
import {
    CVE_NAME_WIDTH,
    SEVERITY_WIDTH,
    CVSS_SCORE_WIDTH,
    COUNT_WIDTH,
    DATE_WIDTH,
    TABLE_HEADER_STYLE,
    TABLE_CELL_STYLE,
    formatDate,
} from '../utils/tableDefaults';

const severityNames: Record<number, string> = {
    0: 'Unknown',
    1: 'Low',
    2: 'Moderate',
    3: 'Important',
    4: 'Critical',
};

function severityColor(severity: number): 'red' | 'orange' | 'blue' | 'grey' {
    switch (severity) {
        case 4:
            return 'red';
        case 3:
            return 'orange';
        case 2:
            return 'blue';
        default:
            return 'grey';
    }
}

function severityLabel(severity: number): string {
    return severityNames[severity] ?? 'Unknown';
}

function formatCvss(cvss: number): string {
    return cvss ? cvss.toFixed(1) : '-';
}

// Column keys must match backend sortBy values. Non-sortable columns use empty string.
const cveSortColumns = ['cveName', 'severity', 'cvss', 'imageCount', '', 'firstSeen', '', ''];

function CveListPage() {
    const [cveFilter, setCveFilter] = useState('');
    const { sortBy, sortDir, getThSortProps } = useSort(cveSortColumns, 1);
    const { page, perPage, offset, onSetPage, onPerPageSelect } = usePagination(20);
    const { data, loading, error } = useCveList(perPage, offset, sortBy, sortDir, cveFilter);

    const cves: ProtoCVEListItem[] = data?.cves ?? [];
    const totalCount = data?.totalCount ?? 0;

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Vuln Management V5</Title>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <ProtoNav />
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            <SearchInput
                                placeholder="Filter by CVE (e.g. CVE-2026-0968)"
                                value={cveFilter}
                                onChange={(_event, value) => setCveFilter(value)}
                                onClear={() => setCveFilter('')}
                                style={{ minWidth: '320px' }}
                            />
                        </ToolbarItem>
                        <ToolbarItem>
                            {loading && <Spinner size="md" />}
                            {!loading &&
                                `${cves.length} of ${totalCount} CVEs`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading CVEs: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Vuln Management V5 CVE list" variant="compact">
                    <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                        <Tr>
                            <Th {...getThSortProps(0)} style={TABLE_HEADER_STYLE}>CVE</Th>
                            <Th {...getThSortProps(1)} style={TABLE_HEADER_STYLE} info={{ tooltip: 'Highest severity reported by any advisory for this CVE' }}>Top Severity</Th>
                            <Th {...getThSortProps(2)} style={TABLE_HEADER_STYLE} info={{ tooltip: 'Highest CVSS score reported by any advisory for this CVE' }}>Top CVSS</Th>
                            <Th {...getThSortProps(3)} style={TABLE_HEADER_STYLE} info={{ tooltip: 'Number of distinct images affected by this CVE' }}>Images</Th>
                            <Th style={TABLE_HEADER_STYLE} info={{ tooltip: 'Whether a fix is available from any advisory source' }}>Fixable</Th>
                            <Th {...getThSortProps(5)} style={TABLE_HEADER_STYLE}>First Seen</Th>
                            <Th style={TABLE_HEADER_STYLE}>Published</Th>
                            <Th style={TABLE_HEADER_STYLE} info={{ tooltip: 'EPSS: Exploit Prediction Scoring System probability' }}>EPSS</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {cves.map((cve) => (
                            <Tr key={cve.cveName}>
                                <Td dataLabel="CVE" style={{ ...TABLE_CELL_STYLE, width: `${CVE_NAME_WIDTH}px` }}>
                                    <Link
                                        to={`${vulnerabilitiesPrototypeCvePath}/${encodeURIComponent(cve.cveName)}`}
                                    >
                                        {cve.cveName}
                                    </Link>
                                </Td>
                                <Td dataLabel="Severity" style={{ ...TABLE_CELL_STYLE, width: `${SEVERITY_WIDTH}px` }}>
                                    <Label color={severityColor(cve.severity)}>
                                        {severityLabel(cve.severity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="CVSS" style={{ ...TABLE_CELL_STYLE, width: `${CVSS_SCORE_WIDTH}px` }}>
                                    {formatCvss(cve.cvss)}
                                </Td>
                                <Td dataLabel="Images" style={{ ...TABLE_CELL_STYLE, width: `${COUNT_WIDTH}px` }}>{cve.imageCount}</Td>
                                <Td dataLabel="Fixable" style={{ ...TABLE_CELL_STYLE, width: `${COUNT_WIDTH}px` }}>
                                    {cve.fixable ? 'Yes' : 'No'}
                                </Td>
                                <Td dataLabel="First Seen" style={{ ...TABLE_CELL_STYLE, width: `${DATE_WIDTH}px` }}>
                                    {formatDate(cve.firstSeen)}
                                </Td>
                                <Td dataLabel="Published" style={{ ...TABLE_CELL_STYLE, width: `${DATE_WIDTH}px` }}>
                                    {formatDate(cve.publishedDate ?? null)}
                                </Td>
                                <Td dataLabel="EPSS" style={{ ...TABLE_CELL_STYLE, width: `${COUNT_WIDTH}px` }}>
                                    {cve.epssProbability != null
                                        ? `${(cve.epssProbability * 100).toFixed(1)}%`
                                        : '-'}
                                </Td>
                            </Tr>
                        ))}
                        {!loading && cves.length === 0 && (
                            <Tr>
                                <Td colSpan={8}>
                                    <Bullseye>No CVEs found</Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
                <Pagination
                    itemCount={totalCount}
                    perPage={perPage}
                    page={page}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                />
            </PageSection>
        </>
    );
}

export default CveListPage;
