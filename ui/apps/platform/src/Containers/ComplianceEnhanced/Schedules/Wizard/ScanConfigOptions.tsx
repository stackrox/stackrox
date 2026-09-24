import { useRef, useState } from 'react';
import type { FormEvent, KeyboardEvent, ReactElement } from 'react';
import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import {
    Button,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
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
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    TimePicker,
    Title,
} from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';
import get from 'lodash/get';

import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';

import usePageAction from 'hooks/usePageAction';
import {
    allNodesRole,
    isValidNodeRole,
    nodeRoleValidationMessage,
} from '../compliance.scanConfigs.utils';
import type { PageActions, ScanConfigFormValues } from '../compliance.scanConfigs.utils';

import {
    helperTextForName,
    helperTextForNameEdit,
    helperTextForNodeRoles,
    helperTextForTime,
} from './useFormikScanConfig';

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
            // Whitespace-only or empty: discard the draft input and clear any stale error.
            setNodeRoleInput('');
            setNodeRoleInputError('');
            return;
        }
        if (!isValidNodeRole(trimmed)) {
            setNodeRoleInputError(`"${trimmed}" is invalid. ${nodeRoleValidationMessage}`);
            return;
        }
        if (nodeRolesRef.current.includes(trimmed)) {
            // Duplicate: give feedback instead of silently swallowing (backend rejects dupes).
            setNodeRoleInputError(`"${trimmed}" is already in the list.`);
            setNodeRoleInput('');
            return;
        }
        setNodeRoleInputError('');
        setNodeRoleInput('');
        updateNodeRoles((currentRoles) => {
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

    function clearNodeRoles() {
        setNodeRoleInput('');
        setNodeRoleInputError('');
        updateNodeRoles(() => []);
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

    // Combine the two error sources for the single node-roles helper slot: the uncommitted
    // draft-input error takes precedence, otherwise fall back to the committed (submit-time)
    // yup error, which is only ever a string for this field.
    const committedError = get(formik.errors, 'parameters.nodeRoles');
    const displayError =
        nodeRoleInputError || (typeof committedError === 'string' ? committedError : undefined);

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
                                {/*
                                    Unlike the sibling fields (which use FormLabelGroup), this field
                                    has an uncommitted draft-input sub-state, so it uses a plain
                                    FormGroup with a single combined helper slot: draft-input error,
                                    else the committed (submit-time) error, else the help text. This
                                    mirrors the chip-input pattern in DiagnosticBundleForm and avoids
                                    rendering two competing FormHelperText elements.
                                */}
                                <FormGroup label="Roles" fieldId="parameters.nodeRoles">
                                    <TextInputGroup>
                                        <TextInputGroupMain
                                            inputId="parameters.nodeRoles"
                                            aria-label="Roles"
                                            placeholder="Type a role and press Enter to add"
                                            value={nodeRoleInput}
                                            onChange={(_event, value) => {
                                                setNodeRoleInput(value);
                                                if (nodeRoleInputError) {
                                                    setNodeRoleInputError('');
                                                }
                                            }}
                                            onKeyDown={handleNodeRoleKeyDown}
                                            onBlur={() => addNodeRole(nodeRoleInput)}
                                        >
                                            <LabelGroup>
                                                {formik.values.parameters.nodeRoles.map((role) => (
                                                    <Label
                                                        key={role}
                                                        variant="outline"
                                                        onClose={(event) => {
                                                            event.stopPropagation();
                                                            removeNodeRole(role);
                                                        }}
                                                        closeBtnAriaLabel={`Remove ${role}`}
                                                    >
                                                        {role}
                                                    </Label>
                                                ))}
                                            </LabelGroup>
                                        </TextInputGroupMain>
                                        <TextInputGroupUtilities>
                                            {(formik.values.parameters.nodeRoles.length > 0 ||
                                                nodeRoleInput) && (
                                                <Button
                                                    icon={<TimesIcon />}
                                                    variant="plain"
                                                    onClick={clearNodeRoles}
                                                    aria-label="Clear all node roles"
                                                />
                                            )}
                                        </TextInputGroupUtilities>
                                    </TextInputGroup>
                                    <FormHelperText>
                                        <HelperText isLiveRegion>
                                            <HelperTextItem
                                                variant={displayError ? 'error' : 'default'}
                                            >
                                                {displayError || helperTextForNodeRoles}
                                            </HelperTextItem>
                                        </HelperText>
                                    </FormHelperText>
                                </FormGroup>
                            </FlexItem>
                        </Flex>
                    </StackItem>
                </Stack>
            </Form>
        </>
    );
}

export default ScanConfigOptions;
