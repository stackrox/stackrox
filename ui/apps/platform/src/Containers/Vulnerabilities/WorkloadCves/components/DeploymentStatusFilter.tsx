import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { deploymentStatuses } from 'types/deploymentStatus';

type DeploymentStatusFilterProps = {
    onChange?: () => void;
};

export default function DeploymentStatusFilter({ onChange }: DeploymentStatusFilterProps) {
    const [status, setStatus] = useURLStringUnion('deploymentStatus', deploymentStatuses);

    return (
        <ToggleGroup aria-label="Deployment status">
            <ToggleGroupItem
                buttonId="deployment-status-deployed"
                text="Deployed"
                isSelected={status === 'DEPLOYED'}
                onChange={() => {
                    setStatus('DEPLOYED');
                    onChange?.();
                }}
            />
            <ToggleGroupItem
                buttonId="deployment-status-deleted"
                text="Deleted"
                isSelected={status === 'DELETED'}
                onChange={() => {
                    setStatus('DELETED');
                    onChange?.();
                }}
            />
        </ToggleGroup>
    );
}
