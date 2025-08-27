import React from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { LabelGroup } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilityState } from 'types/cve.proto';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import ExpandRowTh from 'Components/ExpandRowTh';
import { TableUIState } from 'utils/getTableUIState';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';

// import KnownExploitLabel from '../../components/KnownExploitLabel'; // Ross CISA KEV
import PendingExceptionLabel from '../../components/PendingExceptionLabel';
import DeploymentComponentVulnerabilitiesTable, {
    deploymentComponentVulnerabilitiesFragment,
} from './DeploymentComponentVulnerabilitiesTable';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
import { FormattedDeploymentVulnerability, formatEpssProbabilityAsPercent } from './table.utils';

export const tableId = 'WorkloadCvesDeploymentVulnerabilitiesTable';
export const defaultColumns = {
    rowExpansion: {
        title: 'Row expansion',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cve: {
        title: 'CVE',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    operatingSystem: {
        title: 'Operating system',
        isShownByDefault: true,
    },
    cveSeverity: {
        title: 'CVE severity',
        isShownByDefault: true,
    },
    cveStatus: {
        title: 'CVE status',
        isShownByDefault: true,
    },
    epssProbability: {
        title: 'EPSS probability',
        isShownByDefault: true,
    },
    affectedComponents: {
        title: 'Affected components',
        isShownByDefault: true,
    },
    firstDiscovered: {
        title: 'First discovered',
        isShownByDefault: true,
    },
    publishedOn: {
        title: 'Published',
        isShownByDefault: true,
    },
} as const;

export const deploymentWithVulnerabilitiesFragment = gql`
    ${deploymentComponentVulnerabilitiesFragment}
    fragment DeploymentWithVulnerabilities on Deployment {
        id
        images(query: $query) {
            ...ImageMetadataContext
        }
        imageVulnerabilities(query: $query, pagination: $pagination) {
            vulnerabilityId: id
            cve
            cveBaseInfo {
                epss {
                    epssProbability
                }
            }
            operatingSystem
            publishedOn
            summary
            pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
            images(query: $query) {
                imageId: id
                imageComponents(query: $query) {
                    ...DeploymentComponentVulnerabilities
                }
            }
        }
    }
`;

export type DeploymentVulnerabilitiesTableProps = {
    tableState: TableUIState<FormattedDeploymentVulnerability>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    vulnerabilityState: VulnerabilityState;
    onClearFilters: () => void;
    tableConfig: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function DeploymentVulnerabilitiesTable({
    tableState,
    getSortParams,
    isFiltered,
    vulnerabilityState,
    onClearFilters,
    tableConfig,
}: DeploymentVulnerabilitiesTableProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const expandedRowSet = useSet<string>();
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh className={getVisibilityClass('rowExpansion')} />
                    <Th className={getVisibilityClass('cve')} sort={getSortParams('CVE')}>
                        CVE
                    </Th>
                    <Th className={getVisibilityClass('operatingSystem')}>Operating system</Th>
                    <Th
                        className={getVisibilityClass('cveSeverity')}
                        sort={getSortParams('Severity')}
                    >
                        CVE severity
                    </Th>
                    <Th className={getVisibilityClass('cveStatus')}>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th
                        className={getVisibilityClass('epssProbability')}
                        info={infoForEpssProbability}
                        sort={getSortParams('EPSS Probability')}
                    >
                        EPSS probability
                    </Th>
                    <Th className={getVisibilityClass('affectedComponents')}>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('firstDiscovered')}>First discovered</Th>
                    <Th className={getVisibilityClass('publishedOn')}>Published</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                colSpan={colSpan}
                tableState={tableState}
                emptyProps={{ message: 'There were no CVEs detected for this deployment' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((vulnerability, rowIndex) => {
                        const {
                            vulnerabilityId,
                            cve,
                            cveBaseInfo,
                            operatingSystem,
                            severity,
                            summary,
                            isFixable,
                            images,
                            affectedComponentsText,
                            discoveredAtImage,
                            publishedOn,
                            pendingExceptionCount,
                        } = vulnerability;
                        const epssProbability = cveBaseInfo?.epss?.epssProbability;

                        const labels: ReactNode[] = [];
                        /*
                        // Ross CISA KEV
                        if (isFeatureFlagEnabled('ROX_WHATEVER') && TODO) {
                            labels.push(<KnownExploitLabel isCompact />);
                        }
                        */
                        if (pendingExceptionCount > 0) {
                            labels.push(
                                <PendingExceptionLabel
                                    cve={cve}
                                    isCompact
                                    vulnerabilityState={vulnerabilityState}
                                />
                            );
                        }

                        const isExpanded = expandedRowSet.has(vulnerabilityId);

                        // Table borders={false} prop above and Tbody style prop below
                        // to prevent unwanted border between main row and conditional labels row.
                        //
                        // Td style={{ paddingTop: 0 }} prop emulates vertical space when label was in cell instead of row
                        // and assumes adjacent empty cell has no paddingTop.
                        return (
                            <Tbody
                                key={vulnerabilityId}
                                style={{
                                    borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                }}
                                isExpanded={isExpanded}
                            >
                                <Tr>
                                    <Td
                                        className={getVisibilityClass('rowExpansion')}
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(vulnerabilityId),
                                        }}
                                    />
                                    <Td
                                        className={getVisibilityClass('cve')}
                                        dataLabel="CVE"
                                        modifier="nowrap"
                                    >
                                        <Link to={urlBuilder.cveDetails(cve, vulnerabilityState)}>
                                            {cve}
                                        </Link>
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('operatingSystem')}
                                        modifier="nowrap"
                                        dataLabel="Operating system"
                                    >
                                        {operatingSystem}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveSeverity')}
                                        modifier="nowrap"
                                        dataLabel="CVE severity"
                                    >
                                        <VulnerabilitySeverityIconText severity={severity} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveStatus')}
                                        modifier="nowrap"
                                        dataLabel="CVE status"
                                    >
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('epssProbability')}
                                        modifier="nowrap"
                                        dataLabel="EPSS probability"
                                    >
                                        {formatEpssProbabilityAsPercent(epssProbability)}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('affectedComponents')}
                                        dataLabel="Affected components"
                                    >
                                        {affectedComponentsText}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('firstDiscovered')}
                                        modifier="nowrap"
                                        dataLabel="First discovered"
                                    >
                                        <DateDistance date={discoveredAtImage} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('publishedOn')}
                                        modifier="nowrap"
                                        dataLabel="Published"
                                    >
                                        {publishedOn ? (
                                            <DateDistance date={publishedOn} />
                                        ) : (
                                            'Not available'
                                        )}
                                    </Td>
                                </Tr>
                                {labels.length !== 0 && (
                                    <Tr>
                                        <Td />
                                        <Td colSpan={colSpan - 1} style={{ paddingTop: 0 }}>
                                            <LabelGroup numLabels={labels.length}>
                                                {labels}
                                            </LabelGroup>
                                        </Td>
                                    </Tr>
                                )}
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            <>
                                                {summary && (
                                                    <p className="pf-v5-u-mb-md">{summary}</p>
                                                )}
                                                {images.length > 0 ? (
                                                    <DeploymentComponentVulnerabilitiesTable
                                                        images={images}
                                                        cve={cve}
                                                        vulnerabilityState={vulnerabilityState}
                                                    />
                                                ) : (
                                                    <PartialCVEDataAlert />
                                                )}
                                            </>
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default DeploymentVulnerabilitiesTable;
