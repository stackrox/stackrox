import React from 'react';

import {
    Bullseye,
    Button,
    Divider,
    Flex,
    Form,
    FormGroup,
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
import isEqual from 'lodash/isEqual';
import sortBy from 'lodash/sortBy';
import { FormikHandlers, useFormik } from 'formik';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { VulnerabilitiesDeferralConfig } from 'services/DeferralConfigService';
import { useVulnerabilitiesDeferralConfig } from './useVulnerabilitiesDeferralConfig';

type BaseSettingProps = {
    fieldId: string;
    isEnabled: boolean;
    handleChange: FormikHandlers['handleChange'];
};

function NumericSetting({
    fieldId,
    value,
    isEnabled,
    handleChange,
}: BaseSettingProps & {
    value: number;
}) {
    return (
        <>
            <GridItem span={8} md={4} xl={3}>
                <FormGroup>
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            id={`${fieldId}.numDays`}
                            type="number"
                            style={{ width: '70px' }}
                            value={value}
                            onChange={(_, e) => handleChange(e)}
                            isDisabled={!isEnabled}
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
                        isChecked={isEnabled}
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
    isEnabled,
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
                        isChecked={isEnabled}
                        onChange={(_, e) => handleChange(e)}
                    />
                </FormGroup>
            </GridItem>
        </>
    );
}
function getDefaultConfig(): VulnerabilitiesDeferralConfig {
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

function ensureMinimumDayOptions(
    config: VulnerabilitiesDeferralConfig
): VulnerabilitiesDeferralConfig {
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
    const { config, isConfigLoading, isUpdateInProgress, configLoadError } =
        useVulnerabilitiesDeferralConfig();

    const { values, handleChange } = useFormik({
        enableReinitialize: true,
        // Ensure that there are at least 4 day options in case this array was set to zero via the API
        initialValues: ensureMinimumDayOptions(config ?? getDefaultConfig()),
        onSubmit: () => {},
    });

    const isConfigDirty = !isEqual(ensureMinimumDayOptions(config ?? getDefaultConfig()), values);

    const { dayOptions, fixableCveOptions, customDate } = values.expiryOptions;

    return (
        <>
            <div className="pf-u-py-md pf-u-px-md pf-u-px-lg-on-xl">
                <Split className="pf-u-align-items-center">
                    <SplitItem isFilled>
                        <Text>Configure deferral behavior for vulnerabilities</Text>
                    </SplitItem>
                    <SplitItem>
                        <Button
                            variant="primary"
                            isDisabled={!isConfigDirty}
                            isLoading={isUpdateInProgress}
                        >
                            Save
                        </Button>
                    </SplitItem>
                </Split>
            </div>
            <Divider component="div" />
            <PageSection variant="light" component="div">
                <Title headingLevel="h2">Configure deferral times</Title>
                {isConfigLoading && (
                    <Bullseye>
                        <Spinner aria-label="Loading current vulnerability deferral configuration" />
                    </Bullseye>
                )}
                {configLoadError && (
                    <Bullseye>
                        <EmptyStateTemplate
                            title="Error loading vulnerability deferral configuration"
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-u-danger-color-100"
                        >
                            {getAxiosErrorMessage(configLoadError)}
                        </EmptyStateTemplate>
                    </Bullseye>
                )}
                {!isConfigLoading && (
                    <Form className="pf-u-py-lg">
                        <Grid hasGutter>
                            {dayOptions.map(({ numDays, enabled }, index) => {
                                return (
                                    <NumericSetting
                                        // Note, if we ever support removing or reordering day options, we'll need to
                                        // use a non-index key here.
                                        // eslint-disable-next-line react/no-array-index-key
                                        key={index}
                                        fieldId={`expiryOptions.dayOptions[${index}]`}
                                        value={numDays}
                                        isEnabled={enabled}
                                        handleChange={handleChange}
                                    />
                                );
                            })}
                            {/* TODO Need 'Indefinitely' added to the API */}
                            <BooleanSetting
                                fieldId="TODO"
                                label="Indefinitely (TODO)"
                                isEnabled={false}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.fixableCveOptions.allFixable"
                                label="Expires when all CVEs fixable"
                                isEnabled={fixableCveOptions.allFixable}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.fixableCveOptions.anyFixable"
                                label="Expires when any CVE fixable"
                                isEnabled={fixableCveOptions.anyFixable}
                                handleChange={handleChange}
                            />
                            <BooleanSetting
                                fieldId="expiryOptions.customDate"
                                label="Allow custom date"
                                isEnabled={customDate}
                                handleChange={handleChange}
                            />
                        </Grid>
                    </Form>
                )}
            </PageSection>
        </>
    );
}

export default VulnerabilitiesConfiguration;
