import React, { ReactElement } from 'react';
import {
    Alert,
    Flex,
    Form,
    FormGroup,
    Radio,
    Select,
    SelectOption,
    TextInput,
    Title,
} from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

import {
    InitBundleWizardFormikProps,
    installationOptions,
    platformOptions,
} from './InitBundleWizard.utils';

export type InitBundleWizardStep1Props = {
    formik: InitBundleWizardFormikProps;
};

function InitBundleWizardStep1({ formik }: InitBundleWizardStep1Props): ReactElement {
    const { errors, handleBlur, setFieldValue, touched, values } = formik;
    const installationToggle = useSelectToggle();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel="h2">Select options</Title>
            <Form>
                <FormLabelGroup
                    fieldId="name"
                    label="Name"
                    isRequired
                    errors={errors}
                    touched={touched}
                >
                    <TextInput
                        type="text"
                        id="name"
                        name="name"
                        isRequired
                        value={values.name}
                        onBlur={handleBlur}
                        onChange={onChange}
                    />
                </FormLabelGroup>
                <Alert
                    variant="info"
                    isInline
                    title="You can use one bundle for multiple clusters on the same platform with the same installation method."
                    component="p"
                />
                <FormGroup fieldId="platform" label="Platform of secured clusters" isRequired>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        {Object.entries(platformOptions).map(([platformKey, platformLabel]) => (
                            <Radio
                                key={platformKey}
                                name={platformKey}
                                value={platformKey}
                                onChange={() => {
                                    setFieldValue('platform', platformKey);
                                    if (platformKey !== 'OpenShift') {
                                        setFieldValue('installation', 'Helm');
                                    }
                                }}
                                label={platformLabel}
                                id={platformKey}
                                isChecked={values.platform === platformKey}
                            />
                        ))}
                    </Flex>
                </FormGroup>
                <FormGroup
                    fieldId="installation"
                    label="Installation method for secured cluster services"
                    isRequired
                >
                    <Select
                        variant="single"
                        toggleAriaLabel="Installation method menu toggle"
                        aria-label="Select an installation method"
                        isDisabled={values.platform !== 'OpenShift'}
                        onToggle={installationToggle.onToggle}
                        onSelect={(_event, value) => {
                            setFieldValue('installation', value);
                        }}
                        selections={values.installation}
                        isOpen={installationToggle.isOpen}
                        // className="pf-u-flex-basis-0"
                    >
                        {Object.entries(installationOptions)
                            .filter(
                                ([installationKey]) =>
                                    values.platform === 'OpenShift' ||
                                    installationKey !== 'Operator'
                            )
                            .map(([installationKey, installationLabel]) => (
                                <SelectOption key={installationKey} value={installationKey}>
                                    {installationLabel}
                                </SelectOption>
                            ))}
                    </Select>
                </FormGroup>
            </Form>
        </Flex>
    );
}

export default InitBundleWizardStep1;
