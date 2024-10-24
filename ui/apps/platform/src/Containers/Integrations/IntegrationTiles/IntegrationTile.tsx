import React, { ReactElement, CSSProperties } from 'react';
import {
    Badge,
    Card,
    CardFooter,
    CardHeader,
    CardTitle,
    Flex,
    GalleryItem,
    Truncate,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';

type IntegrationTileProps = {
    categories?: string;
    image: string;
    label: string;
    linkTo: string;
    numIntegrations: number;
    isTechPreview?: boolean;
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
    isTechPreview = false,
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
                        <>
                            <img src={image} alt="" style={{ height: '100px' }} />
                        </>
                    </CardHeader>
                    <CardTitle className="pf-v5-u-color-100" style={{ whiteSpace: 'nowrap' }}>
                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                            <Truncate position="middle" content={label} />
                            {isTechPreview && <TechPreviewLabel />}
                        </Flex>
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
