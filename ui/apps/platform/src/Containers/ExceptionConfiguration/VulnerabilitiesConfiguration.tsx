import React from 'react';

import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Bullseye,
    Button,
    Divider,
    Flex,
    Form,
    FormGroup,
    FormGroupProps,
    Grid,
    GridItem,
    PageSection,
    Spinner,
    Split,
    SplitItem,
    Switch,
    Text,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import get from 'lodash/get';
import isEqual from 'lodash/isEqual';
import sortBy from 'lodash/sortBy';
import { FormikHandlers, useFormik } from 'formik';
import * as yup from 'yup';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { VulnerabilitiesExceptionConfig } from 'services/ExceptionConfigService';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import usePermissions from 'hooks/usePermissions';

import { useVulnerabilitiesExceptionConfig } from './useVulnerabilitiesExceptionConfig';

type BaseSettingProps = {
    fieldId: string;
    isSettingEnabled: boolean;
    isDisabled: boolean;
    handleChange: FormikHandlers['handleChange'];
};

function NumericSetting({
    fieldId,
    value,
    isSettingEnabled,
    isDisabled,
    handleChange,
    validated,
    helperTextInvalid,
}: BaseSettingProps & {
    value: number;
    validated: FormGroupProps['validated'];
    helperTextInvalid: FormGroupProps['helperTextInvalid'];
}) {
    return (
        <>
            <GridItem span={8} md={4} xl={3}>
                <FormGroup validated={validated} helperTextInvalid={helperTextInvalid}>
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            id={`${fieldId}.numDays`}
                            type="number"
                            style={{ width: '100px' }}
                            value={value}
                            onChange={(_, e) => handleChange(e)}
                            isDisabled={!isSettingEnabled || isDisabled}
                            validated={validated}
                        />
                        <span>days</span>
                    </Flex>
                </FormGroup>
            </GridItem>
            <GridItem span={4} md={8} xl={9}>
                <FormGroup>
                    <Switch
                        id={`${fieldId}.enabled`}
                        label="Enabled"
                        labelOff="Disabled"
                        isChecked={isSettingEnabled}
                        isDisabled={isDisabled}
                        onChange={(_, e) => handleChange(e)}
                    />
                </FormGroup>
            </GridItem>
        </>
    );
}

function BooleanSetting({
    fieldId,
    label,
    isSettingEnabled,
    isDisabled,
    handleChange,
}: BaseSettingProps & {
    label: string;
}) {
    return (
        <>
            <GridItem className="pf-u-py-xs" span={8} md={4} xl={3}>
                <p>{label}</p>
            </GridItem>
            <GridItem className="pf-u-py-xs" span={4} md={8} xl={9}>
                <FormGroup>
                    <Switch
                        id={fieldId}
                        label="Enabled"
                        labelOff="Disabled"
                        isChecked={isSettingEnabled}
                        isDisabled={isDisabled}
                        onChange={(_, e) => handleChange(e)}
                    />
                </FormGroup>
            </GridItem>
        </>
    );
}

function getDefaultConfig(): VulnerabilitiesExceptionConfig {
    return {
        expiryOptions: {
            dayOptions: [
                { numDays: 1, enabled: false },
                { numDays: 1, enabled: false },
                { numDays: 1, enabled: false },
                { numDays: 1, enabled: false },
            ],
            fixableCveOptions: {
                allFixable: false,
                anyFixable: false,
            },
            customDate: false,
        },
    };
}

const validationSchema = yup.object({
    expiryOptions: yup.object({
        dayOptions: yup
            .array()
            .ensure()
            .of(
                yup.object({
                    numDays: yup
                        .number()
                        .test(
                            'isPositive',
                            'Number of days must be greater than zero',
                            (value) => typeof value === 'number' && value > 0
                        )
                        .required('Number of days must not be empty'),
                    enabled: yup.boolean().required(),
                })
            )
            .test('dayValuesAreUnique', (dayOptions, testContext) => {
                if (!dayOptions) {
                    return true;
                }

                const dayValueToIndexMap: Record<number, number> = {};
                let error: yup.ValidationError | undefined;

                // If there are duplicate, enabled `dayOptions` with the same `numDays` value return a validation
                // error at the first index of duplication.
                dayOptions.forEach((dayOption, currentIndex) => {
                    if (!dayOption.enabled || error) {
                        return;
                    }
                    const existingIndex = dayValueToIndexMap[dayOption.numDays];
                    if (existingIndex !== undefined) {
                        error = testContext.createError({
                            path: `expiryOptions.dayOptions[${existingIndex}].numDays`,
                            message: 'Number of days must be unique',
                        });
                    }
                    dayValueToIndexMap[dayOption.numDays] = currentIndex;
                });

                return error || true; // `yup` expects either and error object on validation failure, or `true` on validation success
            }),
        fixableCveOptions: yup
            .object({
                allFixable: yup.boolean().required(),
                anyFixable: yup.boolean().required(),
            })
            .required(),
        // TODO Need 'Indefinitely' added to validation once it is available in the API
        customDate: yup.boolean().required(),
    }),
});

