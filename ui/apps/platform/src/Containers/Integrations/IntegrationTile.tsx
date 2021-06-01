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

type IntegrationType = {
    label: string;
    image: string;
    categories: string;
};

type IntegrationTileProps = {
    integration: IntegrationType;
    onClick: (IntegrationType) => void;
    numIntegrations: number;
};

const styleCard = {
    cursor: 'pointer',
} as CSSProperties;

function IntegrationTile({
    integration,
    onClick,
    numIntegrations = 0,
}: IntegrationTileProps): ReactElement {
    function onClickHandler() {
        return onClick(integration);
    }

    function handleKeyUp(e) {
        return e.key === 'Enter' ? onClick(integration) : null;
    }

    const { image, label, categories } = integration;

    return (
        <Card
            isHoverable
            isCompact
            isFlat
            onClick={onClickHandler}
            onKeyUp={handleKeyUp}
            style={styleCard}
            role="button"
        >
            <CardHeader className="pf-u-mb-lg">
                <CardHeaderMain>
                    <img src={image} alt={label} style={{ height: '100px' }} />
                </CardHeaderMain>
                <CardActions>{numIntegrations > 0 && <Badge>{numIntegrations}</Badge>}</CardActions>
            </CardHeader>
            <CardTitle className="pf-u-color-100">{label}</CardTitle>
            {categories !== '' && categories !== undefined && (
                <CardFooter className="pf-u-color-200">{categories}</CardFooter>
            )}
        </Card>
    );
}

export default IntegrationTile;
