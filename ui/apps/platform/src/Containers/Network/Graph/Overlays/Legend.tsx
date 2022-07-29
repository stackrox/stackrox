import React, { useState, ReactElement } from 'react';
import {
    Button,
    ButtonVariant,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    List,
} from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import LegendTile from 'Containers/Network/Graph/Overlays/LegendTile';

function LegendContent(): ReactElement {
    return (
        <>
            <List isPlain data-testid="deployment-legend">
                <LegendTile name="deployment" description="Deployment" />
                <LegendTile
                    name="deployment-external-connections"
                    description="Deployment with active external connections"
                />
                <LegendTile
                    name="deployment-allowed-connections"
                    description="Deployment with allowed external connections"
                />
                <LegendTile
                    name="non-isolated-deployment-allowed"
                    description="Non-isolated deployment (all connections allowed)"
                />
            </List>
            <Divider component="div" className="pf-u-py-sm" />
            <List isPlain data-testid="namespace-legend">
                <LegendTile name="namespace" description="Namespace" />
                <LegendTile
                    name="namespace-allowed-connection"
                    description="Namespace with allowed external connections"
                />
                <LegendTile name="namespace-connection" description="Namespace connection" />
            </List>
            <Divider component="div" className="pf-u-py-sm" />
            <List isPlain data-testid="connection-legend">
                <LegendTile name="active-connection" description="Active connection" />
                <LegendTile name="allowed-connection" description="Allowed connection" />
                <LegendTile
                    name="namespace-egress-ingress"
                    description="Namespace external egress/ingress traffic"
                />
            </List>
        </>
    );
}

// Note, most of the utility styles related to text here can be removed when the app is fully
// migrated to PatternFly, or at the very least once the network graph has been moved to
// be under a top level <PageSection> component.
function Legend(): ReactElement {
    const [isOpen, toggleOpen] = useState(true);

    function toggleLegend() {
        toggleOpen(!isOpen);
    }

    function handleKeyUp(e) {
        return e.key === 'Enter' ? toggleLegend() : null;
    }

    return (
        <Card
            data-testid="legend"
            className="pf-u-mb-sm pf-u-ml-sm pf-u-color-100"
            style={{ position: 'absolute', bottom: 0, left: 0 }}
            isCompact
            isRounded
        >
            {isOpen && (
                <>
                    <CardHeader className="pf-u-justify-content-space-between">
                        <CardTitle className="pf-u-font-size-sm pf-u-font-family-heading-sans-serif">
                            Legend
                        </CardTitle>
                        <Button
                            aria-label="Close legend"
                            className="pf-u-p-xs"
                            isSmall
                            onClick={toggleLegend}
                            variant={ButtonVariant.plain}
                        >
                            <TimesIcon />
                        </Button>
                    </CardHeader>
                    <CardBody className="pf-u-font-family-sans-serif pf-u-color-200">
                        <LegendContent />
                    </CardBody>
                </>
            )}
            {!isOpen && (
                <Button
                    aria-label="Open legend"
                    isSmall
                    onClick={toggleLegend}
                    onKeyUp={handleKeyUp}
                    tabIndex={0}
                    variant={ButtonVariant.plain}
                    className="pf-u-font-family-heading-sans-serif pf-u-color-100"
                >
                    Legend
                </Button>
            )}
        </Card>
    );
}

export default Legend;
