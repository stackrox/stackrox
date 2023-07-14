import React from 'react';
import { Button, Flex, Modal, Text, Title } from '@patternfly/react-core';
import NetworkPoliciesYAML from './NetworkPoliciesYAML';

export type CompareYAMLModalProps = {
    current: string;
    generated: string;
    isOpen: boolean;
    onClose: () => void;
};

function CompareYAMLModal({ current, generated, isOpen, onClose }: CompareYAMLModalProps) {
    return (
        <Modal
            className="compare-yaml-modal"
            isOpen={isOpen}
            header={
                <>
                    <Title headingLevel="h2">Compare with existing network policies</Title>
                    <Text>
                        Compare changes in the generated network policies to the existing network
                        policies.
                    </Text>
                </>
            }
            onClose={onClose}
            actions={[
                <Button key="close" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Flex
                className="pf-u-mt-md"
                direction={{ default: 'column', md: 'row' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <Flex direction={{ default: 'column' }} style={{ flex: '1' }}>
                    <Text className="pf-u-font-weight-bold">Generated network policies</Text>
                    <NetworkPoliciesYAML yaml={generated} height="400px" />
                </Flex>
                <Flex direction={{ default: 'column' }} style={{ flex: '1' }}>
                    <Text className="pf-u-font-weight-bold">Existing network policies</Text>
                    <NetworkPoliciesYAML yaml={current} height="400px" />
                </Flex>
            </Flex>
        </Modal>
    );
}

export default CompareYAMLModal;
