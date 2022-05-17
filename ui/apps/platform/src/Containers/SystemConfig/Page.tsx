import React, { ReactElement, useState } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ActionGroup, Button, PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';
import { useFormik } from 'formik';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';
import { actions as telemetryActions } from 'reducers/telemetryConfig';
import { PublicConfig, PrivateConfig, SystemConfig } from 'types/config.proto';
import { TelemetryConfig } from 'types/telemetry.proto';

import SystemConfigForm from './SystemConfigForm';
import Details from './Details';

export type PageProps = {
    systemConfig: SystemConfig;
    saveSystemConfig: (config) => void;
    telemetryConfig: TelemetryConfig;
    saveTelemetryConfig: (config) => void;
};

type InitialValues = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
    telemetryConfig: TelemetryConfig;
};

function getInitialValues(
    systemConfig: SystemConfig,
    telemetryConfig: TelemetryConfig
): InitialValues {
    const { privateConfig } = systemConfig;
    const publicConfig: PublicConfig = {
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
    return {
        privateConfig,
        publicConfig,
        telemetryConfig,
    };
}

const Page = ({
    systemConfig,
    saveSystemConfig,
    telemetryConfig,
    saveTelemetryConfig,
}: PageProps): ReactElement => {
    const initialValues = getInitialValues(systemConfig, telemetryConfig);
    const [isEditing, setIsEditing] = useState(false);
    const { submitForm, setFieldValue, values, dirty, isValid, isSubmitting, setSubmitting } =
        useFormik({
            initialValues,
            onSubmit,
        });

    function editSystemConfig() {
        setIsEditing(true);
    }

    function cancelEdit() {
        setIsEditing(false);
    }

    function onSubmit(config) {
        saveSystemConfig(config);
        saveTelemetryConfig(config.telemetryConfig);
        setIsEditing(false);
        setSubmitting(false);
    }

    function onSubmitForm(event) {
        event.preventDefault();
        return submitForm();
    }

    return (
        <>
            <PageSection variant="light" sticky="top">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">System Configuration</Title>
                    </FlexItem>
                    <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                        <FlexItem>
                            {isEditing ? (
                                <ActionGroup className="pf-u-display-flex pf-u-justify-content-flex-end">
                                    <Button
                                        variant="secondary"
                                        className="pf-u-mr-sm"
                                        onClick={cancelEdit}
                                        data-testid="cancel-btn"
                                    >
                                        Cancel
                                    </Button>
                                    <Button
                                        variant="primary"
                                        data-testid="save-btn"
                                        form="system-config-edit-form"
                                        type="submit"
                                        isDisabled={!dirty || !isValid || isSubmitting}
                                        isLoading={isSubmitting}
                                    >
                                        Save
                                    </Button>
                                </ActionGroup>
                            ) : (
                                <Button
                                    variant="primary"
                                    data-testid="edit-btn"
                                    onClick={editSystemConfig}
                                >
                                    Edit
                                </Button>
                            )}
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection>
                {isEditing ? (
                    <SystemConfigForm
                        values={values}
                        onSubmitForm={onSubmitForm}
                        setFieldValue={setFieldValue}
                    />
                ) : (
                    <Details systemConfig={systemConfig} telemetryConfig={telemetryConfig} />
                )}
            </PageSection>
        </>
    );
};

const mapStateToProps = createStructuredSelector({
    systemConfig: selectors.getSystemConfig,
    telemetryConfig: selectors.getTelemetryConfig,
});

const mapDispatchToProps = {
    saveSystemConfig: actions.saveSystemConfig,
    saveTelemetryConfig: telemetryActions.saveTelemetryConfig,
};

export default connect(mapStateToProps, mapDispatchToProps)(Page);
