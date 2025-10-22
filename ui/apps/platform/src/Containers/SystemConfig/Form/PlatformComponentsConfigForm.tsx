import React, { useState } from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    FormGroup,
    FormSection,
    Grid,
    Modal,
    Stack,
    Tab,
    Tabs,
    TabTitleText,
    Text,
    TextArea,
    TextInput,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { PlusCircleIcon, RedoAltIcon, TrashIcon } from '@patternfly/react-icons';
import type { FormikErrors } from 'formik';
import type { Values } from './formTypes';

export type PlatformComponentsConfigFormProps = {
    values: Values;
    onChange: (value: unknown, event: unknown) => Promise<void> | Promise<FormikErrors<Values>>;
    onCustomChange: (value: unknown, id: unknown) => Promise<void> | Promise<FormikErrors<Values>>;
    defaultRedHatLayeredProductsRule: string;
};

function PlatformComponentsConfigForm({
    values,
    onChange,
    onCustomChange,
    defaultRedHatLayeredProductsRule,
}: PlatformComponentsConfigFormProps) {
    const [activeTabKey, setActiveTabKey] = useState<string | number>(0);
    const [isModalOpen, setIsModalOpen] = useState(false);

    function toggleModal() {
        setIsModalOpen((value) => !value);
    }

    const handleTabClick = (_event, tabIndex: string | number) => {
        setActiveTabKey(tabIndex);
    };

    return (
        <>
            <FormSection>
                <Title headingLevel="h2">Platform components configuration</Title>
                <Card isFlat>
                    <Tabs
                        activeKey={activeTabKey}
                        onSelect={handleTabClick}
                        aria-label="Platform components configuration tabs"
                        role="region"
                    >
                        <Tab eventKey={0} title={<TabTitleText>Core system</TabTitleText>}>
                            <div className="pf-v5-u-p-md">
                                <Title headingLevel="h3">Core system components</Title>
                                <Text>
                                    Core system components are not customizable and are set by the
                                    system. These definitions may change over time as the system is
                                    upgraded.
                                </Text>
                                <Divider component="div" className="pf-v5-u-py-md" />
                                <FormGroup
                                    label="Namespace rules (Regex)"
                                    fieldId="platformComponentConfigRules.coreSystemRule.namespaceRule.regex"
                                >
                                    <TextArea
                                        isDisabled
                                        type="text"
                                        id="platformComponentConfigRules.coreSystemRule.namespaceRule.regex"
                                        name="platformComponentConfigRules.coreSystemRule.namespaceRule.regex"
                                        value={
                                            values?.platformComponentConfigRules?.coreSystemRule
                                                .namespaceRule.regex
                                        }
                                        onChange={(event, value) => onChange(value, event)}
                                    />
                                </FormGroup>
                            </div>
                        </Tab>
                        <Tab
                            eventKey={1}
                            title={<TabTitleText>Red Hat layered products</TabTitleText>}
                        >
                            <div className="pf-v5-u-p-md">
                                <Title headingLevel="h3">Red Hat layered products</Title>
                                <Text>
                                    Components found in Red Hat layered and partner product
                                    namespaces are included in the platform definition by default.
                                    Enter one or more namespaces using regex, separated by | (pipe
                                    symbol). For more information on the syntax structure, see{' '}
                                    <Button
                                        variant="link"
                                        component="a"
                                        href="https://github.com/google/re2/wiki/syntax"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        isInline
                                    >
                                        RE2 syntax reference
                                    </Button>
                                    .
                                </Text>
                                <Alert
                                    variant="info"
                                    component="p"
                                    isInline
                                    title="All customization will be preserved after a central upgrade."
                                    className="pf-v5-u-mt-md"
                                >
                                    Use the reset button to revert all changes back to the default
                                    definition.
                                </Alert>
                                <Divider component="div" className="pf-v5-u-py-md" />
                                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                    <FlexItem flex={{ default: 'flex_1' }}>
                                        <FormGroup
                                            isRequired
                                            label="Namespace rules (Regex)"
                                            fieldId="platformComponentConfigRules.redHatLayeredProductsRule.namespaceRule.regex"
                                        >
                                            <TextArea
                                                isRequired
                                                type="text"
                                                id="platformComponentConfigRules.redHatLayeredProductsRule.namespaceRule.regex"
                                                name="platformComponentConfigRules.redHatLayeredProductsRule.namespaceRule.regex"
                                                value={
                                                    values?.platformComponentConfigRules
                                                        ?.redHatLayeredProductsRule.namespaceRule
                                                        .regex
                                                }
                                                onChange={(event, value) => onChange(value, event)}
                                            />
                                        </FormGroup>
                                    </FlexItem>
                                    <FlexItem>
                                        <Tooltip content={<div>Reset to default definition</div>}>
                                            <Button
                                                variant="plain"
                                                aria-label="Reset to default definition"
                                                onClick={toggleModal}
                                            >
                                                <RedoAltIcon />
                                            </Button>
                                        </Tooltip>
                                    </FlexItem>
                                </Flex>
                            </div>
                        </Tab>
                        <Tab eventKey={2} title={<TabTitleText>Custom components</TabTitleText>}>
                            <div className="pf-v5-u-p-md">
                                <Title headingLevel="h3">Custom platform components</Title>
                                <Text>
                                    Extend the platform definition by defining namespaces for
                                    additional applications and products. Enter one or more
                                    namespaces using regex, separated by | (pipe symbol). For more
                                    information on the syntax structure, see{' '}
                                    <Button
                                        variant="link"
                                        component="a"
                                        href="https://github.com/google/re2/wiki/syntax"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        isInline
                                    >
                                        RE2 syntax reference
                                    </Button>
                                    .
                                </Text>
                                <Divider component="div" className="pf-v5-u-py-md" />
                                <Grid hasGutter md={6}>
                                    {values.platformComponentConfigRules.customRules.map(
                                        (_customRule, index) => {
                                            const headerActions = (
                                                <Button
                                                    variant="plain"
                                                    aria-label="Remove custom component"
                                                    onClick={() => {
                                                        const newCustomRules =
                                                            values.platformComponentConfigRules.customRules.filter(
                                                                (_, i) => i !== index
                                                            );
                                                        return onCustomChange(
                                                            newCustomRules,
                                                            'platformComponentConfigRules.customRules'
                                                        );
                                                    }}
                                                >
                                                    <TrashIcon />
                                                </Button>
                                            );
                                            return (
                                                // @TODO: Consider a more appropriate way to handle the keys
                                                // eslint-disable-next-line react/no-array-index-key
                                                <Card key={index}>
                                                    <CardHeader
                                                        actions={{ actions: headerActions }}
                                                    >
                                                        <CardTitle>
                                                            Custom component {index + 1}
                                                        </CardTitle>
                                                    </CardHeader>
                                                    <CardBody>
                                                        <Stack hasGutter>
                                                            <FormGroup
                                                                label="Name"
                                                                isRequired
                                                                fieldId={`platformComponentConfigRules.customRules[${index}].name`}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    id={`platformComponentConfigRules.customRules[${index}].name`}
                                                                    name={`platformComponentConfigRules.customRules[${index}].name`}
                                                                    value={
                                                                        values
                                                                            ?.platformComponentConfigRules
                                                                            ?.customRules?.[index]
                                                                            .name
                                                                    }
                                                                    onChange={(event, value) =>
                                                                        onChange(value, event)
                                                                    }
                                                                />
                                                            </FormGroup>
                                                            <FormGroup
                                                                label="Namespace rules (Regex)"
                                                                isRequired
                                                                fieldId={`platformComponentConfigRules.customRules[${index}].namespaceRule.regex`}
                                                            >
                                                                <TextArea
                                                                    isRequired
                                                                    type="text"
                                                                    id={`platformComponentConfigRules.customRules[${index}].namespaceRule.regex`}
                                                                    name={`platformComponentConfigRules.customRules[${index}].namespaceRule.regex`}
                                                                    value={
                                                                        values
                                                                            ?.platformComponentConfigRules
                                                                            ?.customRules?.[index]
                                                                            ?.namespaceRule.regex
                                                                    }
                                                                    onChange={(event, value) =>
                                                                        onChange(value, event)
                                                                    }
                                                                />
                                                            </FormGroup>
                                                        </Stack>
                                                    </CardBody>
                                                </Card>
                                            );
                                        }
                                    )}
                                </Grid>
                                <Button
                                    variant="link"
                                    icon={<PlusCircleIcon />}
                                    onClick={() => {
                                        const currentCustomRules =
                                            values.platformComponentConfigRules.customRules;
                                        return onCustomChange(
                                            [
                                                ...currentCustomRules,
                                                { name: '', namespaceRule: { regex: '' } },
                                            ],
                                            'platformComponentConfigRules.customRules'
                                        );
                                    }}
                                    className={`${values.platformComponentConfigRules.customRules.length !== 0 && 'pf-v5-u-mt-md'}`}
                                >
                                    Add custom platform component
                                </Button>
                            </div>
                        </Tab>
                    </Tabs>
                </Card>
            </FormSection>
            <Modal
                variant="medium"
                titleIconVariant="warning"
                title="Reset Red Hat layered products to default?"
                isOpen={isModalOpen}
                onClose={toggleModal}
                actions={[
                    <Button
                        key="confirm"
                        variant="primary"
                        onClick={() => {
                            toggleModal();
                            return onCustomChange(
                                defaultRedHatLayeredProductsRule,
                                'platformComponentConfigRules.redHatLayeredProductsRule.namespaceRule.regex'
                            );
                        }}
                    >
                        Reset to default
                    </Button>,
                    <Button key="cancel" variant="link" onClick={toggleModal}>
                        Cancel
                    </Button>,
                ]}
            >
                This will overwrite your custom regex and restore the default Red Hat layered
                products regex. This action cannot be undone.
            </Modal>
        </>
    );
}

export default PlatformComponentsConfigForm;
