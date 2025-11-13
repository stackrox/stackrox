import { Button, Content, Flex, Title } from '@patternfly/react-core';
import { Modal } from '@patternfly/react-core/deprecated';
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
                    <Content component="p">
                        Compare the generated network policies to the existing network policies.
                    </Content>
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
                className="pf-v6-u-mt-md"
                direction={{ default: 'column', md: 'row' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <Flex direction={{ default: 'column' }} style={{ flex: '1' }}>
                    <Content component="p" className="pf-v6-u-font-weight-bold">
                        Existing network policies
                    </Content>
                    <NetworkPoliciesYAML
                        yaml={current}
                        style={{ '--pf-v5-u-max-height--MaxHeight': '400px' }}
                    />
                </Flex>
                <Flex direction={{ default: 'column' }} style={{ flex: '1' }}>
                    <Content component="p" className="pf-v6-u-font-weight-bold">
                        Generated network policies
                    </Content>
                    <NetworkPoliciesYAML
                        yaml={generated}
                        style={{ '--pf-v5-u-max-height--MaxHeight': '400px' }}
                    />
                </Flex>
            </Flex>
        </Modal>
    );
}

export default CompareYAMLModal;
