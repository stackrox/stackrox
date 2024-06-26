import React from 'react';
import { Link } from 'react-router-dom';
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
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';

import DeploymentComponentVulnerabilitiesTable, {
    deploymentComponentVulnerabilitiesFragment,
} from './DeploymentComponentVulnerabilitiesTable';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import { FormattedDeploymentVulnerability } from './table.utils';

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
            operatingSystem
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
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make Required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    onClearFilters: () => void;
};

function DeploymentVulnerabilitiesTable({
    tableState,
    getSortParams,
    isFiltered,
    vulnerabilityState,
    onClearFilters,
}: DeploymentVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <Th>OS</Th>
                    <Th sort={getSortParams('Severity')}>CVE severity</Th>
                    <Th>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                colSpan={7}
                tableState={tableState}
                emptyProps={{ message: 'There were no CVEs detected for this deployment' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((vulnerability, rowIndex) => {
                        const {
                            vulnerabilityId,
                            cve,
                            operatingSystem,
                            severity,
                            summary,
                            isFixable,
                            images,
                            affectedComponentsText,
                            discoveredAtImage,
                            pendingExceptionCount,
                        } = vulnerability;
                        const isExpanded = expandedRowSet.has(vulnerabilityId);

                        return (
                            <Tbody key={vulnerabilityId} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(vulnerabilityId),
                                        }}
                                    />
                                    <Td dataLabel="CVE" modifier="nowrap">
                                        <PendingExceptionLabelLayout
                                            hasPendingException={pendingExceptionCount > 0}
                                            cve={cve}
                                            vulnerabilityState={vulnerabilityState}
                                        >
                                            <Link
                                                to={getWorkloadEntityPagePath(
                                                    'CVE',
                                                    cve,
                                                    vulnerabilityState
                                                )}
                                            >
                                                {cve}
                                            </Link>
                                        </PendingExceptionLabelLayout>
                                    </Td>
                                    <Td modifier="nowrap" dataLabel="OS">
                                        {operatingSystem}
                                    </Td>
                                    <Td modifier="nowrap" dataLabel="Severity">
                                        <VulnerabilitySeverityIconText severity={severity} />
                                    </Td>
                                    <Td modifier="nowrap" dataLabel="CVE Status">
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
                                    </Td>
                                    <Td dataLabel="Affected components">
                                        {affectedComponentsText}
                                    </Td>
                                    <Td modifier="nowrap" dataLabel="First discovered">
                                        <DateDistance date={discoveredAtImage} />
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={6}>
                                        <ExpandableRowContent>
                                            {summary && images.length > 0 ? (
                                                <>
                                                    <p className="pf-v5-u-mb-md">{summary}</p>
                                                    <DeploymentComponentVulnerabilitiesTable
                                                        images={images}
                                                        cve={cve}
                                                        vulnerabilityState={vulnerabilityState}
                                                    />
                                                </>
                                            ) : (
                                                <PartialCVEDataAlert />
                                            )}
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
