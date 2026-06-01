import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
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
import { useAdvisoryList } from './useAdvisoryList';
import type { ProtoAdvisoryListItem } from './useAdvisoryList';

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

function AdvisoryListPage() {
    const { data, loading, error } = useAdvisoryList(100, 0);

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
                    <Thead>
                        <Tr>
                            <Th>Advisory ID</Th>
                            <Th>CVE</Th>
                            <Th>Severity</Th>
                            <Th>CVSS</Th>
                            <Th>Source</Th>
                            <Th>Description</Th>
                            <Th>Fix Available</Th>
                            <Th>Images</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {advisories.map((adv) => (
                            <Tr key={adv.advisoryId}>
                                <Td dataLabel="Advisory ID">
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
                                <Td dataLabel="CVE">
                                    <Link
                                        to={`${vulnerabilitiesPrototypeCvePath}/${encodeURIComponent(adv.cveName)}`}
                                    >
                                        {adv.cveName}
                                    </Link>
                                </Td>
                                <Td dataLabel="Severity">
                                    <Label color={severityColor(adv.severity)}>
                                        {severityLabel(adv.severity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="CVSS">
                                    {formatCvss(adv.cvss)}
                                </Td>
                                <Td dataLabel="Source">{adv.sourceName}</Td>
                                <Td dataLabel="Description">
                                    <Truncate
                                        content={adv.description || '-'}
                                        trailingNumChars={0}
                                    />
                                </Td>
                                <Td dataLabel="Fix Available">
                                    {adv.fixedBy ? 'Yes' : 'No'}
                                </Td>
                                <Td dataLabel="Images">{adv.imageCount}</Td>
                            </Tr>
                        ))}
                        {!loading && advisories.length === 0 && (
                            <Tr>
                                <Td colSpan={8}>
                                    <Bullseye>No advisories found</Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
}

export default AdvisoryListPage;
