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

import LinkShim from 'Components/PatternFly/LinkShim';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { isVulnerabilitySeverity } from 'types/cve.proto';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
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
};

function ImageVulnerabilitiesTable({
    image,
    getSortParams,
    isFiltered,
}: ImageVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <TableComposable variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
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
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={6}>
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
