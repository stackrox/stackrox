import React, { useState } from 'react';
import { Button, Flex, Modal, ModalVariant } from '@patternfly/react-core';
import { CodeEditor, Language } from '@patternfly/react-code-editor';

import CodeEditorDarkModeControl from 'Components/PatternFly/CodeEditorDarkModeControl';
import { NetworkPolicy } from 'types/networkPolicy.proto';
import download from 'utils/download';

export type NetworkPolicyModalProps = {
    networkPolicy: Pick<NetworkPolicy, 'name' | 'yaml'>;
    isOpen: boolean;
    onClose: () => void;
};

function NetworkPolicyModal({ networkPolicy, isOpen, onClose }: NetworkPolicyModalProps) {
    const [isDarkMode, setIsDarkMode] = useState(false);

    function exportYAMLHandler() {
        download(`${networkPolicy.name}.yml`, networkPolicy.yaml, 'yml');
    }

    return (
        <Modal
            title="Network policy details"
            variant={ModalVariant.small}
            isOpen={isOpen}
            onClose={onClose}
            actions={[
                <Button className="pf-u-display-inline-block" onClick={exportYAMLHandler}>
                    Export YAML
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <p>Policy name: {networkPolicy.name}</p>
                <CodeEditor
                    isDarkTheme={isDarkMode}
                    customControls={
                        <CodeEditorDarkModeControl
                            isDarkMode={isDarkMode}
                            onToggleDarkMode={() => setIsDarkMode((wasDarkMode) => !wasDarkMode)}
                        />
                    }
                    isCopyEnabled
                    isLineNumbersVisible
                    isReadOnly
                    code={networkPolicy.yaml}
                    language={Language.yaml}
                    height="450px"
                />
            </Flex>
        </Modal>
    );
}

export default NetworkPolicyModal;
