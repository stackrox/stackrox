import React, { CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem, Stack, StackItem } from '@patternfly/react-core';

import { violationsBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { severityLabels } from 'messages/common';

import './SeverityTile.css';
import { severityColors } from 'constants/visuals/colors';
import { policySeverities, PolicySeverity } from 'types/policy.proto';

type SeverityTileProps = {
    severity: PolicySeverity;
    violationCount: number;
    link: string;
};

function SeverityTile({ severity, violationCount, link }: SeverityTileProps) {
    return (
        <Stack
            style={{ '--pf-severity-tile-color': severityColors[severity] } as CSSProperties}
            className="pf-severity-tile pf-u-p-md pf-u-align-items-center"
        >
            <StackItem className="pf-u-font-weight-bold pf-u-font-size-xl">
                {violationCount}
            </StackItem>
            <StackItem>
                <Link to={link}>{severityLabels[severity]}</Link>
            </StackItem>
        </Stack>
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
                    style={{ flexBasis: '0px' }}
                    shrink={{ default: 'shrink' }}
                    grow={{ default: 'grow' }}
                >
                    <SeverityTile
                        key={severity}
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
