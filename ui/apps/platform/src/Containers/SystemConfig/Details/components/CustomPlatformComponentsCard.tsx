import React from 'react';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Divider,
    Modal,
    pluralize,
    Stack,
    StackItem,
    Text,
    Title,
} from '@patternfly/react-core';

import useModal from 'hooks/useModal';
import { PlatformComponentRule } from 'types/config.proto';

export type CustomPlatformComponentsCardProps = {
    customRules: PlatformComponentRule[];
};

function CustomPlatformComponentsCard({ customRules }: CustomPlatformComponentsCardProps) {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Card isFlat>
                <CardTitle>Custom components</CardTitle>
                <CardBody>
                    <Stack hasGutter>
                        <Text>
                            Extend the platform definition by defining namespaces for additional
                            applications and products.
                        </Text>
                        <Divider component="div" />
                        <Text component="small" className="pf-v5-u-color-200">
                            Namespaces match (Regex)
                        </Text>
                        {customRules.length === 0 && <CodeBlock>None</CodeBlock>}
                        {customRules.length >= 1 && (
                            <CodeBlock>
                                <Text component="small" className="pf-v5-u-color-200">
                                    {customRules[0].name}
                                </Text>
                                <div className="truncate-multiline">
                                    {customRules[0].namespaceRule.regex}
                                </div>
                            </CodeBlock>
                        )}
                        {customRules.length > 1 && (
                            <StackItem className="pf-v5-u-text-align-center pf-v5-u-mt-sm">
                                <Button variant="link" isInline onClick={openModal}>
                                    View more
                                </Button>
                            </StackItem>
                        )}
                    </Stack>
                </CardBody>
            </Card>
            <Modal
                variant="small"
                title="All custom components"
                description="View all namespace matches (Regex) for custom components"
                isOpen={isModalOpen}
                onClose={closeModal}
                tabIndex={0} // enables keyboard-accessible scrolling of a modal’s content
            >
                <Stack hasGutter>
                    <Title headingLevel="h2" className="pf-v5-u-color-100">
                        {pluralize(customRules.length, 'result')} found
                    </Title>
                    {customRules.map((rule) => {
                        return (
                            <CodeBlock>
                                <Text component="small" className="pf-v5-u-color-200">
                                    {rule.name}
                                </Text>
                                <div>{rule.namespaceRule.regex}</div>
                            </CodeBlock>
                        );
                    })}
                </Stack>
            </Modal>
        </>
    );
}

export default CustomPlatformComponentsCard;
