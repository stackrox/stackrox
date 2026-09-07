import type { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import operatorLogo from 'images/operator-logo.png';

function OperatorIndicator(): ReactElement {
    return (
        <Tooltip content="This cluster is managed by a Kubernetes Operator.">
            <span className="pf-v6-u-display-inline-block pf-v6-u-flex-shrink-0">
                <img
                    style={{ width: '20px', height: '20px' }}
                    src={operatorLogo}
                    alt="Managed by a Kubernetes Operator"
                />
            </span>
        </Tooltip>
    );
}

export default OperatorIndicator;
