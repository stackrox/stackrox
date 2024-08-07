import React from 'react';
import { Link, generatePath } from 'react-router-dom';
import { Flex, List, ListItem, Text, Button, FlexItem, Alert } from '@patternfly/react-core';
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
        <>
            <div className="pf-v5-u-mb-md">
                <Alert
                    title="Include or exclude selected CVEs"
                    component="p"
                    variant="info"
                    isInline
                >
                    You currently have ({selectedCVEIds.length}) selected. Review your selection
                    below.
                </Alert>
            </div>
            <List isPlain isBordered>
                {cves.map(({ cve, summary }) => {
                    const isSelected = selectedCVEIds.includes(cve);
                    return (
                        <ListItem
                            key={cve}
                            className={!isSelected ? 'pf-v5-u-background-color-200' : ''}
                        >
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
                                    <FlexItem align={{ default: 'alignRight' }}>
                                        {isSelected ? (
                                            <Button
                                                variant="link"
                                                aria-label={`Remove ${cve}`}
                                                onClick={onRemoveHandler(cve)}
                                                icon={<MinusCircleIcon />}
                                                isDanger
                                            >
                                                Exclude CVE
                                            </Button>
                                        ) : (
                                            <Button
                                                variant="link"
                                                aria-label={`Add ${cve}`}
                                                onClick={onAddHandler(cve)}
                                                icon={<PlusCircleIcon />}
                                            >
                                                Include CVE
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
        </>
    );
}

export default CveSelections;
