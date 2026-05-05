import type { ReactElement } from 'react';
import {
    Card,
    CardFooter,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Gallery,
    GalleryItem,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

import ServiceNowLogo from 'images/servicenow.svg?react';

import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';

const source = 'apiClients';

const serviceNowStoreUrl =
    'https://store.servicenow.com/store/app/edea7344476072502ec7c1c4f16d4343';

function ApiClientIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>
                <GalleryItem>
                    <Card isClickable isCompact data-testid="integration-tile">
                        <CardHeader
                            selectableActions={{
                                to: serviceNowStoreUrl,
                                isExternalLink: true,
                                selectableActionAriaLabel:
                                    'View ServiceNow VR app in ServiceNow Store (opens in a new tab)',
                            }}
                            className="pf-v6-u-mb-lg"
                        >
                            <ServiceNowLogo
                                aria-label="ServiceNow VR logo"
                                role="img"
                                style={{ height: '100px', width: 'auto', maxWidth: '100%' }}
                            />
                        </CardHeader>
                        <CardTitle className="pf-v6-u-color-100" style={{ whiteSpace: 'nowrap' }}>
                            <Flex
                                alignItems={{ default: 'alignItemsBaseline' }}
                                gap={{ default: 'gapSm' }}
                            >
                                <FlexItem>ServiceNow VR</FlexItem>
                                <ExternalLinkAltIcon />
                            </Flex>
                        </CardTitle>
                        <CardFooter className="pf-v6-u-color-200">
                            Pull ACS vulnerability data into ServiceNow Vulnerability Response.
                        </CardFooter>
                    </Card>
                </GalleryItem>
            </Gallery>
        </IntegrationsTabPage>
    );
}

export default ApiClientIntegrationsTab;
