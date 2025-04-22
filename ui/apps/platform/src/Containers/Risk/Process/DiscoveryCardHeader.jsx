import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { Tooltip } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import { addDeleteProcesses } from 'services/ProcessesService';
import { getDeploymentAndProcessIdFromProcessGroup } from 'utils/processUtils';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer flex justify-between h-14';
const headerClassName = 'hover:bg-primary-200 hover:border-primary-300';

function DiscoveryCardHeader({ icon, deploymentId, process, processEpoch, setProcessEpoch }) {
    const { name, containerName, suspicious } = process;

    function addBaseline(evt) {
        evt.stopPropagation();
        const { clusterId, namespace } = getDeploymentAndProcessIdFromProcessGroup(process);
        const addProcessesQuery = {
            keys: [{ deploymentId, containerName, clusterId, namespace }],
            addElements: [{ processName: name }],
        };
        addDeleteProcesses(addProcessesQuery).then(() => {
            // This is so that the parent component knows that one of the child components
            // modified the state server side and knows to re-render. Updating the processEpoch
            // value is just a way of causing the parent to reload the data from the server
            // and rerender all of the children.
            setProcessEpoch(processEpoch + 1);
        });
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
                {suspicious && (
                    <div className="border-l border-r flex items-center justify-center w-16">
                        <Tooltip content="Add to baseline">
                            <button
                                type="button"
                                onClick={addBaseline}
                                className="border rounded p-px mr-3 ml-3 flex items-center"
                                aria-label="Add process to baseline"
                            >
                                <Icon.Plus className="h-4 w-4" />
                            </button>
                        </Tooltip>
                    </div>
                )}
                <button type="button" className="pl-3 pr-3" aria-label="Expand or Collapse">
                    {icon}
                </button>
            </div>
        </div>
    );
}

DiscoveryCardHeader.propTypes = {
    icon: PropTypes.node.isRequired,
    deploymentId: PropTypes.string.isRequired,
    process: PropTypes.shape({
        name: PropTypes.string.isRequired,
        containerName: PropTypes.string.isRequired,
        suspicious: PropTypes.bool.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
};

export default DiscoveryCardHeader;
