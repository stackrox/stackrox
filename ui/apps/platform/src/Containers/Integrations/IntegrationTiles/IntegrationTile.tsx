import React, { ReactElement, CSSProperties } from 'react';
import {
    Badge,
    Card,
    CardActions,
    CardFooter,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    GalleryItem,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';

type IntegrationTileProps = {
    categories?: string;
    image: string;
    label: string;
    linkTo: string;
    numIntegrations: number;
};

const styleCard = {
    cursor: 'pointer',
} as CSSProperties;

function IntegrationTile({
    categories,
    image,
    label,
    linkTo,
    numIntegrations,
}: IntegrationTileProps): ReactElement {
    return (
        <GalleryItem>
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
                    {categories && <CardFooter className="pf-u-color-200">{categories}</CardFooter>}
                </Card>
            </Link>
        </GalleryItem>
    );
}

export default IntegrationTile;
