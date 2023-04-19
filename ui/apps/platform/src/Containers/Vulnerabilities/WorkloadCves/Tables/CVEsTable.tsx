import React from 'react';
import { gql } from '@apollo/client';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant, Tooltip } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getDistanceStrictAsPhrase, getDateTime } from 'utils/dateUtils';
import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';

import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';

export const cveListQuery = gql`
    query getImageCVEList($query: String, $pagination: Pagination) {
        imageCVEs(query: $query, pagination: $pagination) {
            cve
            affectedImageCountBySeverity {
                critical
                important
                moderate
                low
            }
            topCVSS
            affectedImageCount
            firstDiscoveredInSystem
        }
    }
`;

type ImageCVE = {
    cve: string;
    // summary: string;
    affectedImageCountBySeverity: {
        critical: number;
        important: number;
        moderate: number;
        low: number;
    };
    topCVSS: string;
    affectedImageCount: number;
    firstDiscoveredInSystem: Date | null;
};

type CVEsTableProps = {
    cves: ImageCVE[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function CVEsTable({ cves, getSortParams, isFiltered }: CVEsTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        <TableComposable borders={false} variant="compact">
            <Thead>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <Th tooltip="Severity of this CVE across images">
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th tooltip="Highest CVSS score of this CVE across images">Top CVSS</Th>
                    <Th tooltip="Ratio of total environment affect by this CVE">
                        Affected images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th tooltip="Time since this CVE first affected an entity">First discovered</Th>
                </Tr>
            </Thead>
            {cves.map(
                (
                    {
                        cve,
                        // summary,
                        affectedImageCountBySeverity,
                        topCVSS,
                        affectedImageCount,
                        firstDiscoveredInSystem,
                    },
                    rowIndex
                ) => {
                    const isExpanded = expandedRowSet.has(cve);

                    return (
                        <Tbody
                            key={cve}
                            style={{
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                            }}
                            isExpanded={isExpanded}
                        >
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => expandedRowSet.toggle(cve),
                                    }}
                                />
                                <Td>
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('CVE', cve)}
                                    >
                                        {cve}
                                    </Button>
                                </Td>
                                <Td>
                                    <SeverityCountLabels
                                        critical={affectedImageCountBySeverity.critical}
                                        important={affectedImageCountBySeverity.important}
                                        moderate={affectedImageCountBySeverity.moderate}
                                        low={affectedImageCountBySeverity.low}
                                    />
                                </Td>
                                <Td>{topCVSS}</Td>
                                <Td>{affectedImageCount}</Td>
                                <Td>
                                    <Tooltip content={getDateTime(firstDiscoveredInSystem)}>
                                        <div>
                                            {getDistanceStrictAsPhrase(
                                                firstDiscoveredInSystem,
                                                new Date()
                                            )}
                                        </div>
                                    </Tooltip>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </TableComposable>
    );
}

export default CVEsTable;
