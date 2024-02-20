import React, { ReactElement, useState } from 'react';
import {
    Alert,
    Button,
    Divider,
    Flex,
    FlexItem,
    Modal,
    Tab,
    TabTitleText,
    Tabs,
} from '@patternfly/react-core';

import SecureClusterUsingHelmChart from './SecureClusterUsingHelmChart';
import SecureClusterUsingOperator from './SecureClusterUsingOperator';

type TabKey = 'Operator' | 'Helm';

const headingLevel = 'h2';

export type SecureClusterModalProps = {
    isModalOpen: boolean;
    setIsModalOpen: (isOpen: boolean) => void;
};

function SecureClusterModal({ isModalOpen, setIsModalOpen }): ReactElement {
    const [activeKey, setActiveKey] = useState<TabKey>('Operator');

    function onClose() {
        setIsModalOpen(false);
    }

    function onSelect(_event, tabKey) {
        setActiveKey(tabKey);
    }

    return (
        <Modal
            variant="medium"
            title="Review installation methods"
            isOpen={isModalOpen}
            onClose={onClose}
            actions={[
                <Button key="Close" variant="primary" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Tabs activeKey={activeKey} onSelect={onSelect}>
                        <Tab eventKey="Operator" title={<TabTitleText>Operator</TabTitleText>} />
                        <Tab eventKey="Helm" title={<TabTitleText>Helm chart</TabTitleText>} />
                    </Tabs>
                    <Divider component="div" />
                </FlexItem>
                {activeKey === 'Operator' ? (
                    <SecureClusterUsingOperator headingLevel={headingLevel} />
                ) : (
                    <SecureClusterUsingHelmChart headingLevel={headingLevel} />
                )}
                <Alert
                    variant="info"
                    isInline
                    title="You can use one bundle to secure multiple clusters that have the same installation method."
                    component="p"
                />
            </Flex>
        </Modal>
    );
}

export default SecureClusterModal;