function ensureMinimumDayOptions(
    config: VulnerabilitiesExceptionConfig
): VulnerabilitiesExceptionConfig {
    const minimumLength = 4;
    const dayOptions = [...config.expiryOptions.dayOptions];
    while (dayOptions.length < minimumLength) {
        dayOptions.push({ numDays: 1, enabled: false });
    }

    return {
        ...config,
        expiryOptions: {
            ...config.expiryOptions,
            dayOptions: sortBy(dayOptions, 'numDays'),
        },
    };
}

function VulnerabilitiesConfiguration() {
    const { toasts, addToast, removeToast } = useToasts();

    const { config, isConfigLoading, isUpdateInProgress, configLoadError, updateConfig } =
        useVulnerabilitiesExceptionConfig();

    const exceptionConfig = ensureMinimumDayOptions(config ?? getDefaultConfig());

    const { values, handleChange, errors, submitForm } = useFormik({
        enableReinitialize: true,
        // Ensure that there are at least 4 day options in case this array was set to zero via the API
        initialValues: exceptionConfig,
        validationSchema,
        onSubmit: (formValues) =>
            updateConfig(formValues, {
                onSuccess: () => {
                    addToast('The configuration was updated successfully', 'success');
                },
                onError: (err: unknown) => {
                    addToast(
                        'There was an error updating the configuration',
                        'danger',
                        getAxiosErrorMessage(err)
                    );
                },
            }),
    });

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Administration');

    const isConfigDirty = !isEqual(exceptionConfig, values);
    const hasFormError = Object.keys(errors).length > 0;

    const { dayOptions, fixableCveOptions, customDate } = values.expiryOptions;

    return (
        <>
            <div className="pf-u-py-md pf-u-px-md pf-u-px-lg-on-xl">
                <Split className="pf-u-align-items-center">
                    <SplitItem isFilled>
                        <Text>Configure exception behavior for vulnerabilities</Text>
                    </SplitItem>
                    {hasWriteAccessForPage && (
                        <SplitItem>
                            <Button
                                variant="primary"
                                isDisabled={!isConfigDirty || hasFormError}
                                isLoading={isUpdateInProgress}
                                onClick={submitForm}
                            >
                                Save
                            </Button>
                        </SplitItem>
                    )}
                </Split>
            </div>
            <Divider component="div" />
            <PageSection variant="light" component="div">
                <Title headingLevel="h2">Configure exception times</Title>
                {isConfigLoading && (
                    <Bullseye>
                        <Spinner aria-label="Loading current vulnerability exception configuration" />
                    </Bullseye>
                )}
                {configLoadError && (
                    <Bullseye>
                        <EmptyStateTemplate
                            title="Error loading vulnerability exception configuration"
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-u-danger-color-100"
                        >
                            {getAxiosErrorMessage(configLoadError)}
                        </EmptyStateTemplate>
                    </Bullseye>
                )}
                {!isConfigLoading && !configLoadError && (
                    <Form className="pf-u-py-lg">
                        <Grid hasGutter>
                            {dayOptions.map(({ numDays, enabled }, index) => {
                                const fieldIdPrefix = `expiryOptions.dayOptions[${index}]`;
                                const fieldError = get(errors, `${fieldIdPrefix}.numDays`);
                                const validated = fieldError ? 'error' : 'default';
                                return (
                                    <NumericSetting
                                        // Note, if we ever support removing or reordering day options, we'll need to
                                        // use a non-index key here.
                                        // eslint-disable-next-line react/no-array-index-key
                                        key={index}
                                        fieldId={fieldIdPrefix}
                                        value={numDays}
                                        isSettingEnabled={enabled}
                                        isDisabled={!hasWriteAccessForPage}
                                        handleChange={handleChange}
                                        validated={validated}
                                        helperTextInvalid={fieldError}
                                    />
                                );
                            })}
                            {/* TODO Need 'Indefinitely' added to the API */}
                            <BooleanSetting
                                fieldId="TODO"
                                label="Indefinitely (TODO)"
                                isSettingEnabled={false}
                                isDisabled={!hasWriteAccessForPage}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.fixableCveOptions.allFixable"
                                label="Expires when all CVEs fixable"
                                isSettingEnabled={fixableCveOptions.allFixable}
                                isDisabled={!hasWriteAccessForPage}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.fixableCveOptions.anyFixable"
                                label="Expires when any CVE fixable"
                                isSettingEnabled={fixableCveOptions.anyFixable}
                                isDisabled={!hasWriteAccessForPage}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.customDate"
                                label="Allow custom date"
                                isSettingEnabled={customDate}
                                isDisabled={!hasWriteAccessForPage}
                                handleChange={handleChange}
                            />
                        </Grid>
                    </Form>
                )}
            </PageSection>
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        timeout={variant === 'success'}
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default VulnerabilitiesConfiguration;
