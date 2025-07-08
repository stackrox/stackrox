import React from 'react';
import { Button, Tooltip } from '@patternfly/react-core';
import {
    AngleDownIcon,
    AngleUpIcon,
    ExclamationCircleIcon,
    PlusIcon,
} from '@patternfly/react-icons';

import usePermissions from 'hooks/usePermissions';
import { addProcessesToBaseline } from 'services/ProcessBaselineService';
import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';

import { getClusterIdAndNamespaceFromProcessGroup } from './process.utils';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer flex justify-between h-14';
const headerClassName = 'hover:bg-primary-200 hover:border-primary-300';

export type ProcessDiscoveryCardHeaderProps = {
    isExpanded: boolean;
    deploymentId: string;
    process: ProcessNameAndContainerNameGroup;
    processEpoch: number;
    setProcessEpoch: (number) => void;
};

function ProcessDiscoveryCardHeader({
    isExpanded,
    deploymentId,
    process,
    processEpoch,
    setProcessEpoch,
}: ProcessDiscoveryCardHeaderProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForDeploymentExtension = hasReadWriteAccess('DeploymentExtension');

    const { name, containerName, suspicious } = process;

    function addBaseline(evt) {
        evt.stopPropagation();
        const { clusterId, namespace } = getClusterIdAndNamespaceFromProcessGroup(process);
        const addProcessesQuery = {
            keys: [{ deploymentId, containerName, clusterId, namespace }],
            addElements: [{ processName: name }],
        };
        return addProcessesToBaseline(addProcessesQuery).then(() => {
            // This is so that the parent component knows that one of the child components
            // modified the state server side and knows to re-render. Updating the processEpoch
            // value is just a way of causing the parent to reload the data from the server
            // and rerender all of the children.
            setProcessEpoch(processEpoch + 1);
        });
        // TODO catch finally?
    }

    const trimmedName = name.length > 48 ? `${name.substring(0, 48)}...` : name;
    const style = suspicious ? { backgroundColor: 'var(--pf-v5-global--palette--red-50)' } : {};
    return (
        <div className={`${titleClassName} ${headerClassName}`} style={style}>
            <div className="p-3 text-base-600 flex flex-col">
                <div className="font-700">
                    {trimmedName}
                    {suspicious && (
                        <ExclamationCircleIcon
                            className="ml-4"
                            color="var(--pf-v5-global--danger-color--100)"
                        />
                    )}
                </div>
                <div className="text-sm">{`in container ${containerName} `}</div>
            </div>
            <div className="flex content-center">
                {hasWriteAccessForDeploymentExtension && suspicious && (
                    <div className="border-l border-r flex items-center justify-center w-16">
                        <Tooltip content="Add process to baseline">
                            <Button
                                variant="control"
                                aria-label="Add process to baseline"
                                icon={<PlusIcon />}
                                onClick={addBaseline}
                            />
                        </Tooltip>
                    </div>
                )}
                <button type="button" className="pl-3 pr-3" aria-label="Expand or Collapse">
                    {isExpanded ? <AngleUpIcon /> : <AngleDownIcon />}
                </button>
            </div>
        </div>
    );
}

export default ProcessDiscoveryCardHeader;
