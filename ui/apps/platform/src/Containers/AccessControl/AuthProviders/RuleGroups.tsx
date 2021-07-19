/* eslint-disable react/no-array-index-key */
/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable react/jsx-no-bind */
import React, { ReactElement } from 'react';
import { FieldArray } from 'formik';
import { Button, Flex, FlexItem, FormGroup, SelectOption, TextInput } from '@patternfly/react-core';
import { ArrowRightIcon, PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';

import { Group } from 'services/AuthService';
import { Role } from 'services/RolesService';
import SelectSingle from 'Components/SelectSingle';

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
};

const ruleKeys = ['userid', 'name', 'email', 'groups'];

function RuleGroups({
    authProviderId,
    onChange,
    setFieldValue,
    groups = [],
    roles = [],
    disabled = false,
}: RuleGroupsProps): ReactElement {
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
                                    <FormGroup label="Key" fieldId={`groups[${index}].props.key`}>
                                        <SelectSingle
                                            id={`groups[${index}].props.key`}
                                            value={groups[`${index}`].props.key}
                                            isDisabled={disabled}
                                            handleSelect={setFieldValue}
                                            direction="up"
                                        >
                                            {ruleKeys.map((ruleKey) => (
                                                <SelectOption key={ruleKey} value={ruleKey} />
                                            ))}
                                        </SelectSingle>
                                    </FormGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormGroup
                                        label="Value"
                                        fieldId={`groups[${index}].props.value`}
                                    >
                                        <TextInput
                                            type="text"
                                            id={`groups[${index}].props.value`}
                                            value={groups[`${index}`].props.value}
                                            onChange={onChange}
                                            isDisabled={disabled}
                                        />
                                    </FormGroup>
                                </FlexItem>
                                <FlexItem>
                                    <ArrowRightIcon style={{ transform: 'translate(0, 42px)' }} />
                                </FlexItem>
                                <FlexItem>
                                    <FormGroup label="Role" fieldId={`groups[${index}].roleName`}>
                                        <SelectSingle
                                            id={`groups[${index}].roleName`}
                                            value={groups[`${index}`].roleName}
                                            isDisabled={disabled}
                                            handleSelect={setFieldValue}
                                            direction="up"
                                        >
                                            {roles.map(({ name }) => (
                                                <SelectOption key={name} value={name} />
                                            ))}
                                        </SelectSingle>
                                    </FormGroup>
                                </FlexItem>
                                {!disabled && (
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
                                    icon={<PlusCircleIcon className="pf-u-mr-sm" />}
                                    onClick={() =>
                                        arrayHelpers.push({
                                            roleName: roles[0]?.name || '',
                                            props: {
                                                authProviderId: authProviderId || '',
                                                key: ruleKeys[0],
                                                value: '',
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
