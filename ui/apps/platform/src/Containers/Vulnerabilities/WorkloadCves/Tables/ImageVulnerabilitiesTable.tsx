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

import LinkShim from 'Components/PatternFly/LinkShim';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { isVulnerabilitySeverity } from 'types/cve.proto';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { graphql } from 'generated/graphql-codegen';
import {
    ImageMetadataContextFragment,
    ImageVulnerabilitiesFragment,
} from 'generated/graphql-codegen/graphql';
import { isNonNullish } from 'utils/type.utils';

import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import ImageComponentVulnerabilitiesTable from './ImageComponentVulnerabilitiesTable';

import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';
import { getAnyVulnerabilityIsFixable } from './table.utils';

export const imageVulnerabilitiesFragment = graphql(/* GraphQL */ `
    fragment ImageVulnerabilities on Image {
        imageVulnerabilities(query: $query, pagination: $pagination) {
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
    }
`);

export type ImageVulnerabilitiesTableProps = {
    image: ImageMetadataContextFragment & ImageVulnerabilitiesFragment;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function ImageVulnerabilitiesTable({
    image,
    getSortParams,
    isFiltered,
}: ImageVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();

    const vulnerabilities = image.imageVulnerabilities.filter(isNonNullish);

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
            {vulnerabilities.length === 0 && <EmptyTableResults colSpan={7} />}
            {vulnerabilities.map(
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

                    const components = imageComponents.filter(isNonNullish);

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
                                    {components.length === 1
                                        ? components[0].name
                                        : `${components.length} components`}
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
                                            componentVulnerabilities={components}
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
