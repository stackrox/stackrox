import React from 'react';
import { Link } from 'react-router-dom';
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

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilitySeverity } from 'types/cve.proto';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';

import EmptyTableResults from '../components/EmptyTableResults';
import DeploymentComponentVulnerabilitiesTable, {
    DeploymentComponentVulnerability,
    ImageMetadataContext,
    deploymentComponentVulnerabilitiesFragment,
} from './DeploymentComponentVulnerabilitiesTable';
import { getAnyVulnerabilityIsFixable, getHighestVulnerabilitySeverity } from './table.utils';
import DateDistanceTd from '../components/DatePhraseTd';

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
        vulnerabilityId: string;
        cve: string;
        summary: string;
        images: {
            imageId: string;
            imageComponents: DeploymentComponentVulnerability[];
        }[];
    }[];
};

type DeploymentVulnerabilityImageMapping = {
    imageMetadataContext: ImageMetadataContext;
    componentVulnerabilities: DeploymentComponentVulnerability[];
};

function formatVulnerabilityData(deployment: DeploymentWithVulnerabilities): {
    vulnerabilityId: string;
    cve: string;
    severity: VulnerabilitySeverity;
    isFixable: boolean;
    discoveredAtImage: Date | null;
    summary: string;
    affectedComponentsText: string;
    images: DeploymentVulnerabilityImageMapping[];
}[] {
    // Create a map of image ID to image metadata for easy lookup
    // We use 'Partial' here because there is no guarantee that the image will be found
    const imageMap: Partial<Record<string, ImageMetadataContext>> = {};
    deployment.images.forEach((image) => {
        imageMap[image.id] = image;
    });

    return deployment.imageVulnerabilities.map((vulnerability) => {
        const { vulnerabilityId, cve, summary, images } = vulnerability;
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

        const vulnerabilityImages = images
            .map((img) => ({
                imageMetadataContext: imageMap[img.imageId],
                componentVulnerabilities: img.imageComponents,
            }))
            // filter out values where the vulnerability->image mapping is missing
            .filter(
                (vulnImageMap): vulnImageMap is DeploymentVulnerabilityImageMapping =>
                    !!vulnImageMap.imageMetadataContext
            );

        return {
            vulnerabilityId,
            cve,
            severity: highestVulnSeverity,
            isFixable: isAnyVulnFixable,
            discoveredAtImage: oldestDiscoveredVulnDate,
            summary,
            affectedComponentsText,
            images: vulnerabilityImages,
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
                    <Th>CVE severity</Th>
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
            {vulnerabilities.length === 0 && <EmptyTableResults colSpan={7} />}
            {vulnerabilities.map((vulnerability, rowIndex) => {
                const {
                    vulnerabilityId,
                    cve,
                    severity,
                    summary,
                    isFixable,
                    images,
                    affectedComponentsText,
                    discoveredAtImage,
                } = vulnerability;
                const isExpanded = expandedRowSet.has(cve);

                return (
                    <Tbody key={vulnerabilityId} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => expandedRowSet.toggle(cve),
                                }}
                            />
                            <Td dataLabel="CVE">
                                <Link to={getEntityPagePath('CVE', cve)}>{cve}</Link>
                            </Td>
                            <Td modifier="nowrap" dataLabel="Severity">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td modifier="nowrap" dataLabel="CVE Status">
                                <VulnerabilityFixableIconText isFixable={isFixable} />
                            </Td>
                            <Td dataLabel="Affected components">{affectedComponentsText}</Td>
                            <Td modifier="nowrap" dataLabel="First discovered">
                                <DateDistanceTd date={discoveredAtImage} />
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
