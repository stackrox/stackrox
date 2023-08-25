import React, { ReactElement, CSSProperties } from 'react';
import {
    Badge,
    Card,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    CardFooter,
    CardActions,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';

type IntegrationType = {
    label: string;
    image: string;
    categories: string;
};

type IntegrationTileProps = {
    integration: IntegrationType;
    numIntegrations: number;
    linkTo: string;
};

const styleCard = {
    cursor: 'pointer',
} as CSSProperties;

function IntegrationTile({
    integration,
    numIntegrations = 0,
    linkTo,
}: IntegrationTileProps): ReactElement {
    const { image, label, categories } = integration;

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
                {categories !== '' && categories !== undefined && (
                    <CardFooter className="pf-u-color-200">{categories}</CardFooter>
                )}
            </Card>
        </Link>
    );
}

export default IntegrationTile;
