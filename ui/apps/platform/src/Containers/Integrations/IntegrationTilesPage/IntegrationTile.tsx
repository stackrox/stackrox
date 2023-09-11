import React, { ReactElement, CSSProperties } from 'react';
import {
    Badge,
    Card,
    CardActions,
    CardFooter,
    CardHeader,
    CardHeaderMain,
    CardTitle,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { IntegrationDescriptor } from '../utils/integrationsList';

type IntegrationTileProps = {
    integration: IntegrationDescriptor;
    numIntegrations: number;
    linkTo: string;
};

const styleCard = {
    cursor: 'pointer',
} as CSSProperties;

function IntegrationTile({
    integration,
    numIntegrations,
    linkTo,
}: IntegrationTileProps): ReactElement {
    const { image, label } = integration;

    return (
        <Link to={linkTo} data-testid="integration-tile">
            <Card isSelectableRaised isCompact isFlat style={styleCard}>
                <CardHeader className="pf-u-mb-lg">
                    <CardHeaderMain>
                        <img src={image} alt="" style={{ height: '100px' }} />
                    </CardHeaderMain>
                    <CardActions>
                        {numIntegrations > 0 && <Badge>{numIntegrations}</Badge>}
                    </CardActions>
                </CardHeader>
                <CardTitle className="pf-u-color-100">{label}</CardTitle>
                {'categories' in integration && integration.categories && (
                    <CardFooter className="pf-u-color-200">{integration.categories}</CardFooter>
                )}
            </Card>
        </Link>
    );
}

export default IntegrationTile;
