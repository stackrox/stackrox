import React, { ReactElement, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import {
    Alert,
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    FormSection,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    PageSection,
    Split,
    SplitItem,
    Switch,
    Text,
    TextArea,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { useFormik } from 'formik';
import * as yup from 'yup';

import { PublicConfigAction } from 'reducers/publicConfig';
import { saveSystemConfig } from 'services/SystemConfigService';
import { PrivateConfig, PublicConfig, SystemConfig } from 'types/config.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { selectors } from 'reducers';
import { initializeAnalytics } from 'global/initializeAnalytics';

import ColorPicker from 'Components/ColorPicker';
import ClusterLabelsTable from 'Containers/Clusters/ClusterLabelsTable';
import FormSelect from './FormSelect';
import { convertBetweenBytesAndMB } from '../SystemConfig.utils';

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
    onCancelEditConfig: () => void;
};

const validationSchema = yup.object().shape({
    privateConfig: yup.object().shape({
        reportRetentionConfig: yup.object().shape({
            downloadableReportGlobalRetentionBytes: yup
                .number()
                .min(convertBetweenBytesAndMB(50, 'MB'), 'The number must be at least 50 MB')
                .required(),
        }),
    }),
});

const SystemConfigForm = ({
    systemConfig,
    setSystemConfig,
    onCancelEditConfig,
}: SystemConfigFormProps): ReactElement => {
    const dispatch = useDispatch();
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const isTelemetryConfigured = useSelector(selectors.getIsTelemetryConfigured);
    const telemetryConfig = useSelector(selectors.getTelemetryConfig);

    const { privateConfig } = systemConfig;
    const publicConfig = getCompletePublicConfig(systemConfig);
    const {
        dirty,
        errors,
        isSubmitting,
        isValid,
        setFieldValue,
        setSubmitting,
        submitForm,
        values,
    } = useFormik<Values>({
        initialValues: { privateConfig, publicConfig },
        validationSchema,
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
                            telemetryConfig.endpoint,
                            telemetryConfig.userId
                        );
                    }

                    dispatch(action);
                    setSystemConfig(data);
                    setErrorMessage(null);
                    setSubmitting(false);
                    onCancelEditConfig();

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

    function onDownloadableReportChange(value, event) {
        const valueInBytes = convertBetweenBytesAndMB(value, 'MB');
        return setFieldValue(event.target.id, valueInBytes);
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

    const downloadableReportRetentionError =
        errors.privateConfig?.reportRetentionConfig?.downloadableReportGlobalRetentionBytes;

    return (
        <>
            <PageSection variant="light">
                <Flex>
                    <Title headingLevel="h1">System Configuration</Title>
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Button
                            variant="primary"
                            type="button"
                            isDisabled={!dirty || !isValid || isSubmitting}
                            isLoading={isSubmitting}
                            onClick={submitForm}
                        >
                            Save
                        </Button>
                    </FlexItem>
                    <Button
                        variant="secondary"
                        onClick={onCancelEditConfig}
                        isDisabled={isSubmitting}
                    >
                        Cancel
                    </Button>
                </Flex>
            </PageSection>
            <Divider component="div" />
            {typeof errorMessage === 'string' && (
                <Alert
                    variant="danger"
                    isInline
                    title="Failed to save system configuration"
                    component="p"
                >
                    {errorMessage}
                </Alert>
            )}
            <PageSection variant="light" isFilled>
                <Form className="overflow-scroll">
                    <FormSection>
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
                                            values?.privateConfig?.alertConfig
                                                ?.allRuntimeRetentionDurationDays
                                        }
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem>
                                <FormGroup
                                    label="Prepared downloadable vulnerability reports retention days"
                                    isRequired
                                    fieldId="privateConfig.reportRetentionConfig.downloadableReportRetentionDays"
                                >
                                    <TextInput
                                        isRequired
                                        type="number"
                                        id="privateConfig.reportRetentionConfig.downloadableReportRetentionDays"
                                        name="privateConfig.reportRetentionConfig.downloadableReportRetentionDays"
                                        value={
                                            values?.privateConfig?.reportRetentionConfig
                                                ?.downloadableReportRetentionDays
                                        }
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem>
                                <FormGroup
                                    label="Prepared downloadable vulnerability reports limit"
                                    isRequired
                                    fieldId="privateConfig.reportRetentionConfig.downloadableReportGlobalRetentionBytes"
                                >
                                    <Split hasGutter className="pf-v5-u-align-items-center">
                                        <SplitItem isFilled>
                                            <TextInput
                                                isRequired
                                                type="number"
                                                id="privateConfig.reportRetentionConfig.downloadableReportGlobalRetentionBytes"
                                                name="privateConfig.reportRetentionConfig.downloadableReportGlobalRetentionBytes"
                                                value={convertBetweenBytesAndMB(
                                                    values?.privateConfig?.reportRetentionConfig
                                                        ?.downloadableReportGlobalRetentionBytes,
                                                    'B'
                                                )}
                                                onChange={(event, value) =>
                                                    onDownloadableReportChange(value, event)
                                                }
                                                min={50}
                                                validated={
                                                    downloadableReportRetentionError
                                                        ? 'error'
                                                        : 'default'
                                                }
                                            />
                                        </SplitItem>
                                        <SplitItem>
                                            <Text>MB</Text>
                                        </SplitItem>
                                    </Split>
                                    <FormHelperText>
                                        <HelperText>
                                            <HelperTextItem
                                                variant={
                                                    downloadableReportRetentionError
                                                        ? 'error'
                                                        : 'default'
                                                }
                                            >
                                                {downloadableReportRetentionError}
                                            </HelperTextItem>
                                        </HelperText>
                                    </FormHelperText>
                                </FormGroup>
                            </GridItem>
                            <GridItem>
                                <FormGroup
                                    label="Administration events retention days"
                                    isRequired
                                    fieldId="privateConfig.administrationEventsConfig.retentionDurationDays"
                                >
                                    <TextInput
                                        isRequired
                                        type="number"
                                        id="privateConfig.administrationEventsConfig.retentionDurationDays"
                                        name="privateConfig.administrationEventsConfig.retentionDurationDays"
                                        value={
                                            values?.privateConfig?.administrationEventsConfig
                                                ?.retentionDurationDays
                                        }
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
                                    />
                                </FormGroup>
                            </GridItem>
                        </Grid>
                    </FormSection>
                    <FormSection>
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
                                        onChange={(event, value) => onChange(value, event)}
                                        min={0}
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
                    </FormSection>
                    <FormSection>
                        <Title headingLevel="h2">Public configuration</Title>
                        <Grid hasGutter>
                            <GridItem sm={12} md={6}>
                                <Card isFlat data-testid="header-config">
                                    <CardHeader
                                        actions={{
                                            actions: (
                                                <>
                                                    <Switch
                                                        id="publicConfig.header.enabled"
                                                        label="Enabled"
                                                        labelOff="Disabled"
                                                        isChecked={
                                                            values?.publicConfig?.header?.enabled
                                                        }
                                                        onChange={(event, value) =>
                                                            onChange(value, event)
                                                        }
                                                    />
                                                </>
                                            ),
                                            hasNoOffset: false,
                                            className: undefined,
                                        }}
                                    >
                                        {
                                            <>
                                                <CardTitle component="h3">
                                                    Header configuration
                                                </CardTitle>
                                            </>
                                        }
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
                                                            value={
                                                                values?.publicConfig?.header?.text
                                                            }
                                                            onChange={(event, value) =>
                                                                onChange(value, event)
                                                            }
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
                                                            color={
                                                                values?.publicConfig?.header?.color
                                                            }
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
                                                                values?.publicConfig?.header
                                                                    ?.size ?? 'UNSET'
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
                                                                values?.publicConfig?.header
                                                                    ?.backgroundColor
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
                                    <CardHeader
                                        actions={{
                                            actions: (
                                                <>
                                                    <Switch
                                                        id="publicConfig.footer.enabled"
                                                        label="Enabled"
                                                        labelOff="Disabled"
                                                        isChecked={
                                                            values?.publicConfig?.footer?.enabled
                                                        }
                                                        onChange={(event, value) =>
                                                            onChange(value, event)
                                                        }
                                                    />
                                                </>
                                            ),
                                            hasNoOffset: false,
                                            className: undefined,
                                        }}
                                    >
                                        {
                                            <>
                                                <CardTitle component="h3">
                                                    Footer configuration
                                                </CardTitle>
                                            </>
                                        }
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
                                                            value={
                                                                values?.publicConfig?.footer?.text
                                                            }
                                                            onChange={(event, value) =>
                                                                onChange(value, event)
                                                            }
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
                                                            color={
                                                                values?.publicConfig?.footer?.color
                                                            }
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
                                                                values?.publicConfig?.footer
                                                                    ?.size ?? 'UNSET'
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
                                                                values?.publicConfig?.footer
                                                                    ?.backgroundColor
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
                                    <CardHeader
                                        actions={{
                                            actions: (
                                                <>
                                                    <Switch
                                                        id="publicConfig.loginNotice.enabled"
                                                        label="Enabled"
                                                        labelOff="Disabled"
                                                        isChecked={
                                                            values?.publicConfig?.loginNotice
                                                                ?.enabled
                                                        }
                                                        onChange={(event, value) =>
                                                            onChange(value, event)
                                                        }
                                                    />
                                                </>
                                            ),
                                            hasNoOffset: false,
                                            className: undefined,
                                        }}
                                    >
                                        {
                                            <>
                                                <CardTitle component="h3">
                                                    Login configuration
                                                </CardTitle>
                                            </>
                                        }
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
                                                    onChange={(event, value) =>
                                                        onChange(value, event)
                                                    }
                                                />
                                            </FormGroup>
                                        </FormSection>
                                    </CardBody>
                                </Card>
                            </GridItem>
                            {isTelemetryConfigured && (
                                <GridItem md={6}>
                                    <Card isFlat data-testid="telemetry-config">
                                        <CardHeader
                                            actions={{
                                                actions: (
                                                    <>
                                                        <Switch
                                                            id="publicConfig.telemetry.enabled"
                                                            label="Enabled"
                                                            labelOff="Disabled"
                                                            isChecked={
                                                                values?.publicConfig?.telemetry
                                                                    ?.enabled
                                                            }
                                                            onChange={(event, value) =>
                                                                onChange(value, event)
                                                            }
                                                        />
                                                    </>
                                                ),
                                                hasNoOffset: false,
                                                className: undefined,
                                            }}
                                        >
                                            {
                                                <>
                                                    <CardTitle component="h3">
                                                        Online Telemetry Data Collection
                                                    </CardTitle>
                                                </>
                                            }
                                        </CardHeader>
                                        <Divider component="div" />
                                        <CardBody>
                                            <p className="pf-v5-u-mb-sm">
                                                Online telemetry data collection allows Red Hat to
                                                use anonymized information to enhance your user
                                                experience. Consult the documentation to see what is
                                                collected, and for information about how to opt out.
                                            </p>
                                        </CardBody>
                                    </Card>
                                </GridItem>
                            )}
                        </Grid>
                    </FormSection>
                </Form>
            </PageSection>
        </>
    );
};

export default SystemConfigForm;
