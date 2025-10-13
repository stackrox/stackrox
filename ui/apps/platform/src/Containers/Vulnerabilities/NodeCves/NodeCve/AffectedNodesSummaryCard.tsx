import React from 'react';
import { Card, CardBody, CardTitle, Grid, GridItem, pluralize } from '@patternfly/react-core';

export type AffectedNodesSummaryCardProps = {
    affectedNodeCount: number;
    totalNodeCount: number;
    operatingSystemCount: number;
};

function AffectedNodesSummaryCard({
    affectedNodeCount,
    totalNodeCount,
    operatingSystemCount,
}: AffectedNodesSummaryCardProps) {
    return (
        <Card isCompact isFlat isFullHeight>
            <CardTitle>Affected nodes</CardTitle>
            <CardBody>
                <Grid>
                    <GridItem span={12} className="pf-v5-u-pt-sm">
                        {affectedNodeCount} / {totalNodeCount} affected nodes
                    </GridItem>
                    <GridItem span={12} className="pf-v5-u-pt-sm">
                        {pluralize(operatingSystemCount, 'operating system')} affected
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
}

export default AffectedNodesSummaryCard;
