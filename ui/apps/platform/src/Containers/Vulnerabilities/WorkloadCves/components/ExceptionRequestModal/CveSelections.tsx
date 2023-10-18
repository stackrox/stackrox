import React from 'react';
import { Link, generatePath } from 'react-router-dom';
import { Flex, List, ListItem, Text } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';

const vulnerabilitiesWorkloadCveSinglePath = `${vulnerabilitiesWorkloadCvesPath}/cves/:cve`;

export type CveSelectionsProps = {
    cves: { cve: string; summary: string }[];
};

function CveSelections({ cves }: CveSelectionsProps) {
    return (
        <List isPlain isBordered>
            {cves.map(({ cve, summary }) => (
                <ListItem key={cve}>
                    <Flex direction={{ default: 'column' }}>
                        <Text>
                            <Link
                                target="_blank"
                                to={generatePath(vulnerabilitiesWorkloadCveSinglePath, { cve })}
                            >
                                {cve} <ExternalLinkAltIcon className="pf-u-display-inline" />
                            </Link>
                        </Text>
                        <Text>{summary}</Text>
                    </Flex>
                </ListItem>
            ))}
        </List>
    );
}

export default CveSelections;
