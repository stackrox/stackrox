import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import helm from 'images/helm.svg';

function HelmIndicator(): ReactElement {
    return (
        <Tooltip content="This cluster is managed by Helm.">
            <span className="w-5 h-5 inline-block pf-v5-u-flex-shrink-0">
                <img className="w-5 h-5" src={helm} alt="Managed by Helm" />
            </span>
        </Tooltip>
    );
}

export default HelmIndicator;
