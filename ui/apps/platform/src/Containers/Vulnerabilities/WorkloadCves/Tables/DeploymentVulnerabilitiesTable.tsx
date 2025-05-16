import React from 'react';
import { Link } from 'react-router-dom';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

// Omit for 4.7 release until CVE/advisory separatipn is available in 4.8 release.
// import useFeatureFlags from 'hooks/useFeatureFlags';
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
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';

import DeploymentComponentVulnerabilitiesTable, {
    convertToFlatDeploymentComponentVulnerabilitiesFragment, // deploymentComponentVulnerabilitiesFragment
} from './DeploymentComponentVulnerabilitiesTable';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
import { FormattedDeploymentVulnerability, formatEpssProbabilityAsPercent } from './table.utils';

export const tableId = 'WorkloadCvesDeploymentVulnerabilitiesTable';
export const defaultColumns = {
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

// After release, replace temporary function
// with deploymentWithVulnerabilitiesFragment
// that has unconditional deploymentComponentVulnerabilitiesFragment.
export function convertToFlatDeploymentWithVulnerabilitiesFragment(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
        ${convertToFlatDeploymentComponentVulnerabilitiesFragment(isFlattenCveDataEnabled)}
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
}

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
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const expandedRowSet = useSet<string>();
    // Omit for 4.7 release until CVE/advisory separatipn is available in 4.8 release.
    // const { isFeatureFlagEnabled } = useFeatureFlags();
    // const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = false;

    const colSpan = 7 + (isEpssProbabilityColumnEnabled ? 1 : 0) - hiddenColumnCount;

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams('CVE')}>CVE</Th>
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
                    {isEpssProbabilityColumnEnabled && (
                        <Th
                            className={getVisibilityClass('epssProbability')}
                            info={infoForEpssProbability}
                            sort={getSortParams('EPSS Probability')}
                        >
                            EPSS probability
                        </Th>
                    )}
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
                                                to={getAbsoluteUrl(
                                                    getWorkloadEntityPagePath(
                                                        'CVE',
                                                        cve,
                                                        vulnerabilityState
                                                    )
                                                )}
                                            >
                                                {cve}
                                            </Link>
                                        </PendingExceptionLabelLayout>
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
                                    {isEpssProbabilityColumnEnabled && (
                                        <Td
                                            className={getVisibilityClass('epssProbability')}
                                            modifier="nowrap"
                                            dataLabel="EPSS probability"
                                        >
                                            {formatEpssProbabilityAsPercent(epssProbability)}
                                        </Td>
                                    )}
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
