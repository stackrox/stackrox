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
    Title,
    pluralize,
} from '@patternfly/react-core';
import { Modal } from '@patternfly/react-core/deprecated';

import type { PlatformComponentRule } from 'types/config.proto';

export type CustomPlatformComponentsCardProps = {
    customRules: PlatformComponentRule[];
};

function CustomPlatformComponentsCard({ customRules }: CustomPlatformComponentsCardProps) {
    const [isModalOpen, setIsModalOpen] = useState(false);

    function toggleModal() {
        setIsModalOpen((value) => !value);
    }

    return (
        <>
            <Card>
                <CardTitle>Custom components</CardTitle>
                <CardBody>
                    <Stack hasGutter>
                        <Content component="p">
                            Extend the platform definition by defining namespaces for additional
                            applications and products.
                        </Content>
                        <Divider component="div" />
                        <Content component="small" className="pf-v6-u-color-200">
                            Namespaces match (Regex)
                        </Content>
                        {customRules.length === 0 && <CodeBlock>None</CodeBlock>}
                        {customRules.length >= 1 && (
                            <CodeBlock>
                                <Content component="small" className="pf-v6-u-color-200">
                                    {customRules[0].name}
                                </Content>
                                <div className="truncate-multiline">
                                    {customRules[0].namespaceRule.regex}
                                </div>
                            </CodeBlock>
                        )}
                        {customRules.length > 1 && (
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
                title="All custom components"
                description="View all namespace matches (Regex) for custom components"
                isOpen={isModalOpen}
                onClose={toggleModal}
                tabIndex={0} // enables keyboard-accessible scrolling of a modal’s content
            >
                <Stack hasGutter>
                    <Title headingLevel="h2" className="pf-v6-u-color-100">
                        {pluralize(customRules.length, 'result')} found
                    </Title>
                    {customRules.map((rule) => {
                        return (
                            <CodeBlock key={rule.name}>
                                <Content component="small" className="pf-v6-u-color-200">
                                    {rule.name}
                                </Content>
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
