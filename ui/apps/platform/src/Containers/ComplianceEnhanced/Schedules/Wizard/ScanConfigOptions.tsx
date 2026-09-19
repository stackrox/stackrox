import { useRef, useState } from 'react';
import type { FormEvent, KeyboardEvent, ReactElement } from 'react';
import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import {
    Divider,
    Flex,
    FlexItem,
    Form,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Label,
    LabelGroup,
    PageSection,
    Stack,
    StackItem,
    TextArea,
    TextInput,
    TimePicker,
    Title,
} from '@patternfly/react-core';

import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';

import usePageAction from 'hooks/usePageAction';
import { allNodesRole, isValidNodeRole } from '../compliance.scanConfigs.utils';
import type { PageActions, ScanConfigFormValues } from '../compliance.scanConfigs.utils';

import { helperTextForName, helperTextForNameEdit, helperTextForTime } from './useFormikScanConfig';

import './ScanConfigOptions.css';

function ScanConfigOptions(): ReactElement {
    const formik: FormikContextType<ScanConfigFormValues> = useFormikContext();
    const { pageAction } = usePageAction<PageActions>();
    const isEditAction = pageAction === 'edit';
    const [nodeRoleInput, setNodeRoleInput] = useState('');
    const [nodeRoleInputError, setNodeRoleInputError] = useState('');

    // Keep a ref to the latest node roles so that handlers reading them (addNodeRole,
    // removeNodeRole) always derive from current state rather than a stale render closure.
    // Without this, a blur-commit (addNodeRole) followed by a chip-remove click
    // (removeNodeRole) in the same tick clobbers the just-added role, because the click
    // handler was created on a render before the blur updated formik state, and formik's
    // setFieldValue is async so it has not re-rendered yet. Composing updates through the
    // ref makes back-to-back updates in the same tick see each other's result.
    const nodeRolesRef = useRef(formik.values.parameters.nodeRoles);
    // Sync from formik on each render so external value changes (e.g. loading an
    // existing config) are reflected.
    nodeRolesRef.current = formik.values.parameters.nodeRoles;

    function updateNodeRoles(updater: (currentRoles: string[]) => string[]) {
        const newRoles = updater(nodeRolesRef.current);
        nodeRolesRef.current = newRoles;
        formik.setFieldValue('parameters.nodeRoles', newRoles);
    }

    function addNodeRole(role: string) {
        const trimmed = role.trim();
        if (!trimmed) {
            return;
        }
        if (!isValidNodeRole(trimmed)) {
            setNodeRoleInputError(
                `"${trimmed}" is invalid. Use 1-39 alphanumeric characters and hyphens, starting and ending with a letter or number, or @all.`
            );
            return;
        }
        setNodeRoleInputError('');
        setNodeRoleInput('');
        updateNodeRoles((currentRoles) => {
            if (currentRoles.includes(trimmed)) {
                return currentRoles;
            }
            if (trimmed === allNodesRole) {
                return [allNodesRole];
            }
            if (currentRoles.includes(allNodesRole)) {
                return [trimmed];
            }
            return [...currentRoles, trimmed];
        });
    }

    function removeNodeRole(role: string) {
        updateNodeRoles((currentRoles) => currentRoles.filter((r) => r !== role));
    }

    function handleNodeRoleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
        if (e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            addNodeRole(nodeRoleInput);
        }
    }

    function handleSelectChange(id: string, value: string): void {
        formik.setFieldValue('parameters.daysOfWeek', []);
        formik.setFieldValue('parameters.daysOfMonth', []);
        formik.setFieldValue(id, value);
    }

    function handleTimeChange(_event: FormEvent<HTMLInputElement>, time: string): void {
        formik.setFieldValue('parameters.time', time);
    }

    function onScheduledDaysChange(id: string, selection: string[]) {
        formik.setFieldValue(id, selection, true);
    }

    return (
        <>
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v6-u-py-lg pf-v6-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Parameters</Title>
                    </FlexItem>
                    <FlexItem>Set name and schedule to scan on a recurring basis</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v6-u-py-lg pf-v6-u-px-lg" id="scan-schedules-parameters">
                <Stack hasGutter>
                    <StackItem>
                        <Stack hasGutter>
                            <StackItem>
                                <FormLabelGroup
                                    label="Name"
                                    isRequired
                                    fieldId="parameters.name"
                                    errors={formik.errors}
                                    touched={formik.touched}
                                    helperText={
                                        isEditAction ? helperTextForNameEdit : helperTextForName
                                    }
                                >
                                    <TextInput
                                        isRequired
                                        type="text"
                                        id="parameters.name"
                                        name="parameters.name"
                                        value={formik.values.parameters.name}
                                        isDisabled={isEditAction}
                                        validated={
                                            formik.errors?.parameters?.name &&
                                            formik.touched?.parameters?.name
                                                ? 'error'
                                                : 'default'
                                        }
                                        onChange={(event) => formik.handleChange(event)}
                                        onBlur={formik.handleBlur}
                                    />
                                </FormLabelGroup>
                            </StackItem>
                            <StackItem>
                                <FormLabelGroup
                                    label="Description"
                                    fieldId="parameters.description"
                                    errors={formik.errors}
                                >
                                    <TextArea
                                        isRequired
                                        type="text"
                                        id="parameters.description"
                                        name="parameters.description"
                                        value={formik.values.parameters.description}
                                        onChange={(event) => formik.handleChange(event)}
                                        onBlur={formik.handleBlur}
                                    />
                                </FormLabelGroup>
                            </StackItem>
                        </Stack>
                    </StackItem>
                    <StackItem>
                        <Divider component="div" />
                    </StackItem>
                    <StackItem>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h3">Schedule</Title>
                            </FlexItem>
                            <FlexItem flex={{ default: 'flexNone' }}>
                                <Flex direction={{ default: 'column' }}>
                                    <Flex direction={{ default: 'row' }}>
                                        <FlexItem>
                                            <FormLabelGroup
                                                label="Frequency"
                                                fieldId="parameters.intervalType"
                                                isRequired
                                                errors={formik.errors}
                                                touched={formik.touched}
                                            >
                                                <RepeatScheduleDropdown
                                                    fieldId="parameters.intervalType"
                                                    value={
                                                        formik.values.parameters.intervalType || ''
                                                    }
                                                    handleSelect={handleSelectChange}
                                                    includeDailyOption
                                                    onBlur={formik.handleBlur}
                                                />
                                            </FormLabelGroup>
                                        </FlexItem>
                                        <FlexItem>
                                            <FormLabelGroup
                                                label="On day(s)"
                                                fieldId={
                                                    formik.values.parameters.intervalType ===
                                                    'WEEKLY'
                                                        ? 'parameters.daysOfWeek'
                                                        : 'parameters.daysOfMonth'
                                                }
                                                errors={formik.errors}
                                                isRequired={
                                                    formik.values.parameters.intervalType ===
                                                        'WEEKLY' ||
                                                    formik.values.parameters.intervalType ===
                                                        'MONTHLY'
                                                }
                                                touched={formik.touched}
                                            >
                                                <DayPickerDropdown
                                                    fieldId={
                                                        formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? 'parameters.daysOfWeek'
                                                            : 'parameters.daysOfMonth'
                                                    }
                                                    value={
                                                        (formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? formik.values.parameters.daysOfWeek
                                                            : formik.values.parameters
                                                                  .daysOfMonth) ?? []
                                                    }
                                                    handleSelect={onScheduledDaysChange}
                                                    intervalType={
                                                        formik.values.parameters.intervalType
                                                    }
                                                    isEditable={
                                                        formik.values.parameters.intervalType ===
                                                            'MONTHLY' ||
                                                        formik.values.parameters.intervalType ===
                                                            'WEEKLY'
                                                    }
                                                    toggleId={
                                                        formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? 'parameters.daysOfWeek'
                                                            : 'parameters.daysOfMonth'
                                                    }
                                                    onBlur={() => {
                                                        const fieldId =
                                                            formik.values.parameters
                                                                .intervalType === 'WEEKLY'
                                                                ? 'parameters.daysOfWeek'
                                                                : 'parameters.daysOfMonth';
                                                        formik.setFieldTouched(fieldId, true);
                                                    }}
                                                />
                                            </FormLabelGroup>
                                        </FlexItem>
                                    </Flex>
                                    <FlexItem>
                                        <FormLabelGroup
                                            label="Time"
                                            fieldId="parameters.time"
                                            errors={formik.errors}
                                            isRequired
                                            touched={formik.touched}
                                            helperText={helperTextForTime}
                                        >
                                            <TimePicker
                                                time={formik.values.parameters.time}
                                                is24Hour
                                                onChange={handleTimeChange}
                                                inputProps={{
                                                    onBlur: formik.handleBlur,
                                                    name: 'parameters.time',
                                                }}
                                                invalidFormatErrorMessage="" // error messaging is handled by FormLabelGroup
                                                invalidMinMaxErrorMessage=""
                                            />
                                        </FormLabelGroup>
                                    </FlexItem>
                                </Flex>
                            </FlexItem>
                        </Flex>
                    </StackItem>
                    <StackItem>
                        <Divider component="div" />
                    </StackItem>
                    <StackItem>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h3">Node roles</Title>
                            </FlexItem>
                            <FlexItem>
                                <FormLabelGroup
                                    label="Roles"
                                    fieldId="parameters.nodeRoles"
                                    errors={formik.errors}
                                    touched={formik.touched}
                                    helperText="Determines which nodes are scanned for node-type profiles. If left empty, defaults to master and worker. Common roles: master, worker, infra, control-plane. Use @all to scan all nodes."
                                >
                                    <Stack hasGutter>
                                        <StackItem>
                                            <TextInput
                                                type="text"
                                                id="parameters.nodeRoles"
                                                aria-label="Node role"
                                                placeholder="Type a role and press Enter to add"
                                                value={nodeRoleInput}
                                                validated={nodeRoleInputError ? 'error' : 'default'}
                                                onChange={(_event, value) => {
                                                    setNodeRoleInput(value);
                                                    if (nodeRoleInputError) {
                                                        setNodeRoleInputError('');
                                                    }
                                                }}
                                                onKeyDown={handleNodeRoleKeyDown}
                                                onBlur={() => addNodeRole(nodeRoleInput)}
                                            />
                                            {nodeRoleInputError && (
                                                <FormHelperText>
                                                    <HelperText isLiveRegion>
                                                        <HelperTextItem variant="error">
                                                            {nodeRoleInputError}
                                                        </HelperTextItem>
                                                    </HelperText>
                                                </FormHelperText>
                                            )}
                                        </StackItem>
                                        {formik.values.parameters.nodeRoles.length > 0 && (
                                            <StackItem>
                                                <LabelGroup>
                                                    {formik.values.parameters.nodeRoles.map(
                                                        (role) => (
                                                            <Label
                                                                key={role}
                                                                onClose={() => removeNodeRole(role)}
                                                            >
                                                                {role}
                                                            </Label>
                                                        )
                                                    )}
                                                </LabelGroup>
                                            </StackItem>
                                        )}
                                    </Stack>
                                </FormLabelGroup>
                            </FlexItem>
                        </Flex>
                    </StackItem>
                </Stack>
            </Form>
        </>
    );
}

export default ScanConfigOptions;
