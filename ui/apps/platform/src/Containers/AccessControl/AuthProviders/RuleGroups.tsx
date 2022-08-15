/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { FieldArray } from 'formik';
import { Button, Flex, FlexItem, FormGroup, SelectOption, TextInput } from '@patternfly/react-core';
import { ArrowRightIcon, PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';

import { Group } from 'services/AuthService';
import { Role } from 'services/RolesService';
import SelectSingle from 'Components/SelectSingle';

export type RuleGroupErrors = {
    roleName?: string;
    props?: {
        key?: string;
        value?: string;
        id?: string;
    };
};

export type RuleGroupsProps = {
    authProviderId: string;
    onChange: (
        _value: unknown,
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) => void;
    roles: Role[];
    groups: Group[];
    setFieldValue: (name: string, value: string) => void;
    disabled: boolean | undefined;
    errors?: RuleGroupErrors[];
    ruleAttributes?: string[];
};

function getAugmentedRuleKeys(ruleKeys, groups) {
    const newRuleKeys = [...ruleKeys];

    groups.forEach((group) => {
        const alreadyInList = newRuleKeys.find((key) => key === group?.props?.key);

        if (group.props.key && !alreadyInList) {
            newRuleKeys.push(group.props.key);
        }
    });

    return newRuleKeys as string[];
}

function RuleGroups({
    authProviderId,
    onChange,
    setFieldValue,
    groups = [],
    roles = [],
    disabled = false,
    errors = [],
    ruleAttributes = [],
}: RuleGroupsProps): ReactElement {
    const augmentedRuleKeys = getAugmentedRuleKeys(ruleAttributes, groups);

    function isDisabled(group: Group) {
        return disabled || group?.props?.traits?.mutabilityMode !== 'ALLOW_MUTATE';
    }

    return (
        <FieldArray
            name="groups"
            render={(arrayHelpers) => (
                <>
                    {groups.length === 0 && <p>No custom rules defined</p>}
                    {groups.length > 0 &&
                        groups.map((group, index: number) => (
                            <Flex key={`${group.props.authProviderId}_custom_rule_${index}`}>
                                <FlexItem>
                                    <FormGroup
                                        label="Key"
                                        fieldId={`groups[${index}].props.key`}
                                        helperTextInvalid={errors[index]?.props?.key ?? ''}
                                        validated={errors[index]?.props?.key ? 'error' : 'default'}
                                    >
                                        <SelectSingle
                                            id={`groups[${index}].props.key`}
                                            value={groups[`${index}`].props.key ?? ''}
                                            isDisabled={isDisabled(group)}
                                            handleSelect={setFieldValue}
                                            direction="up"
                                            isCreatable
                                            variant="typeahead"
                                            placeholderText="Select or enter a key"
                                        >
                                            {augmentedRuleKeys.map((ruleKey) => (
                                                <SelectOption key={ruleKey} value={ruleKey} />
                                            ))}
                                        </SelectSingle>
                                    </FormGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormGroup
                                        label="Value"
                                        fieldId={`groups[${index}].props.value`}
                                        helperTextInvalid={errors[index]?.props?.value || ''}
                                        validated={
                                            errors[index]?.props?.value ? 'error' : 'default'
                                        }
                                    >
                                        <TextInput
                                            type="text"
                                            id={`groups[${index}].props.value`}
                                            value={groups[`${index}`].props.value}
                                            onChange={onChange}
                                            isDisabled={isDisabled(group)}
                                        />
                                    </FormGroup>
                                </FlexItem>
                                <FlexItem>
                                    <ArrowRightIcon style={{ transform: 'translate(0, 42px)' }} />
                                </FlexItem>
                                <FlexItem>
                                    <FormGroup
                                        label="Role"
                                        fieldId={`groups[${index}].roleName`}
                                        helperTextInvalid={errors[index]?.roleName || ''}
                                        validated={errors[index]?.roleName ? 'error' : 'default'}
                                    >
                                        <SelectSingle
                                            id={`groups[${index}].roleName`}
                                            value={groups[`${index}`].roleName}
                                            isDisabled={isDisabled(group)}
                                            handleSelect={setFieldValue}
                                            direction="up"
                                            placeholderText="Select a role"
                                        >
                                            {roles.map(({ name }) => (
                                                <SelectOption key={name} value={name} />
                                            ))}
                                        </SelectSingle>
                                    </FormGroup>
                                </FlexItem>
                                {!isDisabled(group) && (
                                    <FlexItem>
                                        <Button
                                            variant="plain"
                                            aria-label="Delete rule"
                                            style={{ transform: 'translate(0, 42px)' }}
                                            onClick={() => arrayHelpers.remove(index)}
                                        >
                                            <TrashIcon />
                                        </Button>
                                    </FlexItem>
                                )}
                            </Flex>
                        ))}
                    {!disabled && (
                        <Flex>
                            <FlexItem>
                                <Button
                                    variant="link"
                                    isInline
                                    isDisabled={!!errors?.length}
                                    icon={<PlusCircleIcon className="pf-u-mr-sm" />}
                                    onClick={() =>
                                        arrayHelpers.push({
                                            roleName: '',
                                            props: {
                                                authProviderId: authProviderId || '',
                                                key: '',
                                                value: '',
                                                id: '',
                                            },
                                        })
                                    }
                                >
                                    Add new rule
                                </Button>
                            </FlexItem>
                        </Flex>
                    )}
                </>
            )}
        />
    );
}

export default RuleGroups;
