import React from 'react';
import { HashLink as Link } from 'react-router-hash-link';
import { Tooltip, Truncate } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { SecurityIcon } from '@patternfly/react-icons';

import { ImageName } from 'types/image.proto';
import { severityColors } from 'constants/visuals/colors';
import { vulnManagementPath } from 'routePaths';

type VulnCounts = {
    total: number;
    fixable: number;
};

export type ImageData = {
    images: {
        id: string;
        name: Partial<ImageName>;
        priority: number;
        vulnCounter: {
            important: VulnCounts;
            critical: VulnCounts;
        };
    }[];
};

export type CveStatusOption = 'Fixable' | 'All';

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
                {images.map(({ id, name, priority, vulnCounter }) => (
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
                            <SecurityIcon
                                className="pf-u-display-inline pf-u-mr-xs"
                                color={severityColors.critical}
                            />
                            <span>
                                {cveStatusOption === 'Fixable'
                                    ? `${vulnCounter.critical.fixable} fixable`
                                    : `${vulnCounter.critical.total} CVEs`}
                            </span>
                        </Td>
                        <Td className="pf-u-pr-0" dataLabel={columnNames.importantCves}>
                            <SecurityIcon
                                className="pf-u-display-inline pf-u-mr-xs"
                                color={severityColors.important}
                            />
                            {cveStatusOption === 'Fixable'
                                ? `${vulnCounter.important.fixable} fixable`
                                : `${vulnCounter.important.total} CVEs`}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default ImagesAtMostRiskTable;
