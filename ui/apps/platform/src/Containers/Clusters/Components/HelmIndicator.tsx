import type { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import HelmLogo from 'images/helm.svg?react';

function HelmIndicator(): ReactElement {
    return (
        <Tooltip content="This cluster is managed by Helm.">
            <span className="w-5 h-5 inline-block pf-v6-u-flex-shrink-0">
                <HelmLogo className="w-5 h-5" />
            </span>
        </Tooltip>
    );
}

export default HelmIndicator;
