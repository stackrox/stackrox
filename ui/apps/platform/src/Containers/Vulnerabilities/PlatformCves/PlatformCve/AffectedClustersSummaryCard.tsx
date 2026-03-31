import { Card, CardBody, CardTitle, Grid, GridItem } from '@patternfly/react-core';

export type AffectedClustersSummaryCardProps = {
    affectedClusterCount: number;
    totalClusterCount: number;
};

function AffectedClustersSummaryCard({
    affectedClusterCount,
    totalClusterCount,
}: AffectedClustersSummaryCardProps) {
    return (
        <Card isCompact isFullHeight>
            <CardTitle>Affected clusters</CardTitle>
            <CardBody>
                <Grid>
                    <GridItem span={12} className="pf-v6-u-pt-sm">
                        {affectedClusterCount} / {totalClusterCount} affected clusters
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
}

export default AffectedClustersSummaryCard;
