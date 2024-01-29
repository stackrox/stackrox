import React from 'react';
import { Link, generatePath } from 'react-router-dom';
import { Flex, List, ListItem, Text, pluralize } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';

const vulnerabilitiesWorkloadCveSinglePath = `${vulnerabilitiesWorkloadCvesPath}/cves/:cve`;

export type CveSelectionsProps = {
    cves: { cve: string; summary: string; numAffectedImages: number }[];
};

function CveSelections({ cves }: CveSelectionsProps) {
    return (
        <List isPlain isBordered>
            {cves.map(({ cve, summary, numAffectedImages }) => (
                <ListItem key={cve}>
                    <Flex direction={{ default: 'column' }}>
                        <Flex direction={{ default: 'row' }}>
                            <ExternalLink>
                                <Link
                                    target="_blank"
                                    to={generatePath(vulnerabilitiesWorkloadCveSinglePath, { cve })}
                                >
                                    {cve}
                                </Link>
                            </ExternalLink>
                            <Text>Across {pluralize(numAffectedImages, 'image')}</Text>
                        </Flex>
                        <Text>{summary}</Text>
                    </Flex>
                </ListItem>
            ))}
        </List>
    );
}

export default CveSelections;
