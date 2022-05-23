import React, { ReactElement } from 'react';
import {
    ActionGroup,
    Button,
    TextArea,
    Form,
    FormSection,
    FormGroup,
    TextInput,
    Card,
    CardHeader,
    CardTitle,
    CardBody,
    CardHeaderMain,
    Divider,
    SelectOption,
    Grid,
    GridItem,
    CardActions,
    Switch,
} from '@patternfly/react-core';
import { useFormik } from 'formik';

import ColorPicker from 'Components/ColorPicker';
import { getProductBranding } from 'constants/productBranding';
import { PrivateConfig, PublicConfig, SystemConfig } from 'types/config.proto';
import { TelemetryConfig } from 'types/telemetry.proto';
import { ConfigTelemetryDetailContent } from '../ConfigTelemetryDetailWidget';
import FormSelect from './FormSelect';

function getCompletePublicConfig(systemConfig: SystemConfig): PublicConfig {
    return {
        header: {
            color: systemConfig?.publicConfig?.header?.color || '#000000',
            backgroundColor: systemConfig?.publicConfig?.header?.backgroundColor || '#FFFFFF',
            text: systemConfig?.publicConfig?.header?.text || '',
            enabled: systemConfig?.publicConfig?.header?.enabled || false,
            size: systemConfig?.publicConfig?.header?.size || 'UNSET',
        },
        footer: {
            color: systemConfig?.publicConfig?.footer?.color || '#000000',
            backgroundColor: systemConfig?.publicConfig?.footer?.backgroundColor || '#FFFFFF',
            text: systemConfig?.publicConfig?.footer?.text || '',
            enabled: systemConfig?.publicConfig?.footer?.enabled || false,
            size: systemConfig?.publicConfig?.footer?.size || 'UNSET',
        },
        loginNotice: {
            text: systemConfig?.publicConfig?.loginNotice?.text || '',
            enabled: systemConfig?.publicConfig?.loginNotice?.enabled || false,
        },
    };
}

type Values = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
    telemetryConfig: TelemetryConfig;
};

export type SystemConfigFormProps = {
    systemConfig: SystemConfig;
    telemetryConfig: TelemetryConfig;
    onCancel: () => void;
    onSubmit: (
        systemConfigSubmitted: SystemConfig,
        telemetryConfigSubmitted: TelemetryConfig
    ) => void;
};

