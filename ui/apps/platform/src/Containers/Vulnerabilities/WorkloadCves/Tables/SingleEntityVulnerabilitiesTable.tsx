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
import { ImageVulnerabilitiesResponse } from '../hooks/useImageVulnerabilities';
import { getEntityPagePath } from '../searchUtils';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import ComponentVulnerabilitiesTable from './ComponentVulnerabilitiesTable';

export type SingleEntityVulnerabilitiesTableProps = {
    image: ImageVulnerabilitiesResponse['image'];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

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
                    <Th sort={getSortParams('Severity')}>Severity</Th>
                    <Th sort={getSortParams('Fixable')}>
                        CVE Status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    {/* TODO Add sorting for these columns once aggregate sorting is available in BE */}
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {image.imageVulnerabilities.map(
                (
                    { cve, severity, summary, isFixable, imageComponents, discoveredAtImage },
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
                                <Td colSpan={5}>
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
