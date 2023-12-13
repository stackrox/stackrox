import React from 'react';
import { Link, generatePath } from 'react-router-dom';
import { Flex, List, ListItem, Text, pluralize, Button, FlexItem } from '@patternfly/react-core';
import { MinusCircleIcon, PlusCircleIcon } from '@patternfly/react-icons';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';

const vulnerabilitiesWorkloadCveSinglePath = `${vulnerabilitiesWorkloadCvesPath}/cves/:cve`;

export type CveSelectionsProps = {
    cves: { cve: string; summary: string; numAffectedImages: number }[];
    selectedCVEIds: string[];
    onAdd: (cve: string) => void;
    onRemove: (cve: string) => void;
};

function CveSelections({ cves, selectedCVEIds, onAdd, onRemove }: CveSelectionsProps) {
    const onAddHandler = (cve: string) => () => {
        onAdd(cve);
    };

    const onRemoveHandler = (cve: string) => () => {
        onRemove(cve);
    };

    return (
        <List isPlain isBordered>
            {cves.map(({ cve, summary, numAffectedImages }) => {
                const isSelected = selectedCVEIds.includes(cve);
                return (
                    <ListItem key={cve}>
                        <Flex direction={{ default: 'column' }}>
                            <Flex direction={{ default: 'row' }}>
                                <ExternalLink>
                                    <Link
                                        target="_blank"
                                        to={generatePath(vulnerabilitiesWorkloadCveSinglePath, {
                                            cve,
                                        })}
                                    >
                                        {cve}
                                    </Link>
                                </ExternalLink>
                                <Text>Across {pluralize(numAffectedImages, 'image')}</Text>
                                <FlexItem align={{ default: 'alignRight' }}>
                                    {isSelected ? (
                                        <Button
                                            variant="link"
                                            aria-label={`Remove ${cve}`}
                                            onClick={onRemoveHandler(cve)}
                                            icon={<MinusCircleIcon />}
                                            isDanger
                                        >
                                            Remove
                                        </Button>
                                    ) : (
                                        <Button
                                            variant="link"
                                            aria-label={`Add ${cve}`}
                                            onClick={onAddHandler(cve)}
                                            icon={<PlusCircleIcon />}
                                        >
                                            Add
                                        </Button>
                                    )}
                                </FlexItem>
                            </Flex>
                            <Text>{summary}</Text>
                        </Flex>
                    </ListItem>
                );
            })}
        </List>
    );
}

export default CveSelections;
