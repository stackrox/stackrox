import React, { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Button,
    Divider,
    Flex,
    FlexItem,
    Modal,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
} from '@patternfly/react-core';

import SecureClusterUsingHelmChart from './SecureClusterUsingHelmChart';
import SecureClusterUsingOperator from './SecureClusterUsingOperator';

type TabKey = 'Operator' | 'Helm';

const headingLevel = 'h2';

const idHelm = 'SecureClusterUsingHelm';
const idOperator = 'SecureClusterUsingOperator';

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
                        <Tab
                            eventKey="Operator"
                            tabContentId={idOperator}
                            title={<TabTitleText>Operator</TabTitleText>}
                        />
                        <Tab
                            eventKey="Helm"
                            tabContentId={idHelm}
                            title={<TabTitleText>Helm chart</TabTitleText>}
                        />
                    </Tabs>
                    <Divider component="div" />
                </FlexItem>
                {activeKey === 'Operator' ? (
                    <TabContent id={idOperator}>
                        <SecureClusterUsingOperator headingLevel={headingLevel} />
                    </TabContent>
                ) : (
                    <TabContent id={idHelm}>
                        <SecureClusterUsingHelmChart headingLevel={headingLevel} />
                    </TabContent>
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
