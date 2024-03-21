import React, { ReactElement, CSSProperties } from 'react';
import {
    Badge,
    Card,
    CardFooter,
    CardHeader,
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
                <Card isSelectable isCompact isFlat style={styleCard}>
                    <CardHeader
                        actions={{
                            actions: <>{numIntegrations > 0 && <Badge>{numIntegrations}</Badge>}</>,
                            hasNoOffset: false,
                            className: undefined,
                        }}
                        className="pf-v5-u-mb-lg"
                    >
                        actions=
                        {
                            <>
                                <img src={image} alt="" style={{ height: '100px' }} />
                            </>
                        }
                    </CardHeader>
                    <CardTitle className="pf-v5-u-color-100" style={{ whiteSpace: 'nowrap' }}>
                        {label}
                    </CardTitle>
                    {categories && (
                        <CardFooter className="pf-v5-u-color-200">{categories}</CardFooter>
                    )}
                </Card>
            </Link>
        </GalleryItem>
    );
}

export default IntegrationTile;
