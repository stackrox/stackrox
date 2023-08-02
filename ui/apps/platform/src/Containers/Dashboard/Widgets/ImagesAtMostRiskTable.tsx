import React from 'react';
import { HashLink as Link } from 'react-router-hash-link';
import { Tooltip, Truncate } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { CriticalSeverityIcon, ImportantSeverityIcon } from 'Components/PatternFly/SeverityIcons';
import { noViolationsColor } from 'constants/severityColors';
import { ImageName } from 'types/image.proto';
import { vulnManagementPath } from 'routePaths';

type VulnCounts = {
    total: number;
    fixable: number;
};

type ImageVulnerabilityCounter = {
    important: VulnCounts;
    critical: VulnCounts;
};

export type ImageData = {
    images: {
        id: string;
        name: Partial<ImageName>;
        priority: number;
        imageVulnerabilityCounter: ImageVulnerabilityCounter;
    }[];
};

export type CveStatusOption = 'Fixable' | 'All';

function countCritical(
    imageVulnerabilityCounter: ImageVulnerabilityCounter,
    cveStatusOption: CveStatusOption
) {
    return cveStatusOption === 'Fixable'
        ? imageVulnerabilityCounter.critical.fixable
        : imageVulnerabilityCounter.critical.total;
}

function countImportant(
    imageVulnerabilityCounter: ImageVulnerabilityCounter,
    cveStatusOption: CveStatusOption
) {
    return cveStatusOption === 'Fixable'
        ? imageVulnerabilityCounter.important.fixable
        : imageVulnerabilityCounter.important.total;
}

export type ImagesAtMostRiskProps = {
    imageData: ImageData;
    cveStatusOption: CveStatusOption;
};

const columnNames = {
    imageName: 'Images',
    riskPriority: 'Risk priority',
    criticalCves: 'Critical CVEs',
    importantCves: 'Important CVEs',
};

function linkToImage(id: string) {
    return `${vulnManagementPath}/image/${id}#image-findings`;
}

function ImagesAtMostRiskTable({ imageData: { images }, cveStatusOption }: ImagesAtMostRiskProps) {
    return (
        <TableComposable variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th width={35} className="pf-u-pl-0">
                        {columnNames.imageName}
                    </Th>
                    <Th className="pf-u-text-align-center-on-md">{columnNames.riskPriority}</Th>
                    <Th>{columnNames.criticalCves}</Th>
                    <Th className="pf-u-pr-0">{columnNames.importantCves}</Th>
                </Tr>
            </Thead>
            <Tbody>
                {images.map(({ id, name, priority, imageVulnerabilityCounter }) => (
                    <Tr key={id}>
                        <Td className="pf-u-pl-0" dataLabel={columnNames.imageName}>
                            <Link
                                to={linkToImage(id)}
                                scroll={(el: HTMLElement) =>
                                    // TODO This is a heavy handed way to scroll to the CVE section which is loaded on
                                    // the target image page asynchronously. Without a delay, following data loads
                                    // scroll the target element back off the screen.
                                    setTimeout(() => el.scrollIntoView({ behavior: 'smooth' }), 500)
                                }
                            >
                                <Tooltip content={<div>{name.fullName}</div>}>
                                    <Truncate
                                        content={name.remote ?? ''}
                                        position="middle"
                                        trailingNumChars={13}
                                    />
                                </Tooltip>
                            </Link>
                        </Td>
                        <Td
                            className="pf-u-text-align-center-on-md"
                            dataLabel={columnNames.riskPriority}
                        >
                            {priority}
                        </Td>
                        <Td dataLabel={columnNames.criticalCves}>
                            <CriticalSeverityIcon
                                className="pf-u-display-inline pf-u-mr-xs"
                                color={
                                    countCritical(imageVulnerabilityCounter, cveStatusOption) === 0
                                        ? noViolationsColor
                                        : undefined
                                }
                            />
                            <span>
                                {cveStatusOption === 'Fixable'
                                    ? `${imageVulnerabilityCounter.critical.fixable} fixable`
                                    : `${imageVulnerabilityCounter.critical.total} CVEs`}
                            </span>
                        </Td>
                        <Td className="pf-u-pr-0" dataLabel={columnNames.importantCves}>
                            <ImportantSeverityIcon
                                className="pf-u-display-inline pf-u-mr-xs"
                                color={
                                    countImportant(imageVulnerabilityCounter, cveStatusOption) === 0
                                        ? noViolationsColor
                                        : undefined
                                }
                            />
                            {cveStatusOption === 'Fixable'
                                ? `${imageVulnerabilityCounter.important.fixable} fixable`
                                : `${imageVulnerabilityCounter.important.total} CVEs`}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default ImagesAtMostRiskTable;
