import React, { useState, useMemo } from 'react';
import {
    Text,
    EmptyState,
    EmptyStateHeader,
    EmptyStateIcon,
    Bullseye,
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td, ExpandableRowContent } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { MOCK_BASE_IMAGE_CVES } from '../mockData';
import type { CVESeverity } from '../types';

type BaseImageCVEsTabProps = {
    baseImageId: string;
};

function getSeverityIcon(severity: CVESeverity) {
    switch (severity) {
        case 'CRITICAL':
            return SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
        case 'HIGH':
            return SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
        case 'MEDIUM':
            return SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
        case 'LOW':
            return SeverityIcons.LOW_VULNERABILITY_SEVERITY;
        default:
            return SeverityIcons.UNKNOWN_VULNERABILITY_SEVERITY;
    }
}

function getSeverityLabel(severity: CVESeverity) {
    switch (severity) {
        case 'CRITICAL':
            return vulnerabilitySeverityLabels.CRITICAL_VULNERABILITY_SEVERITY;
        case 'HIGH':
            return vulnerabilitySeverityLabels.IMPORTANT_VULNERABILITY_SEVERITY;
        case 'MEDIUM':
            return vulnerabilitySeverityLabels.MODERATE_VULNERABILITY_SEVERITY;
        case 'LOW':
            return vulnerabilitySeverityLabels.LOW_VULNERABILITY_SEVERITY;
        default:
            return vulnerabilitySeverityLabels.UNKNOWN_VULNERABILITY_SEVERITY;
    }
}

/**
 * CVEs tab for base image detail page
 */
function BaseImageCVEsTab({ baseImageId }: BaseImageCVEsTabProps) {
    const [expandedCveIds, setExpandedCveIds] = useState<Set<string>>(new Set());

    // Reset states when baseImageId changes
    React.useEffect(() => {
        setExpandedCveIds(new Set());
    }, [baseImageId]);

    const cves = useMemo(() => MOCK_BASE_IMAGE_CVES[baseImageId] || [], [baseImageId]);

    const toggleRowExpanded = (cveId: string) => {
        setExpandedCveIds((prev) => {
            const newSet = new Set(prev);
            if (newSet.has(cveId)) {
                newSet.delete(cveId);
            } else {
                newSet.add(cveId);
            }
            return newSet;
        });
    };

    if (cves.length === 0) {
        return (
            <Bullseye>
                <EmptyState>
                    <EmptyStateHeader
                        titleText="No CVEs found"
                        icon={<EmptyStateIcon icon={SearchIcon} />}
                        headingLevel="h2"
                    />
                </EmptyState>
            </Bullseye>
        );
    }

    return (
        <>
            {cves.length === 0 ? (
                <Bullseye>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="No CVEs match the current filters"
                            icon={<EmptyStateIcon icon={SearchIcon} />}
                            headingLevel="h3"
                        />
                    </EmptyState>
                </Bullseye>
            ) : (
                <Table variant="compact" borders>
                    <Thead noWrap>
                        <Tr>
                            <Th />
                            <Th>CVE ID</Th>
                            <Th>Severity</Th>
                            <Th>CVSS Score</Th>
                            <Th>Summary</Th>
                            <Th>Fixed By</Th>
                        </Tr>
                    </Thead>
                    {cves.map((cve, rowIndex) => {
                        const isExpanded = expandedCveIds.has(cve.cveId);
                        const SeverityIcon = getSeverityIcon(cve.severity);

                        return (
                            <Tbody key={cve.cveId} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => toggleRowExpanded(cve.cveId),
                                        }}
                                    />
                                    <Td dataLabel="CVE ID">{cve.cveId}</Td>
                                    <Td dataLabel="Severity">
                                        <div
                                            style={{
                                                display: 'flex',
                                                alignItems: 'center',
                                                gap: '8px',
                                            }}
                                        >
                                            <SeverityIcon title={getSeverityLabel(cve.severity)} />
                                            <span>{getSeverityLabel(cve.severity)}</span>
                                        </div>
                                    </Td>
                                    <Td dataLabel="CVSS Score">{cve.cvssScore.toFixed(1)}</Td>
                                    <Td dataLabel="Summary">
                                        <div
                                            style={{
                                                maxWidth: '400px',
                                                overflow: 'hidden',
                                                textOverflow: 'ellipsis',
                                                whiteSpace: 'nowrap',
                                            }}
                                            title={cve.summary}
                                        >
                                            {cve.summary}
                                        </div>
                                    </Td>
                                    <Td dataLabel="Fixed By">
                                        {cve.fixedBy || (
                                            <Text component="small">No fix available</Text>
                                        )}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td colSpan={6}>
                                        <ExpandableRowContent>
                                            <div style={{ padding: '16px' }}>
                                                <Text className="pf-v5-u-font-weight-bold pf-v5-u-mb-sm">
                                                    Affected Components:
                                                </Text>
                                                <Table variant="compact" borders={false}>
                                                    <Thead>
                                                        <Tr>
                                                            <Th>Component Name</Th>
                                                            <Th>Version</Th>
                                                            <Th>Layer Index</Th>
                                                        </Tr>
                                                    </Thead>
                                                    <Tbody>
                                                        {cve.components.map((component) => (
                                                            <Tr
                                                                key={`${component.name}-${component.version}`}
                                                            >
                                                                <Td>{component.name}</Td>
                                                                <Td>{component.version}</Td>
                                                                <Td>{component.layerIndex}</Td>
                                                            </Tr>
                                                        ))}
                                                    </Tbody>
                                                </Table>
                                            </div>
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </Table>
            )}
        </>
    );
}

export default BaseImageCVEsTab;
