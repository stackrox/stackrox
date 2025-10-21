import { DropdownItem, Split, SplitItem } from '@patternfly/react-core';
import type { DropEvent } from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import UploadYAMLButton from './UploadYAMLButton';

type NetworkSimulatorActionsProps = {
    generateNetworkPolicies: () => void;
    undoNetworkPolicies: () => void;
    onFileInputChange: (_event: DropEvent, file: File) => void;
    openNotifyYAMLModal?: () => void;
};

function NetworkSimulatorActions({
    generateNetworkPolicies,
    undoNetworkPolicies,
    onFileInputChange,
    openNotifyYAMLModal,
}: NetworkSimulatorActionsProps) {
    return (
        <Split hasGutter className="pf-v5-u-p-md">
            <SplitItem>
                <UploadYAMLButton onFileInputChange={onFileInputChange} />
            </SplitItem>
            <SplitItem>
                <MenuDropdown
                    popperProps={{
                        direction: 'up',
                        position: 'start',
                    }}
                    toggleText="Actions"
                >
                    {openNotifyYAMLModal && (
                        <DropdownItem key="notify" onClick={openNotifyYAMLModal}>
                            Share YAML with notifiers
                        </DropdownItem>
                    )}
                    <DropdownItem key="generate" onClick={generateNetworkPolicies}>
                        Rebuild rules from active traffic
                    </DropdownItem>
                    <DropdownItem key="undo" onClick={undoNetworkPolicies}>
                        Revert rules to previously applied YAML
                    </DropdownItem>
                </MenuDropdown>
            </SplitItem>
        </Split>
    );
}

export default NetworkSimulatorActions;
