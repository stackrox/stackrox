import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import ExpandRowTh from 'Components/ExpandRowTh';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import type { TableUIState } from 'utils/getTableUIState';
import { generateVisibilityForColumns, getHiddenColumnCount } from 'hooks/useManagedColumns';
import type { ManagedColumns } from 'hooks/useManagedColumns';

import type { CveTableRow } from '../aggregateUtils';
import {
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';
import VirtualMachineComponentsTable from './VirtualMachineComponentsTable';

export const tableId = 'VirtualMachineCvesVulnerabilitiesTable';
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
    cveSeverity: {
        title: 'CVE severity',
        isShownByDefault: true,
    },
    cveStatus: {
        title: 'CVE status',
        isShownByDefault: true,
    },
    cvss: {
        title: 'CVSS',
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
} as const;

export type VirtualMachineVulnerabilitiesTableProps = {
    tableState: TableUIState<CveTableRow>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
    tableConfig: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function VirtualMachineVulnerabilitiesTable({
    tableState,
    getSortParams,
    onClearFilters,
    tableConfig,
}: VirtualMachineVulnerabilitiesTableProps) {
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;
    const expandedRowSet = useSet<string>();

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={false}
        >
            <Thead>
                <Tr>
                    <ExpandRowTh className={getVisibilityClass('rowExpansion')} />
                    <Th className={getVisibilityClass('cve')} sort={getSortParams(CVE_SORT_FIELD)}>
                        CVE
                    </Th>
                    <Th
                        className={getVisibilityClass('cveSeverity')}
                        sort={getSortParams(CVE_SEVERITY_SORT_FIELD)}
                    >
                        CVE severity
                    </Th>
                    <Th className={getVisibilityClass('cveStatus')}>CVE status</Th>
                    <Th
                        className={getVisibilityClass('cvss')}
                        sort={getSortParams(CVSS_SORT_FIELD)}
                    >
                        CVSS
                    </Th>
                    <Th
                        className={getVisibilityClass('epssProbability')}
                        sort={getSortParams(CVE_EPSS_PROBABILITY_SORT_FIELD)}
                    >
                        EPSS probability
                    </Th>
                    <Th className={getVisibilityClass('affectedComponents')}>
                        Affected components
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                errorProps={{
                    title: 'There was an error loading results',
                }}
                emptyProps={{
                    message: 'No CVEs were detected for this virtual machine',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((vulnerability, rowIndex) => {
                        const isExpanded = expandedRowSet.has(vulnerability.cve);
                        return (
                            <Tbody key={vulnerability.cve} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        className={getVisibilityClass('rowExpansion')}
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () =>
                                                expandedRowSet.toggle(vulnerability.cve),
                                        }}
                                    />
                                    <Td className={getVisibilityClass('cve')} dataLabel="CVE">
                                        {vulnerability.cve}{' '}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveSeverity')}
                                        dataLabel="CVE severity"
                                    >
                                        <VulnerabilitySeverityIconText
                                            severity={vulnerability.severity}
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveStatus')}
                                        dataLabel="CVE status"
                                    >
                                        <VulnerabilityFixableIconText
                                            isFixable={vulnerability.isFixable}
                                        />
                                    </Td>
                                    <Td className={getVisibilityClass('cvss')} dataLabel="CVSS">
                                        <CvssFormatted
                                            cvss={vulnerability.cvss}
                                            scoreVersion="v3"
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('epssProbability')}
                                        dataLabel="EPSS probability"
                                    >
                                        {formatEpssProbabilityAsPercent(
                                            vulnerability.epssProbability
                                        )}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('affectedComponents')}
                                        dataLabel="Affected components"
                                    >
                                        {vulnerability.affectedComponents.length === 1
                                            ? vulnerability.affectedComponents[0].name
                                            : `${vulnerability.affectedComponents.length} components`}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td className={getVisibilityClass('rowExpansion')} />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            <VirtualMachineComponentsTable
                                                components={vulnerability.affectedComponents}
                                            />
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

export default VirtualMachineVulnerabilitiesTable;