const SystemConfigForm = ({
    systemConfig,
    telemetryConfig,
    onCancel,
    onSubmit,
}: SystemConfigFormProps): ReactElement => {
    const { type } = getProductBranding();
    const { privateConfig } = systemConfig;
    const publicConfig = getCompletePublicConfig(systemConfig);
    const { submitForm, setFieldValue, values, dirty, isValid, isSubmitting, setSubmitting } =
        useFormik<Values>({
            initialValues: { privateConfig, publicConfig, telemetryConfig },
            onSubmit: () => {
                // TODO next step will call save functions directly from services instead of indirectly via sagas.
                onSubmit(
                    { privateConfig: values.privateConfig, publicConfig: values.publicConfig },
                    values.telemetryConfig
                );
                setSubmitting(false);
            },
        });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function onCustomChange(value, id) {
        return setFieldValue(id, value, false);
    }

    function onSubmitForm(event) {
        event.preventDefault();
        return submitForm();
    }

    return (
        <Form onSubmit={onSubmitForm}>
            <Grid hasGutter md={12}>
                <GridItem md={12}>
                    <Card>
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle>Data Retention Configuration</CardTitle>
                            </CardHeaderMain>
                        </CardHeader>
                        <Divider component="div" />
                        <CardBody>
                            <FormSection>
                                <Grid hasGutter md={6}>
                                    <GridItem>
                                        <FormGroup
                                            label="All Runtime Violations"
                                            isRequired
                                            fieldId="privateConfig.alertConfig.allRuntimeRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.alertConfig.allRuntimeRetentionDurationDays"
                                                name="privateConfig.alertConfig.allRuntimeRetentionDurationDays"
                                                value={
                                                    values?.privateConfig?.alertConfig
                                                        ?.allRuntimeRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Runtime Violations For Deleted Deployments"
                                            isRequired
                                            fieldId="privateConfig.alertConfig.deletedRuntimeRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.alertConfig.deletedRuntimeRetentionDurationDays"
                                                name="privateConfig.alertConfig.deletedRuntimeRetentionDurationDays"
                                                value={
                                                    values?.privateConfig?.alertConfig
                                                        ?.deletedRuntimeRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Resolved Deploy-Phase Violations"
                                            isRequired
                                            fieldId="privateConfig.alertConfig.resolvedDeployRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.alertConfig.resolvedDeployRetentionDurationDays"
                                                name="privateConfig.alertConfig.resolvedDeployRetentionDurationDays"
                                                value={
                                                    values?.privateConfig?.alertConfig
                                                        ?.resolvedDeployRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Attempted Deploy-Phase Violations"
                                            isRequired
                                            fieldId="privateConfig.alertConfig.attemptedDeployRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.alertConfig.attemptedDeployRetentionDurationDays"
                                                name="privateConfig.alertConfig.attemptedDeployRetentionDurationDays"
                                                value={
                                                    values?.privateConfig?.alertConfig
                                                        ?.attemptedDeployRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Attempted Runtime Violations"
                                            isRequired
                                            fieldId="privateConfig.alertConfig.attemptedRuntimeRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.alertConfig.attemptedRuntimeRetentionDurationDays"
                                                name="privateConfig.alertConfig.attemptedRuntimeRetentionDurationDays"
                                                value={
                                                    values?.privateConfig?.alertConfig
                                                        ?.attemptedRuntimeRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Images No Longer Deployed"
                                            isRequired
                                            fieldId="privateConfig.imageRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.imageRetentionDurationDays"
                                                name="privateConfig.imageRetentionDurationDays"
                                                value={
                                                    values?.privateConfig
                                                        ?.imageRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem>
                                        <FormGroup
                                            label="Expired Vulnerability Requests"
                                            isRequired
                                            fieldId="privateConfig.expiredVulnReqRetentionDurationDays"
                                        >
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.expiredVulnReqRetentionDurationDays"
                                                name="privateConfig.expiredVulnReqRetentionDurationDays"
                                                value={
                                                    values?.privateConfig
                                                        ?.expiredVulnReqRetentionDurationDays
                                                }
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                </Grid>
                            </FormSection>
                        </CardBody>
                    </Card>
                </GridItem>
                <GridItem sm={12} md={6}>
                    <Card data-testid="header-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle>Header Configuration</CardTitle>
                            </CardHeaderMain>
                            <CardActions>
                                <Switch
                                    id="publicConfig.header.enabled"
                                    label="Enabled"
                                    labelOff="Disabled"
                                    isChecked={values?.publicConfig?.header?.enabled}
                                    onChange={onChange}
                                />
                            </CardActions>
                        </CardHeader>
                        <Divider component="div" />
                        <CardBody>
                            <FormSection>
                                <Grid hasGutter>
                                    <GridItem md={9}>
                                        <FormGroup
                                            label="Text (2000 character limit)"
                                            fieldId="publicConfig.header.text"
                                        >
                                            <TextArea
                                                isRequired
                                                type="text"
                                                id="publicConfig.header.text"
                                                name="publicConfig.header.text"
                                                value={values?.publicConfig?.header?.text}
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={3}>
                                        <FormGroup
                                            label="Text Color"
                                            isRequired
                                            fieldId="publicConfig.header.color"
                                        >
                                            <ColorPicker
                                                id="publicConfig.header.color"
                                                color={values?.publicConfig?.header?.color}
                                                onChange={onCustomChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={9}>
                                        <FormGroup
                                            label="Text Size"
                                            isRequired
                                            fieldId="publicConfig.header.size"
                                        >
                                            <FormSelect
                                                id="publicConfig.header.size"
                                                value={
                                                    values?.publicConfig?.header?.size ?? 'UNSET'
                                                }
                                                onChange={onCustomChange}
                                            >
                                                <SelectOption key={0} value="SMALL" />
                                                <SelectOption key={1} value="MEDIUM" />
                                                <SelectOption key={2} value="LARGE" />
                                            </FormSelect>
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={3}>
                                        <FormGroup
                                            label="Background Color"
                                            isRequired
                                            fieldId="publicConfig.header.backgroundColor"
                                        >
                                            <ColorPicker
                                                id="publicConfig.header.backgroundColor"
                                                color={
                                                    values?.publicConfig?.header?.backgroundColor
                                                }
                                                onChange={onCustomChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                </Grid>
                            </FormSection>
                        </CardBody>
                    </Card>
                </GridItem>
                <GridItem sm={12} md={6}>
                    <Card data-testid="footer-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle>Footer Configuration</CardTitle>
                            </CardHeaderMain>
                            <CardActions>
                                <Switch
                                    id="publicConfig.footer.enabled"
                                    label="Enabled"
                                    labelOff="Disabled"
                                    isChecked={values?.publicConfig?.footer?.enabled}
                                    onChange={onChange}
                                />
                            </CardActions>
                        </CardHeader>
                        <Divider component="div" />
                        <CardBody>
                            <FormSection>
                                <Grid hasGutter>
                                    <GridItem md={9}>
                                        <FormGroup
                                            label="Text (2000 character limit)"
                                            fieldId="publicConfig.footer.text"
                                        >
                                            <TextArea
                                                isRequired
                                                type="text"
                                                id="publicConfig.footer.text"
                                                name="publicConfig.footer.text"
                                                value={values?.publicConfig?.footer?.text}
                                                onChange={onChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={3}>
                                        <FormGroup
                                            label="Text Color"
                                            isRequired
                                            fieldId="publicConfig.footer.color"
                                        >
                                            <ColorPicker
                                                id="publicConfig.footer.color"
                                                color={values?.publicConfig?.footer?.color}
                                                onChange={onCustomChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={9}>
                                        <FormGroup
                                            label="Text Size"
                                            isRequired
                                            fieldId="publicConfig.footer.size"
                                        >
                                            <FormSelect
                                                id="publicConfig.footer.size"
                                                value={
                                                    values?.publicConfig?.footer?.size ?? 'UNSET'
                                                }
                                                onChange={onCustomChange}
                                            >
                                                <SelectOption key={0} value="SMALL" />
                                                <SelectOption key={1} value="MEDIUM" />
                                                <SelectOption key={2} value="LARGE" />
                                            </FormSelect>
                                        </FormGroup>
                                    </GridItem>
                                    <GridItem md={3}>
                                        <FormGroup
                                            label="Background Color"
                                            isRequired
                                            fieldId="publicConfig.footer.backgroundColor"
                                        >
                                            <ColorPicker
                                                id="publicConfig.footer.backgroundColor"
                                                color={
                                                    values?.publicConfig?.footer?.backgroundColor
                                                }
                                                onChange={onCustomChange}
                                            />
                                        </FormGroup>
                                    </GridItem>
                                </Grid>
                            </FormSection>
                        </CardBody>
                    </Card>
                </GridItem>
                <GridItem md={6}>
                    <Card data-testid="login-notice-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle>Login Configuration</CardTitle>
                            </CardHeaderMain>
                            <CardActions>
                                <Switch
                                    id="publicConfig.loginNotice.enabled"
                                    label="Enabled"
                                    labelOff="Disabled"
                                    isChecked={values?.publicConfig?.loginNotice?.enabled}
                                    onChange={onChange}
                                />
                            </CardActions>
                        </CardHeader>
                        <Divider component="div" />
                        <CardBody>
                            <FormSection>
                                <FormGroup
                                    label="Text (2000 character limit)"
                                    fieldId="publicConfig.loginNotice.text"
                                >
                                    <TextArea
                                        isRequired
                                        type="text"
                                        id="publicConfig.loginNotice.text"
                                        name="publicConfig.loginNotice.text"
                                        value={values?.publicConfig?.loginNotice?.text}
                                        onChange={onChange}
                                    />
                                </FormGroup>
                            </FormSection>
                        </CardBody>
                    </Card>
                </GridItem>
                {type === 'RHACS_BRANDING' && (
                    <GridItem md={6}>
                        <Card>
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle>Online Telemetry Data Collection</CardTitle>
                                </CardHeaderMain>
                                <CardActions>
                                    <Switch
                                        id="telemetryConfig.enabled"
                                        label="Enabled"
                                        labelOff="Disabled"
                                        isChecked={values?.telemetryConfig?.enabled}
                                        onChange={onChange}
                                    />
                                </CardActions>
                            </CardHeader>
                            <Divider component="div" />
                            <CardBody>
                                <ConfigTelemetryDetailContent />
                            </CardBody>
                        </Card>
                    </GridItem>
                )}
            </Grid>
            <ActionGroup>
                <Button
                    variant="primary"
                    type="submit"
                    isDisabled={!dirty || !isValid || isSubmitting}
                    isLoading={isSubmitting}
                >
                    Save
                </Button>
                <Button variant="secondary" onClick={onCancel}>
                    Cancel
                </Button>
            </ActionGroup>
        </Form>
    );
};

export default SystemConfigForm;
