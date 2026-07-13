import { Card, CardBody, CardTitle, Grid, GridItem } from '@patternfly/react-core';
import pluralize from 'pluralize';

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
                        {`${affectedGuestOsCount} ${pluralize('Guest OS', affectedGuestOsCount)} affected`}
                    </GridItem>
                </Grid>
            </CardBody>
        </Card>
    );
}

export default AffectedVirtualMachinesSummaryCard;
