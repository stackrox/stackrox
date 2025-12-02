import { Link } from 'react-router-dom-v5-compat';
import find from 'lodash/find';
import { Tooltip } from '@patternfly/react-core';
import { CheckIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import { riskBasePath } from 'routePaths';

type DeploymentNameColumnProps = {
    original: {
        deployment: { id: string; name: string };
        baselineStatuses: { anomalousProcessesExecuted: boolean }[];
    };
};

export function DeploymentNameColumn({ original }: DeploymentNameColumnProps) {
    const isSuspicious = find(original.baselineStatuses, {
        anomalousProcessesExecuted: true,
    });
    // Borrow layout from IconText component.
    return (
        <div className="flex items-center">
            <span className="pf-v6-u-display-inline-flex pf-v6-u-align-items-center">
                {isSuspicious ? (
                    <Tooltip content="Abnormal processes discovered">
                        <ExclamationCircleIcon
                            color="var(--pf-t--temp--dev--tbd)" /* CODEMODS: original v5 color was --pf-v5-global--danger-color--100 */
                        />
                    </Tooltip>
                ) : (
                    <Tooltip content="No abnormal processes discovered">
                        <CheckIcon />
                    </Tooltip>
                )}
                <span className="pf-v6-u-pl-sm pf-v6-u-text-nowrap">
                    <Link to={`${riskBasePath}/${original.deployment.id}`}>
                        {original.deployment.name}
                    </Link>
                </span>
            </span>
        </div>
    );
}
