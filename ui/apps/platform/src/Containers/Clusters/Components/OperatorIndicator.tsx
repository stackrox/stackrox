import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import operatorLogo from 'images/operator-logo.png';

function OperatorIndicator(): ReactElement {
    return (
        <Tooltip content="This cluster is managed by a Kubernetes Operator.">
            <span className="w-5 h-5 inline-block pf-v5-u-flex-shrink-0">
                <img
                    className="w-5 h-5"
                    src={operatorLogo}
                    alt="Managed by a Kubernetes Operator"
                />
            </span>
        </Tooltip>
    );
}

export default OperatorIndicator;
