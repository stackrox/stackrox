import React from 'react';
import { Button, ButtonVariant } from '@patternfly/react-core';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { gql } from '@apollo/client';
import { min } from 'date-fns';

import LinkShim from 'Components/PatternFly/LinkShim';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { FixableIcon, NotFixableIcon } from 'Components/PatternFly/FixabilityIcons';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilitySeverity } from 'types/cve.proto';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';

import EmptyTableResults from '../components/EmptyTableResults';
import DeploymentComponentVulnerabilitiesTable, {
    DeploymentComponentVulnerability,
    ImageMetadataContext,
    deploymentComponentVulnerabilitiesFragment,
} from './DeploymentComponentVulnerabilitiesTable';
import { getAnyVulnerabilityIsFixable, getHighestVulnerabilitySeverity } from './table.utils';
import DatePhraseTd from '../components/DatePhraseTd';

export const deploymentWithVulnerabilitiesFragment = gql`
    ${deploymentComponentVulnerabilitiesFragment}
    fragment DeploymentWithVulnerabilities on Deployment {
        id
        images(query: $query) {
            ...ImageMetadataContext
        }
        imageVulnerabilities(query: $query, pagination: $pagination) {
            id
            cve
            summary
            images(query: $query) {
                imageId: id
                imageComponents(query: $query) {
                    ...DeploymentComponentVulnerabilities
                }
            }
        }
    }
`;

export type DeploymentWithVulnerabilities = {
    id: string;
    images: ImageMetadataContext[];
    imageVulnerabilities: {
        id: string;
        cve: string;
        summary: string;
        images: {
            imageId: string;
            imageComponents: DeploymentComponentVulnerability[];
        }[];
    }[];
};

function formatVulnerabilityData(deployment: DeploymentWithVulnerabilities): {
    id: string;
    cve: string;
    severity: VulnerabilitySeverity;
    isFixable: boolean;
    discoveredAtImage: Date | null;
    summary: string;
    affectedComponentsText: string;
    images: {
        imageMetadataContext: ImageMetadataContext;
        componentVulnerabilities: DeploymentComponentVulnerability[];
    }[];
}[] {
    const imageMap: Record<string, ImageMetadataContext> = {};
    deployment.images.forEach((image) => {
        imageMap[image.id] = image;
    });

    return deployment.imageVulnerabilities.map((vulnerability) => {
        const { id, cve, summary, images } = vulnerability;
        // Severity, Fixability, and Discovered date are all based on the aggregate value of all components
        const allVulnerableComponents = vulnerability.images.flatMap((img) => img.imageComponents);
        const highestVulnSeverity = getHighestVulnerabilitySeverity(allVulnerableComponents);
        const isAnyVulnFixable = getAnyVulnerabilityIsFixable(allVulnerableComponents);
        const allDiscoveredDates = allVulnerableComponents
            .flatMap((c) => c.imageVulnerabilities.map((v) => v.discoveredAtImage))
            .filter((d): d is string => d !== null);
        const oldestDiscoveredVulnDate = min(...allDiscoveredDates);
        // TODO This logic is used in many places, could extract to a util
        const uniqueComponents = new Set(allVulnerableComponents.map((c) => c.name));
        const affectedComponentsText =
            uniqueComponents.size === 1
                ? uniqueComponents.values().next().value
                : `${uniqueComponents.size} components`;

        return {
            id,
            cve,
            severity: highestVulnSeverity,
            isFixable: isAnyVulnFixable,
            discoveredAtImage: oldestDiscoveredVulnDate,
            summary,
            affectedComponentsText,
            images: images.map((img) => ({
                imageMetadataContext: imageMap[img.imageId],
                componentVulnerabilities: img.imageComponents,
            })),
        };
    });
}

export type DeploymentVulnerabilitiesTableProps = {
    deployment: DeploymentWithVulnerabilities;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function DeploymentVulnerabilitiesTable({
    deployment,
    getSortParams,
    isFiltered,
}: DeploymentVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();

    const vulnerabilities = formatVulnerabilityData(deployment);

    return (
        <TableComposable variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <Th>Severity</Th>
                    <Th>
                        CVE Status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {vulnerabilities.length === 0 && <EmptyTableResults colSpan={7} />}
            {vulnerabilities.map((vulnerability, rowIndex) => {
                const {
                    cve,
                    severity,
                    summary,
                    isFixable,
                    images,
                    affectedComponentsText,
                    discoveredAtImage,
                } = vulnerability;
                const isExpanded = expandedRowSet.has(cve);

                const FixabilityIcon = isFixable ? FixableIcon : NotFixableIcon;

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
                            <Td dataLabel="CVE">
                                <Button
                                    variant={ButtonVariant.link}
                                    isInline
                                    component={LinkShim}
                                    href={getEntityPagePath('CVE', cve)}
                                >
                                    {cve}
                                </Button>
                            </Td>
                            <Td modifier="nowrap" dataLabel="Severity">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td modifier="nowrap" dataLabel="CVE Status">
                                <span>
                                    <FixabilityIcon className="pf-u-display-inline" />
                                    <span className="pf-u-pl-sm">
                                        {isFixable ? 'Fixable' : 'Not fixable'}
                                    </span>
                                </span>
                            </Td>
                            <Td dataLabel="Affected components">{affectedComponentsText}</Td>
                            <Td modifier="nowrap" dataLabel="First discovered">
                                <DatePhraseTd date={discoveredAtImage} />
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td />
                            <Td colSpan={6}>
                                <ExpandableRowContent>
                                    <p className="pf-u-mb-md">{summary}</p>
                                    <DeploymentComponentVulnerabilitiesTable images={images} />
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default DeploymentVulnerabilitiesTable;
