import React, { ReactElement } from 'react';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import helm from 'images/helm.svg';

function HelmIndicator(): ReactElement {
    return (
        <Tooltip content={<TooltipOverlay>This cluster is managed by Helm.</TooltipOverlay>}>
            <span className="w-5 h-5">
                <img className="w-5 h-5" src={helm} alt="Managed by Helm" />
            </span>
        </Tooltip>
    );
}

export default HelmIndicator;
