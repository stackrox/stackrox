import React, { ReactElement, useState } from 'react';
import { useDispatch } from 'react-redux';
import {
    ActionGroup,
    Alert,
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
import { types } from 'reducers/systemConfig';
import { saveSystemConfig } from 'services/SystemConfigService';
import { PrivateConfig, PublicConfig, SystemConfig } from 'types/config.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

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
};

export type SystemConfigFormProps = {
    systemConfig: SystemConfig;
    setSystemConfig: (systemConfig: SystemConfig) => void;
    setIsNotEditing: () => void;
};

const SystemConfigForm = ({
    systemConfig,
    setSystemConfig,
    setIsNotEditing,
}: SystemConfigFormProps): ReactElement => {
    const dispatch = useDispatch();
    const [errorMessage, setErrorMessage] = useState('');

    const { privateConfig } = systemConfig;
    const publicConfig = getCompletePublicConfig(systemConfig);
    const { submitForm, setFieldValue, values, dirty, isValid, isSubmitting, setSubmitting } =
        useFormik<Values>({
            initialValues: { privateConfig, publicConfig },
            onSubmit: () => {
                // Payload for privateConfig allows strings as number values.
                saveSystemConfig({
                    privateConfig: values.privateConfig,
                    publicConfig: values.publicConfig,
                })
                    .then((data) => {
                        // Simulate fetchPublicConfig response to update Redux state.
                        dispatch({
                            type: types.FETCH_PUBLIC_CONFIG.SUCCESS,
                            response: data.publicConfig,
                        });
                        setSystemConfig(data);
                        setIsNotEditing();
                    })
                    .catch((error) => {
                        setErrorMessage(getAxiosErrorMessage(error));
                    })
                    .finally(() => {
                        setSubmitting(false);
                    });
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
                                <CardTitle>Data retention configuration</CardTitle>
                            </CardHeaderMain>
                        </CardHeader>
                        <Divider component="div" />
                        <CardBody>
                            <FormSection>
                                <Grid hasGutter md={6}>
                                    <GridItem>
                                        <FormGroup
                                            label="All runtime violations"
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
                                            label="Runtime violations for deleted deployments"
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
                                            label="Resolved deploy-phase violations"
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
                                            label="Attempted deploy-phase violations"
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
                                            label="Attempted runtime violations"
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
                                            label="Images no longer deployed"
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
                                            label="Expired vulnerability requests"
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
                                <CardTitle>Header configuration</CardTitle>
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
                                            label="Text color"
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
                                            label="Text size"
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
                                            label="Background color"
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
                                <CardTitle>Footer configuration</CardTitle>
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
                                            label="Text color"
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
                                            label="Text size"
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
                                            label="Background color"
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
                                <CardTitle>Login configuration</CardTitle>
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
            </Grid>
            {errorMessage && (
                <Alert variant="danger" isInline title="Failed to save system configuration">
                    {errorMessage}
                </Alert>
            )}
            <ActionGroup>
                <Button
                    variant="primary"
                    type="submit"
                    isDisabled={!dirty || !isValid || isSubmitting}
                    isLoading={isSubmitting}
                >
                    Save
                </Button>
                <Button variant="secondary" onClick={setIsNotEditing} isDisabled={isSubmitting}>
                    Cancel
                </Button>
            </ActionGroup>
        </Form>
    );
};

export default SystemConfigForm;
