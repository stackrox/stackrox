import React from 'react';
import { gql } from '@apollo/client';
import {
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
    ExpandableRowContent,
} from '@patternfly/react-table';
import { Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';

import { getEntityPagePath } from '../searchUtils';
import TooltipTh from '../components/TooltipTh';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import DatePhraseTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';

export const cveListQuery = gql`
    query getImageCVEList($query: String, $pagination: Pagination) {
        imageCVEs(query: $query, pagination: $pagination) {
            cve
            affectedImageCountBySeverity {
                critical {
                    total
                }
                important {
                    total
                }
                moderate {
                    total
                }
                low {
                    total
                }
            }
            topCVSS
            affectedImageCount
            firstDiscoveredInSystem
        }
    }
`;

export const unfilteredImageCountQuery = gql`
    query getUnfilteredImageCount {
        imageCount
    }
`;

type ImageCVE = {
    cve: string;
    // summary: string;
    affectedImageCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    topCVSS: number;
    affectedImageCount: number;
    firstDiscoveredInSystem: string | null;
};

type CVEsTableProps = {
    cves: ImageCVE[];
    unfilteredImageCount: number;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function CVEsTable({ cves, unfilteredImageCount, getSortParams, isFiltered }: CVEsTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        <TableComposable borders={false} variant="compact">
            <Thead>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <TooltipTh tooltip="Severity of this CVE across images">
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh tooltip="Highest CVSS score of this CVE across images">
                        Top CVSS
                    </TooltipTh>
                    <TooltipTh tooltip="Ratio of total environment affect by this CVE">
                        Affected images
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh tooltip="Time since this CVE first affected an entity">
                        First discovered
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
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
                                        critical={affectedImageCountBySeverity.critical.total}
                                        important={affectedImageCountBySeverity.important.total}
                                        moderate={affectedImageCountBySeverity.moderate.total}
                                        low={affectedImageCountBySeverity.low.total}
                                    />
                                </Td>
                                {/* TODO: score version? */}
                                <Td>
                                    <CvssTd cvss={topCVSS} />
                                </Td>
                                <Td>
                                    {/* TODO: fix upon PM feedback */}
                                    {affectedImageCount}/{unfilteredImageCount} affected images
                                </Td>
                                <Td>
                                    <DatePhraseTd date={firstDiscoveredInSystem} />
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={6}>
                                    <ExpandableRowContent>
                                        {/* TODO: add summary once it's in */}
                                        Lorem ipsum dolor sit amet, consectetur adipiscing elit. In
                                        a vehicula nisl. Interdum et malesuada fames ac ante ipsum
                                        primis in faucibus. Duis mollis nisi eget augue rhoncus, a
                                        consectetur magna tincidunt. Nam est diam, aliquet at
                                        hendrerit at, venenatis eu est. Integer pulvinar diam ac dui
                                        efficitur finibus. Vestibulum ante ipsum primis in faucibus
                                        orci luctus et ultrices posuere cubilia curae; Cras eu ex
                                        sit amet enim lacinia placerat eget vitae arcu.
                                    </ExpandableRowContent>
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
