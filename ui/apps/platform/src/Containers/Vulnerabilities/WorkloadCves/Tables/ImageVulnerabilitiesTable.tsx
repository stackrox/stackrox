import React from 'react';
import { Link } from 'react-router-dom';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { VulnerabilityState, isVulnerabilitySeverity } from 'types/cve.proto';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import useMap from 'hooks/useMap';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    ImageMetadataContext,
    imageComponentVulnerabilitiesFragment,
} from './ImageComponentVulnerabilitiesTable';

import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';
import { getAnyVulnerabilityIsFixable } from './table.utils';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { CveSelectionsProps } from '../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../components/CVESelectionTh';
import CVESelectionTd from '../components/CVESelectionTd';
import TooltipTh from '../components/TooltipTh';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';

export const imageVulnerabilitiesFragment = gql`
    ${imageComponentVulnerabilitiesFragment}
    fragment ImageVulnerabilityFields on ImageVulnerability {
        severity
        cve
        summary
        cvss
        scoreVersion
        discoveredAtImage
        imageComponents(query: $query) {
            ...ImageComponentVulnerabilities
        }
    }
`;

export type ImageVulnerability = {
    severity: string;
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    discoveredAtImage: string | null;
    imageComponents: ImageComponentVulnerability[];
};

export type ImageVulnerabilitiesTableProps = {
    image: ImageMetadataContext & {
        imageVulnerabilities: ImageVulnerability[];
    };
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    canSelectRows: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make Required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    createTableActions?: (cve: { cve: string; summary: string }) => IAction[];
};

function ImageVulnerabilitiesTable({
    image,
    getSortParams,
    isFiltered,
    canSelectRows,
    selectedCves,
    vulnerabilityState,
    createTableActions,
}: ImageVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();
    const showExceptionDetailsLink = vulnerabilityState && vulnerabilityState !== 'OBSERVED';

    const colSpan =
        6 +
        (canSelectRows ? 1 : 0) +
        (createTableActions ? 1 : 0) +
        (showExceptionDetailsLink ? 1 : 0);

    return (
        <TableComposable variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <Th sort={getSortParams('Severity')}>CVE Severity</Th>
                    <Th>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('CVSS')}>CVSS</Th>
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                    {showExceptionDetailsLink && (
                        <TooltipTh tooltip="View information about this exception request">
                            Request details
                        </TooltipTh>
                    )}
                    {createTableActions && <Th aria-label="CVE actions" />}
                </Tr>
            </Thead>
            {image.imageVulnerabilities.length === 0 && <EmptyTableResults colSpan={7} />}
            {image.imageVulnerabilities.map(
                (
                    {
                        cve,
                        severity,
                        summary,
                        cvss,
                        scoreVersion,
                        imageComponents,
                        discoveredAtImage,
                    },
                    rowIndex
                ) => {
                    const isFixable = getAnyVulnerabilityIsFixable(imageComponents);
                    const isExpanded = expandedRowSet.has(cve);

                    return (
                        <Tbody key={cve} isExpanded={isExpanded}>
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => expandedRowSet.toggle(cve),
                                    }}
                                />
                                {canSelectRows && (
                                    <CVESelectionTd
                                        selectedCves={selectedCves}
                                        rowIndex={rowIndex}
                                        cve={cve}
                                        summary={summary}
                                    />
                                )}
                                <Td dataLabel="CVE">
                                    <Link to={getEntityPagePath('CVE', cve)}>{cve}</Link>
                                </Td>
                                <Td modifier="nowrap" dataLabel="CVE severity">
                                    {isVulnerabilitySeverity(severity) && (
                                        <VulnerabilitySeverityIconText severity={severity} />
                                    )}
                                </Td>
                                <Td modifier="nowrap" dataLabel="CVE status">
                                    <VulnerabilityFixableIconText isFixable={isFixable} />
                                </Td>
                                <Td modifier="nowrap" dataLabel="CVSS">
                                    <CvssTd cvss={cvss} scoreVersion={scoreVersion} />
                                </Td>
                                <Td dataLabel="Affected components">
                                    {imageComponents.length === 1
                                        ? imageComponents[0].name
                                        : `${imageComponents.length} components`}
                                </Td>
                                <Td dataLabel="First discovered">
                                    <DateDistanceTd date={discoveredAtImage} />
                                </Td>
                                {showExceptionDetailsLink && (
                                    <ExceptionDetailsCell
                                        cve={cve}
                                        vulnerabilityState={vulnerabilityState}
                                    />
                                )}
                                {createTableActions && (
                                    <Td className="pf-u-px-0">
                                        <ActionsColumn
                                            items={createTableActions({ cve, summary })}
                                        />
                                    </Td>
                                )}
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={colSpan}>
                                    <ExpandableRowContent>
                                        <p className="pf-u-mb-md">{summary}</p>
                                        <ImageComponentVulnerabilitiesTable
                                            imageMetadataContext={image}
                                            componentVulnerabilities={imageComponents}
                                        />
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

export default ImageVulnerabilitiesTable;
