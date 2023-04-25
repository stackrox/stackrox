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
import { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';

import LinkShim from 'Components/PatternFly/LinkShim';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import useSet from 'hooks/useSet';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { UseURLSortResult } from 'hooks/useURLSort';
import { FixableIcon, NotFixableIcon } from 'Components/PatternFly/FixabilityIcons';
import { gql } from '@apollo/client';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import ComponentVulnerabilitiesTable, {
    ComponentVulnerability,
    ImageMetadataContext,
    componentVulnerabilitiesFragment,
} from './ComponentVulnerabilitiesTable';

export const imageVulnerabilitiesFragment = gql`
    ${componentVulnerabilitiesFragment}
    fragment ImageVulnerabilityFields on ImageVulnerability {
        id
        severity
        isFixable
        cve
        summary
        cvss
        scoreVersion
        discoveredAtImage
        imageComponents(query: $query) {
            ...ComponentVulnerabilities
        }
    }
`;

export type ImageVulnerability = {
    id: string;
    severity: string;
    isFixable: boolean;
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    discoveredAtImage: Date | null;
    imageComponents: ComponentVulnerability[];
};

export type SingleEntityVulnerabilitiesTableProps = {
    image: ImageMetadataContext & {
        imageVulnerabilities: ImageVulnerability[];
    };
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

// TODO Although the structure of this table is identical for both the Image and Deployment single page
// tables, the data format coming in will be quite different. We will need a layer in between the parent
// component and the table to normalize the data into a common format.
function SingleEntityVulnerabilitiesTable({
    image,
    getSortParams,
    isFiltered,
}: SingleEntityVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <Th>Severity</Th>
                    <Th>
                        CVE Status
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
            {image.imageVulnerabilities.map(
                (
                    {
                        cve,
                        severity,
                        summary,
                        isFixable,
                        cvss,
                        scoreVersion,
                        imageComponents,
                        discoveredAtImage,
                    },
                    rowIndex
                ) => {
                    const SeverityIcon: React.FC<SVGIconProps> | undefined =
                        SeverityIcons[severity];
                    const severityLabel: string | undefined = vulnerabilitySeverityLabels[severity];
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
                                <Td dataLabel="Severity">
                                    <span>
                                        {SeverityIcon && (
                                            <SeverityIcon className="pf-u-display-inline" />
                                        )}
                                        {severityLabel && (
                                            <span className="pf-u-pl-sm">{severityLabel}</span>
                                        )}
                                    </span>
                                </Td>
                                <Td dataLabel="CVE Status">
                                    <span>
                                        <FixabilityIcon className="pf-u-display-inline" />
                                        <span className="pf-u-pl-sm">
                                            {isFixable ? 'Fixable' : 'Not fixable'}
                                        </span>
                                    </span>
                                </Td>
                                <Td dataLabel="CVSS">
                                    {cvss.toFixed(1)} ({scoreVersion})
                                </Td>
                                <Td dataLabel="Affected components">
                                    {imageComponents.length === 1
                                        ? imageComponents[0].name
                                        : `${imageComponents.length} components`}
                                </Td>
                                <Td dataLabel="First discovered">
                                    {getDistanceStrictAsPhrase(discoveredAtImage, new Date())}
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={6}>
                                    <ExpandableRowContent>
                                        <p className="pf-u-mb-md">{summary}</p>
                                        <ComponentVulnerabilitiesTable
                                            showImage={false}
                                            images={[
                                                {
                                                    imageMetadataContext: image,
                                                    componentVulnerabilities: imageComponents,
                                                },
                                            ]}
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

export default SingleEntityVulnerabilitiesTable;
