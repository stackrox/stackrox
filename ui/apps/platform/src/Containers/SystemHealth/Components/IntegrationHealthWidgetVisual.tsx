import React, { ReactElement } from 'react';
import { Message } from '@stackrox/ui-components';

import { integrationsPath } from 'routePaths';
import ViewAllButton from 'Components/ViewAllButton';
import Widget from 'Components/Widget';
import IntegrationsHealth from './IntegrationsHealth';
import { IntegrationMergedItem } from '../utils/integrations';

type IntegrationHealthWidgetProps = {
    id: string;
    integrationText: string;
    integrationsMerged: IntegrationMergedItem[];
    requestHasError: boolean;
};

const IntegrationHealthWidget = ({
    id,
    integrationText,
    integrationsMerged,
    requestHasError,
}: IntegrationHealthWidgetProps): ReactElement => {
    return (
        <Widget
            header={integrationText}
            headerComponents={<ViewAllButton url={`${integrationsPath}#${id}`} />}
            className="h-48 text-lg"
            id={id}
        >
            {requestHasError ? (
                <div className="p-2 w-full">
                    <Message type="error">Request failed for {integrationText}</Message>
                </div>
            ) : (
                <IntegrationsHealth integrationsMerged={integrationsMerged} />
            )}
        </Widget>
    );
};

export default IntegrationHealthWidget;
