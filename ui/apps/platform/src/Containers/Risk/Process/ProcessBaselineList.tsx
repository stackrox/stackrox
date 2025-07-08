import React from 'react';
import { Flex, FlexItem, Switch, Tooltip } from '@patternfly/react-core';
import { LockIcon, LockOpenIcon } from '@patternfly/react-icons';

import usePermissions from 'hooks/usePermissions';
import { lockUnlockProcessBaselines } from 'services/ProcessBaselineService';
import type { ProcessBaseline, ProcessBaselineElement } from 'types/processBaseline.proto';

import ProcessBaselineElementList from './ProcessBaselineElementList';

const lockTooltipText =
    'Locking a container specification process baseline will trigger a violation if an abnormal process is found';
const unlockTooltipText =
    'Unlocking a container specification process baseline will NOT trigger a violation if an abnormal process is found';

export type ProcessBaselineListProps = {
    process: ProcessBaseline;
    processEpoch: number;
    setProcessEpoch: (number) => void;
};
function ProcessBaselineList({ process, processEpoch, setProcessEpoch }) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForDeploymentExtension = hasReadWriteAccess('DeploymentExtension');

    const isLocked = !!process.userLockedTimestamp;
    const { key, elements } = process;
    function toggleCurrentProcessLock() {
        const desiredLocked = !isLocked;
        const query = {
            keys: [{ ...key }],
            locked: desiredLocked,
        };
        return lockUnlockProcessBaselines(query).then(() => {
            // This is so that the parent component knows that one of the child components
            // modified the state server side and knows to re-render. Updating the processEpoch
            // value is just a way of causing the parent to reload the data from the server
            // and rerender all of the children.
            setProcessEpoch(processEpoch + 1);
        });
        // TODO catch finally?
    }

    const sortedElements = elements.sort((a: ProcessBaselineElement, b: ProcessBaselineElement) => {
        if (!a.element || !a.element.processName) {
            return -1;
        }
        if (!b.element || !b.element.processName) {
            return 1;
        }
        return a.element.processName.localeCompare(b.element.processName);
    });
    return (
        <li className="bg-base-100 text-base-600 rounded border border-base-400">
            <div className="text-base-600 flex justify-between items-center border-b border-base-300 p-3">
                <span className="font-700">{key.containerName}</span>
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsMd' }}
                >
                    <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                        <FlexItem>{isLocked ? 'Locked' : 'Unlocked'}</FlexItem>
                        <FlexItem>{isLocked ? <LockIcon /> : <LockOpenIcon />}</FlexItem>
                    </Flex>
                    {hasWriteAccessForDeploymentExtension && (
                        <Tooltip content={isLocked ? unlockTooltipText : lockTooltipText}>
                            <Switch
                                aria-label="Lock or unlock process baseline"
                                isChecked={isLocked}
                                onChange={toggleCurrentProcessLock}
                            />
                        </Tooltip>
                    )}
                </Flex>
            </div>
            <ProcessBaselineElementList
                baselineKey={key}
                elements={sortedElements}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        </li>
    );
}

export default ProcessBaselineList;
