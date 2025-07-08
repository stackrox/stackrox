import React from 'react';
import { Button, Tooltip } from '@patternfly/react-core';
import { MinusIcon } from '@patternfly/react-icons';

import usePermissions from 'hooks/usePermissions';
import { removeProcessesFromBaseline } from 'services/ProcessBaselineService';
import type { ProcessBaselineElement, ProcessBaselineKey } from 'types/processBaseline.proto';

export type ProcessBaselineElementListProps = {
    baselineKey: ProcessBaselineKey;
    elements: ProcessBaselineElement[];
    processEpoch: number;
    setProcessEpoch: (number) => void;
};

const ProcessBaselineElementList = ({
    baselineKey,
    elements,
    processEpoch,
    setProcessEpoch,
}: ProcessBaselineElementListProps) => {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForDeploymentExtension = hasReadWriteAccess('DeploymentExtension');

    if (!elements || !elements.length) {
        return <span className="p-3 block"> No elements in this baseline </span>;
    }

    function deleteCurrentProcess(element) {
        return () => {
            const query = {
                keys: [{ ...baselineKey }],
                removeElements: [element],
            };
            return removeProcessesFromBaseline(query).then(() => {
                setProcessEpoch(processEpoch + 1);
            });
            // TODO catch finally?
        };
    }

    return (
        <ul className="pl-3 pr-3">
            {elements.map(({ element }) => (
                <li
                    key={element.processName}
                    className="py-3 pb-2 leading-normal border-b border-base-300 flex justify-between items-center"
                >
                    <span>{element.processName}</span>
                    {hasWriteAccessForDeploymentExtension && (
                        <Tooltip content="Remove process from baseline">
                            <Button
                                variant="control"
                                aria-label="Remove process from baseline"
                                icon={<MinusIcon />}
                                onClick={deleteCurrentProcess(element)}
                            />
                        </Tooltip>
                    )}
                </li>
            ))}
        </ul>
    );
};

export default ProcessBaselineElementList;
