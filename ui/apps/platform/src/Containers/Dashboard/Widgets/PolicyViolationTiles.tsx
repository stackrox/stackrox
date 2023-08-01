import React, { CSSProperties } from 'react';
import { Button, ButtonVariant, Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';

import { violationsBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { severityLabels } from 'messages/common';
import { policySeverityColorMap } from 'constants/visuals/colors';
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
            className="pf-severity-tile pf-u-w-100 pf-u-px-md pf-u-py-sm pf-u-align-items-center"
            key={severity}
            variant={ButtonVariant.link}
            component={LinkShim}
            href={link}
        >
            <Stack>
                <StackItem className="pf-u-font-weight-bold pf-u-font-size-xl">
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
    return `${violationsBasePath}?${queryString}`;
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
