import { useState } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Content,
    Divider,
    Stack,
    StackItem,
} from '@patternfly/react-core';
import { Modal } from '@patternfly/react-core/deprecated';

import type { PlatformComponentRule } from 'types/config.proto';

export type RedHatLayeredProductsCardProps = {
    rule: PlatformComponentRule | undefined;
};

function RedHatLayeredProductsCard({ rule }: RedHatLayeredProductsCardProps) {
    const [isModalOpen, setIsModalOpen] = useState(false);

    function toggleModal() {
        setIsModalOpen((value) => !value);
    }

    return (
        <>
            <Card>
                <CardTitle>Red Hat layered products</CardTitle>
                <CardBody>
                    <Stack hasGutter>
                        <Content component="p">
                            Components found in Red Hat layered and partner product namespaces are
                            included in the platform definition by default.
                        </Content>
                        <Divider component="div" />
                        <Content component="small" className="pf-v6-u-color-200">
                            Namespaces match (Regex)
                        </Content>
                        <CodeBlock>
                            <div className="truncate-multiline">
                                {rule?.namespaceRule?.regex || 'None'}
                            </div>
                        </CodeBlock>
                        {rule?.namespaceRule.regex !== '' && (
                            <StackItem className="pf-v6-u-text-align-center pf-v6-u-mt-sm">
                                <Button variant="link" isInline onClick={toggleModal}>
                                    View more
                                </Button>
                            </StackItem>
                        )}
                    </Stack>
                </CardBody>
            </Card>
            <Modal
                variant="small"
                title="All Red Hat layered products"
                description="View all namespace matches (Regex) in Red Hat layered products"
                isOpen={isModalOpen}
                onClose={toggleModal}
            >
                <CodeBlock>{rule?.namespaceRule.regex}</CodeBlock>
            </Modal>
        </>
    );
}

export default RedHatLayeredProductsCard;
