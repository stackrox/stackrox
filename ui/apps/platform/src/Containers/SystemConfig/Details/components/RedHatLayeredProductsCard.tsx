import React from 'react';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Divider,
    Modal,
    Stack,
    StackItem,
    Text,
} from '@patternfly/react-core';

import useModal from 'hooks/useModal';
import { PlatformComponentRule } from 'types/config.proto';

export type RedHatLayeredProductsCardProps = {
    rule: PlatformComponentRule | undefined;
};

function RedHatLayeredProductsCard({ rule }: RedHatLayeredProductsCardProps) {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Card isFlat>
                <CardTitle>Red Hat layered products</CardTitle>
                <CardBody>
                    <Stack hasGutter>
                        <Text>
                            Components found in Red Hat layered and partner product namespaces are
                            included in the platform definition by default.
                        </Text>
                        <Divider component="div" />
                        <Text component="small" className="pf-v5-u-color-200">
                            Namespaces match (Regex)
                        </Text>
                        <CodeBlock>
                            <div className="truncate-multiline">{rule?.namespaceRule.regex}</div>
                        </CodeBlock>
                        <StackItem className="pf-v5-u-text-align-center pf-v5-u-mt-sm">
                            <Button variant="link" isInline onClick={openModal}>
                                View more
                            </Button>
                        </StackItem>
                    </Stack>
                </CardBody>
            </Card>
            <Modal
                variant="small"
                title="All Red Hat layered products"
                description="View all namespace matches (Regex) in Red Hat layered products"
                isOpen={isModalOpen}
                onClose={closeModal}
            >
                <CodeBlock>{rule?.namespaceRule.regex}</CodeBlock>
            </Modal>
        </>
    );
}

export default RedHatLayeredProductsCard;
