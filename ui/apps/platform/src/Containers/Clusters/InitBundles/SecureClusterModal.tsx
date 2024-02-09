import React, { ReactElement, useState } from 'react';
import {
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
            variant="large"
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
            </Flex>
        </Modal>
    );
}

export default SecureClusterModal;
