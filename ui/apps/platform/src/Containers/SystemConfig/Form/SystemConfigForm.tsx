import React, { ReactElement, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import {
    ActionGroup,
    Alert,
    Button,
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    Divider,
    Form,
    FormGroup,
    FormSection,
    Grid,
    GridItem,
    SelectOption,
    Switch,
    TextArea,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { useFormik } from 'formik';

import ColorPicker from 'Components/ColorPicker';
import ClusterLabelsTable from 'Containers/Clusters/ClusterLabelsTable';
import { PublicConfigAction } from 'reducers/publicConfig';
import { saveSystemConfig } from 'services/SystemConfigService';
import { PrivateConfig, PublicConfig, SystemConfig } from 'types/config.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { selectors } from 'reducers';
import { initializeAnalytics } from 'global/initializeAnalytics';

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
        telemetry: {
            enabled: systemConfig?.publicConfig?.telemetry?.enabled !== false,
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
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const isTelemetryConfigured = useSelector(selectors.getIsTelemetryConfigured);
    const telemetryConfig = useSelector(selectors.getTelemetryConfig);

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
                        const action: PublicConfigAction = {
                            type: 'config/FETCH_PUBLIC_CONFIG_SUCCESS',
                            response: data.publicConfig || {
                                footer: null,
                                header: null,
                                loginNotice: null,
                                telemetry: null,
                            },
                        };

                        const isTelemetryEnabledCurr = data.publicConfig?.telemetry?.enabled;
                        const isTelemetryEnabledPrev = publicConfig.telemetry?.enabled;
                        if (isTelemetryEnabledCurr && isTelemetryConfigured) {
                            initializeAnalytics(
                                telemetryConfig.storageKeyV1,
                                telemetryConfig.userId
                            );
                        }

                        dispatch(action);
                        setSystemConfig(data);
                        setErrorMessage(null);
                        setSubmitting(false);
                        setIsNotEditing();

                        if (isTelemetryEnabledPrev && !isTelemetryEnabledCurr) {
                            window.location.reload();
                        }
                    })
                    .catch((error) => {
                        setSubmitting(false);
                        setErrorMessage(getAxiosErrorMessage(error));
                    });
            },
        });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onCustomChange(value, id) {
        return setFieldValue(id, value);
    }

    function handleChangeLabels(labels) {
        return onCustomChange(
            labels,
            'privateConfig.decommissionedClusterRetention.ignoreClusterLabels'
        );
    }

    return (
        <Form>
            <Title headingLevel="h2">Private data retention configuration</Title>
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
                                values?.privateConfig?.alertConfig?.allRuntimeRetentionDurationDays
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
                            value={values?.privateConfig?.imageRetentionDurationDays}
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
                            value={values?.privateConfig?.expiredVulnReqRetentionDurationDays}
                            onChange={onChange}
                        />
                    </FormGroup>
                </GridItem>
                <GridItem>
                    <FormGroup
                        label="Vulnerability report run history retention"
                        isRequired
                        fieldId="privateConfig.reportRetentionConfig.historyRetentionDurationDays"
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="privateConfig.reportRetentionConfig.historyRetentionDurationDays"
                            name="privateConfig.reportRetentionConfig.historyRetentionDurationDays"
                            value={
                                values?.privateConfig?.reportRetentionConfig
                                    ?.historyRetentionDurationDays
                            }
                            onChange={onChange}
                        />
                    </FormGroup>
                </GridItem>
            </Grid>
            <Title headingLevel="h3">Cluster deletion</Title>
            <Grid hasGutter md={6}>
                <GridItem>
                    <FormGroup
                        label="Decommissioned cluster age"
                        isRequired
                        fieldId="privateConfig.decommissionedClusterRetention.retentionDurationDays"
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="privateConfig.decommissionedClusterRetention.retentionDurationDays"
                            name="privateConfig.decommissionedClusterRetention.retentionDurationDays"
                            value={
                                values?.privateConfig?.decommissionedClusterRetention
                                    ?.retentionDurationDays
                            }
                            onChange={onChange}
                        />
                    </FormGroup>
                </GridItem>
                <GridItem>
                    <FormGroup
                        label="Ignore clusters which have the following labels"
                        fieldId="privateConfig.decommissionedClusterRetention.ignoreClusterLabels"
                    >
                        <ClusterLabelsTable
                            labels={
                                values.privateConfig.decommissionedClusterRetention
                                    .ignoreClusterLabels
                            }
                            hasAction
                            handleChangeLabels={handleChangeLabels}
                            isValueRequired
                        />
                    </FormGroup>
                </GridItem>
            </Grid>
            <Title headingLevel="h2">Public configuration</Title>
            <Grid hasGutter>
                <GridItem sm={12} md={6}>
                    <Card isFlat data-testid="header-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle component="h3">Header configuration</CardTitle>
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
                                                label="Text color of header"
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
                                                label="Background color of header"
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
                    <Card isFlat data-testid="footer-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle component="h3">Footer configuration</CardTitle>
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
                                                label="Text color of footer"
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
                                                label="Background color of footer"
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
                    <Card isFlat data-testid="login-notice-config">
                        <CardHeader>
                            <CardHeaderMain>
                                <CardTitle component="h3">Login configuration</CardTitle>
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
                {isTelemetryConfigured && (
                    <GridItem md={6}>
                        <Card isFlat data-testid="telemetry-config">
                            <CardHeader>
                                <CardHeaderMain>
                                    <CardTitle component="h3">
                                        Online Telemetry Data Collection
                                    </CardTitle>
                                </CardHeaderMain>
                                <CardActions>
                                    <Switch
                                        id="publicConfig.telemetry.enabled"
                                        label="Enabled"
                                        labelOff="Disabled"
                                        isChecked={values?.publicConfig?.telemetry?.enabled}
                                        onChange={onChange}
                                    />
                                </CardActions>
                            </CardHeader>
                            <Divider component="div" />
                            <CardBody>
                                <p className="pf-u-mb-sm">
                                    Online telemetry data collection allows Red Hat to use
                                    anonymized information to enhance your user experience. Consult
                                    the documentation to see what is collected, and for information
                                    about how to opt out.
                                </p>
                            </CardBody>
                        </Card>
                    </GridItem>
                )}
            </Grid>
            {typeof errorMessage === 'string' && (
                <Alert variant="danger" isInline title="Failed to save system configuration">
                    {errorMessage}
                </Alert>
            )}
            <ActionGroup>
                <Button
                    variant="primary"
                    type="button"
                    isDisabled={!dirty || !isValid || isSubmitting}
                    isLoading={isSubmitting}
                    onClick={submitForm}
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
