import React from 'react';
import { Button, ButtonVariant } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';

import LinkShim from 'Components/PatternFly/LinkShim';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { ImageVulnerabilitiesResponse } from '../hooks/useImageVulnerabilities';
import { getEntityPagePath } from '../searchUtils';

export type SingleEntityVulnerabilitiesTableProps = {
    imageVulnerabilities: ImageVulnerabilitiesResponse['image']['imageVulnerabilities'];
};

function SingleEntityVulnerabilitiesTable({
    imageVulnerabilities,
}: SingleEntityVulnerabilitiesTableProps) {
    return (
        <TableComposable>
            <Thead>
                <Tr>
                    <Th>CVE</Th>
                    <Th>Severity</Th>
                    <Th>CVE Status</Th>
                    <Th>Affected components</Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            <Tbody>
                {imageVulnerabilities.map(
                    ({ cve, severity, isFixable, imageComponents, discoveredAtImage }) => {
                        const SeverityIcon: React.FC<SVGIconProps> | undefined =
                            SeverityIcons[severity];
                        const severityLabel: string | undefined =
                            vulnerabilitySeverityLabels[severity];

                        return (
                            <Tr key={cve}>
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
                        );
                    }
                )}
            </Tbody>
        </TableComposable>
    );
}

export default SingleEntityVulnerabilitiesTable;
