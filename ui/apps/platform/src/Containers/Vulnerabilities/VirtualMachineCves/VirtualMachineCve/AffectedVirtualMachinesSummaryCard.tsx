import { Card, CardBody, CardTitle, Grid, GridItem, pluralize } from '@patternfly/react-core';

type AffectedVirtualMachinesSummaryCardProps = {
    affectedVirtualMachinesCount: number;
    totalVirtualMachinesCount: number;
    affectedGuestOsCount: number;
};

function AffectedVirtualMachinesSummaryCard({
    affectedVirtualMachinesCount,
    totalVirtualMachinesCount,
    affectedGuestOsCount,
}: AffectedVirtualMachinesSummaryCardProps) {
    return (
        <Card isCompact isFullHeight>
            <CardTitle>Affected virtual machines</CardTitle>
            <CardBody>
                <Grid>
                    <GridItem span={12}>
                        {`${affectedVirtualMachinesCount} / ${totalVirtualMachinesCount} affected virtual machines`}
                    </GridItem>
                    <GridItem span={12}>
                        {pluralize(affectedGuestOsCount, 'Guest OS', 'Guest OSes')} affected
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
}

export default AffectedVirtualMachinesSummaryCard;
