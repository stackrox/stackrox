import type { ReactElement } from 'react';
import { Gallery } from '@patternfly/react-core';

import type { IntegrationsTabProps } from './IntegrationsTab.types';
import IntegrationsTabPage from './IntegrationsTabPage';

const source = 'apiClients';

function ApiClientIntegrationsTab({ sourcesEnabled }: IntegrationsTabProps): ReactElement {
    return (
        <IntegrationsTabPage source={source} sourcesEnabled={sourcesEnabled}>
            <Gallery hasGutter>{/* ServiceNow VR tile will be added in Slice 2 */}</Gallery>
        </IntegrationsTabPage>
    );
}

export default ApiClientIntegrationsTab;
