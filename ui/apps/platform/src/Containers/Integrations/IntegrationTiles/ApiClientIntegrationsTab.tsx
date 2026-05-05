import type { ReactElement } from 'react';
import {
    Card,
    CardFooter,
    CardHeader,
    CardTitle,
    Gallery,
    GalleryItem,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
// TODO(ROX-34536): Replace with approved ServiceNow SVG logo
import ServiceNowLogo from 'images/redhat.svg?react';

import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';

const source = 'apiClients';

function ApiClientIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    const { version } = useMetadata();

    const docsUrl = getVersionedDocs(version, 'integrating/integrate-with-servicenow');

    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>
                <GalleryItem>
                    <Card isClickable isCompact data-testid="integration-tile">
                        <CardHeader
                            selectableActions={{
                                to: docsUrl,
                                isExternalLink: true,
                                selectableActionAriaLabel:
                                    'View ServiceNow VR documentation (opens in a new tab)',
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
                            ServiceNow VR{' '}
                            <ExternalLinkAltIcon color="var(--pf-t--global--text--color--link--default)" />
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
