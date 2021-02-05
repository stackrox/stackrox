import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';
import { Message } from '@stackrox/ui-components';

import { integrationsPath } from 'routePaths';
import Widget from 'Components/Widget';
import IntegrationsHealth from './IntegrationsHealth';

type IntegrationHealthWidgetProps = {
    smallButtonClassName: string;
    id: string;
    integrationText: string;
    integrationsMerged: [];
    requestHasError: boolean;
};

const IntegrationHealthWidget = ({
    smallButtonClassName,
    id,
    integrationText,
    integrationsMerged,
    requestHasError,
}: IntegrationHealthWidgetProps): ReactElement => {
    return (
        <Widget
            header={integrationText}
            headerComponents={
                <HashLink to={`${integrationsPath}#${id}`} className={smallButtonClassName}>
                    View All
                </HashLink>
            }
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
