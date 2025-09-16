import React, { CSSProperties } from 'react';
import { Button, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';

import { violationsFullViewPath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { severityLabels } from 'messages/common';
import { policySeverityColorMap } from 'constants/severityColors';
import { policySeverities, PolicySeverity } from 'types/policy.proto';
import LinkShim from 'Components/PatternFly/LinkShim';

import './SeverityTile.css';

type SeverityTileProps = {
    severity: PolicySeverity;
    violationCount: number;
    link: string;
};

function SeverityTile({ severity, violationCount, link }: SeverityTileProps) {
    return (
        <Button
            style={
                { '--pf-severity-tile-color': policySeverityColorMap[severity] } as CSSProperties
            }
            className="pf-severity-tile pf-v5-u-w-100 pf-v5-u-px-md pf-v5-u-py-sm pf-v5-u-align-items-center"
            key={severity}
            variant="link"
            component={LinkShim}
            href={link}
        >
            <Stack>
                <StackItem className="pf-v5-u-font-weight-bold pf-v5-u-font-size-xl">
                    {violationCount}
                </StackItem>
                <StackItem>{severityLabels[severity]}</StackItem>
            </Stack>
        </Button>
    );
}

function linkToViolations(searchFilter, severity) {
    const queryString = getUrlQueryStringForSearchFilter({
        ...searchFilter,
        Severity: severity,
    });
    return `${violationsFullViewPath}&${queryString}`;
}

export type PolicyViolationTilesProps = {
    searchFilter: SearchFilter;
    counts: Record<PolicySeverity, number>;
};

function PolicyViolationTiles({ searchFilter, counts }: PolicyViolationTilesProps) {
    return (
        <Flex direction={{ default: 'row' }}>
            {policySeverities.map((severity) => (
                <FlexItem
                    key={severity}
                    style={{ flexBasis: '0px' }}
                    shrink={{ default: 'shrink' }}
                    grow={{ default: 'grow' }}
                >
                    <SeverityTile
                        severity={severity}
                        violationCount={counts[severity]}
                        link={linkToViolations(searchFilter, severity)}
                    />
                </FlexItem>
            ))}
        </Flex>
    );
}

export default PolicyViolationTiles;
