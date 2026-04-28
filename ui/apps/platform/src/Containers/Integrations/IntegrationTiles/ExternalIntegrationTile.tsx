import type { ComponentType, ReactElement, SVGProps } from 'react';
import { Card, CardHeader, CardTitle, GalleryItem, Icon, Truncate } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

type ExternalIntegrationTileProps = {
    Logo: ComponentType<SVGProps<SVGSVGElement>>;
    label: string;
    url: string;
};

/**
 * Tile for external integrations that open a URL in a new browser tab
 * instead of navigating to an internal page.
 */
function ExternalIntegrationTile({ Logo, label, url }: ExternalIntegrationTileProps): ReactElement {
    return (
        <GalleryItem>
            <Card isClickable isCompact data-testid="external-integration-tile">
                <CardHeader
                    selectableActions={{
                        to: url,
                        isExternalLink: true,
                        selectableActionAriaLabel: `Open ${label} in a new tab`,
                    }}
                    className="pf-v6-u-mb-lg"
                >
                    <Logo
                        aria-label={`${label} logo`}
                        role="img"
                        style={{ height: '100px', width: 'auto', maxWidth: '100%' }}
                    />
                </CardHeader>
                <CardTitle className="pf-v6-u-color-100" style={{ whiteSpace: 'nowrap' }}>
                    <Truncate position="middle" content={label} />
                    <Icon size="sm" className="pf-v6-u-ml-sm">
                        <ExternalLinkAltIcon />
                    </Icon>
                </CardTitle>
            </Card>
        </GalleryItem>
    );
}

export default ExternalIntegrationTile;
