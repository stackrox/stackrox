import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    Pagination,
    PageSection,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Truncate,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

import { vulnerabilitiesPrototypeCvePath } from 'routePaths';

import ProtoNav from '../ProtoNav';
import { usePagination } from '../usePagination';
import { useSort } from '../useSort';
import { useAdvisoryList } from './useAdvisoryList';
import type { ProtoAdvisoryListItem } from './useAdvisoryList';
import {
    SEVERITY_WIDTH,
    COUNT_WIDTH,
    TABLE_HEADER_STYLE,
    TABLE_CELL_STYLE,
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

// Column keys: advisoryId, cve (not sortable), severity, cvss, source (not sortable),
// description (not sortable), fixAvail (not sortable), components (not sortable), imageCount
const advSortColumns = ['advisoryId', '', 'severity', 'cvss', '', '', '', '', 'imageCount'];

function AdvisoryListPage() {
    const { sortBy, sortDir, getThSortProps } = useSort(advSortColumns, 2);
    const { page, perPage, offset, onSetPage, onPerPageSelect } = usePagination(20);
    const { data, loading, error } = useAdvisoryList(perPage, offset, sortBy, sortDir);

    const advisories: ProtoAdvisoryListItem[] = data?.advisories ?? [];
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
                            {loading && <Spinner size="md" />}
                            {!loading &&
                                `${advisories.length} of ${totalCount} Advisories`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading advisories: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Vuln Management V5 advisory list" variant="compact">
                    <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                        <Tr>
                            <Th style={TABLE_HEADER_STYLE} {...getThSortProps(0)}>Advisory ID</Th>
                            <Th style={TABLE_HEADER_STYLE}>CVE</Th>
                            <Th style={TABLE_HEADER_STYLE} {...getThSortProps(2)}>Severity</Th>
                            <Th style={TABLE_HEADER_STYLE} {...getThSortProps(3)}>CVSS</Th>
                            <Th style={TABLE_HEADER_STYLE}>Source</Th>
                            <Th style={TABLE_HEADER_STYLE}>Description</Th>
                            <Th style={TABLE_HEADER_STYLE}>Fix Available</Th>
                            <Th style={TABLE_HEADER_STYLE}>Components</Th>
                            <Th style={TABLE_HEADER_STYLE} {...getThSortProps(8)}>Images</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {advisories.map((adv) => (
                            <Tr key={adv.advisoryId}>
                                <Td style={TABLE_CELL_STYLE} dataLabel="Advisory ID">
                                    {adv.link ? (
                                        <a
                                            href={adv.link}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            {adv.advisoryId}{' '}
                                            <ExternalLinkAltIcon />
                                        </a>
                                    ) : (
                                        adv.advisoryId
                                    )}
                                </Td>
                                <Td style={TABLE_CELL_STYLE} dataLabel="CVE">
                                    <Link
                                        to={`${vulnerabilitiesPrototypeCvePath}/${encodeURIComponent(adv.cveName)}`}
                                    >
                                        {adv.cveName}
                                    </Link>
                                </Td>
                                <Td style={{ ...TABLE_CELL_STYLE, width: `${SEVERITY_WIDTH}px` }} dataLabel="Severity">
                                    <Label color={severityColor(adv.severity)}>
                                        {severityLabel(adv.severity)}
                                    </Label>
                                </Td>
                                <Td style={TABLE_CELL_STYLE} dataLabel="CVSS">
                                    {formatCvss(adv.cvss)}
                                </Td>
                                <Td style={TABLE_CELL_STYLE} dataLabel="Source">{adv.sourceName}</Td>
                                <Td style={TABLE_CELL_STYLE} dataLabel="Description">
                                    <Truncate
                                        content={adv.description || '-'}
                                        trailingNumChars={0}
                                    />
                                </Td>
                                <Td style={TABLE_CELL_STYLE} dataLabel="Fix Available">
                                    {adv.fixedBy ? 'Yes' : 'No'}
                                </Td>
                                <Td style={{ ...TABLE_CELL_STYLE, width: `${COUNT_WIDTH}px` }} dataLabel="Components">
                                    {adv.componentCount}
                                </Td>
                                <Td style={{ ...TABLE_CELL_STYLE, width: `${COUNT_WIDTH}px` }} dataLabel="Images">{adv.imageCount}</Td>
                            </Tr>
                        ))}
                        {!loading && advisories.length === 0 && (
                            <Tr>
                                <Td colSpan={9}>
                                    <Bullseye>No advisories found</Bullseye>
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

export default AdvisoryListPage;
