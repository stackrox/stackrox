import React from 'react';
import { Button, ButtonVariant } from '@patternfly/react-core';
import {
    TableComposable,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
    ExpandableRowContent,
} from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';

import LinkShim from 'Components/PatternFly/LinkShim';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import useSet from 'hooks/useSet';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { ImageVulnerabilitiesResponse } from '../hooks/useImageVulnerabilities';
import { getEntityPagePath } from '../searchUtils';
import ImageComponentsTable from './ImageComponentsTable';
import { ImageDetailsResponse } from '../hooks/useImageDetails';

export type SingleEntityVulnerabilitiesTableProps = {
    image: ImageDetailsResponse['image'] | undefined;
    imageVulnerabilities: ImageVulnerabilitiesResponse['image']['imageVulnerabilities'];
};

function SingleEntityVulnerabilitiesTable({
    image,
    imageVulnerabilities,
}: SingleEntityVulnerabilitiesTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        <TableComposable>
            <Thead>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th>CVE</Th>
                    <Th>Severity</Th>
                    <Th>CVE status</Th>
                    <Th>Affected components</Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {imageVulnerabilities.map(
                (
                    { cve, severity, summary, isFixable, imageComponents, discoveredAtImage },
                    rowIndex
                ) => {
                    const SeverityIcon: React.FC<SVGIconProps> | undefined =
                        SeverityIcons[severity];
                    const severityLabel: string | undefined = vulnerabilitySeverityLabels[severity];
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
                                    {isFixable ? (
                                        <span>
                                            <CheckCircleIcon
                                                className="pf-u-display-inline"
                                                color="var(--pf-global--success-color--100)"
                                            />
                                            <span className="pf-u-pl-sm">Fixable</span>
                                        </span>
                                    ) : (
                                        <span>
                                            <ExclamationCircleIcon
                                                className="pf-u-display-inline"
                                                color="var(--pf-global--danger-color--100)"
                                            />
                                            <span className="pf-u-pl-sm">Not fixable</span>
                                        </span>
                                    )}
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
                                        <p>{summary}</p>
                                        <div
                                            className="pf-u-p-md pf-u-mt-md"
                                            style={{
                                                border: '1px solid var(--pf-c-table--BorderColor)',
                                            }}
                                        >
                                            <ImageComponentsTable
                                                image={image}
                                                imageComponents={imageComponents}
                                            />
                                        </div>
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
