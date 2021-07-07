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
    onChange: (
        _value: unknown,
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) => void;
    roles: Role[];
    groups: Group[];
    setFieldValue: (name: string, value: string) => void;
};

const ruleKeys = ['userid', 'name', 'email', 'super-groups'];

function RuleGroups({
    onChange,
    setFieldValue,
    groups = [],
    roles = [],
}: RuleGroupsProps): ReactElement {
    return (
        <FieldArray
            name="groups"
            render={(arrayHelpers) => (
                <>
                    {groups.map((group, index: number) => (
                        <Flex
                            key={`${group.props.authProviderId}_${group.props.key || ''}_${
                                group.props.value || ''
                            }_${index}`}
                        >
                            <FlexItem>
                                <FormGroup label="Key" fieldId={`groups[${index}].props.key`}>
                                    <SelectSingle
                                        id={`groups[${index}].props.key`}
                                        value={groups[`${index}`].props.key}
                                        isDisabled={false}
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
                                <FormGroup label="Value" fieldId={`groups[${index}].props.value`}>
                                    <TextInput
                                        type="text"
                                        id={`groups[${index}].props.value`}
                                        value={groups[`${index}`].props.value}
                                        onChange={onChange}
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
                                        isDisabled={false}
                                        handleSelect={setFieldValue}
                                        direction="up"
                                    >
                                        {roles.map(({ name }) => (
                                            <SelectOption key={name} value={name} />
                                        ))}
                                    </SelectSingle>
                                </FormGroup>
                            </FlexItem>
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
                        </Flex>
                    ))}
                    <Flex>
                        <FlexItem>
                            <Button
                                variant="link"
                                isInline
                                icon={<PlusCircleIcon className="pf-u-mr-sm" />}
                                onClick={() =>
                                    arrayHelpers.push({
                                        roleName: '',
                                        props: { authProviderId: '', key: '', value: '' },
                                    })
                                }
                            >
                                Add new rule
                            </Button>
                        </FlexItem>
                    </Flex>
                </>
            )}
        />
    );
}

export default RuleGroups;
