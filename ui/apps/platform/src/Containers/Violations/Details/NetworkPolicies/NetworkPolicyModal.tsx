import React from 'react';
import { Button, Flex, Modal } from '@patternfly/react-core';

import { NetworkPolicy } from 'types/networkPolicy.proto';
import download from 'utils/download';
import CodeViewer from 'Components/CodeViewer';

export type NetworkPolicyModalProps = {
    networkPolicy: Pick<NetworkPolicy, 'name' | 'yaml'>;
    isOpen: boolean;
    onClose: () => void;
};

function NetworkPolicyModal({ networkPolicy, isOpen, onClose }: NetworkPolicyModalProps) {
    function exportYAMLHandler() {
        download(`${networkPolicy.name}.yml`, networkPolicy.yaml, 'yml');
    }

    return (
        <Modal
            title="Network policy details"
            variant="small"
            isOpen={isOpen}
            onClose={onClose}
            actions={[
                <Button className="pf-v5-u-display-inline-block" onClick={exportYAMLHandler}>
                    Export YAML
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <p>Policy name: {networkPolicy.name}</p>
                <CodeViewer
                    code={networkPolicy.yaml}
                    style={{
                        '--pf-v5-u-max-height--MaxHeight': '450px',
                    }}
                />
            </Flex>
        </Modal>
    );
}

export default NetworkPolicyModal;
