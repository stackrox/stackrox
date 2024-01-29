import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Divider, PageSection, Wizard } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import { generateClusterInitBundle } from 'services/ClustersService'; // ClusterInitBundle
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundlesHeader from './InitBundlesHeader';
import { InitBundleWizardValues, downloadBundle, initialValues } from './InitBundleWizard.utils';
import InitBundleWizardStep1 from './InitBundleWizardStep1';
import InitBundleWizardStep2 from './InitBundleWizardStep2';

// https://github.com/stackrox/stackrox/blob/master/central/clusterinit/backend/validation.go#L11
const nameValidatorRegExp = /^[a-zA-Z0-9._-]+$/;

const validationSchema: yup.ObjectSchema<InitBundleWizardValues> = yup.object().shape({
    name: yup
        .string()
        .trim()
        .matches(
            nameValidatorRegExp,
            'Name can have only the following characters: letters, digits, period, underscore, hyphen (but no spaces)'
        )
        .required('Bundle name is required'),
    installation: yup.string().trim().required(), // Select
    platform: yup.string().trim().required(), // Radio
});

function InitBundleWizard(): ReactElement {
    const history = useHistory();
    const [errorMessage, setErrorMessage] = useState('');
    const formik = useFormik({
        initialValues,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(true);
            const { installation, name } = values;
            generateClusterInitBundle({ name })
                .then(({ response }) => {
                    setErrorMessage('');
                    downloadBundle(installation, response); // TODO try catch?
                    setSubmitting(false);
                    history.goBack(); // to table
                })
                .catch((error) => {
                    setErrorMessage(getAxiosErrorMessage(error));
                    setSubmitting(false);
                });
        },
        validateOnMount: true, // disable Next when Name is empty
        validationSchema,
    });
    const { isSubmitting, isValid, submitForm } = formik;

    return (
        <>
            <InitBundlesHeader title="Create bundle" />
            <Divider component="div" />
            <PageSection
                variant="light"
                isFilled
                hasOverflowScroll
                padding={{ default: 'noPadding' }}
                className="pf-u-h-100"
            >
                <Wizard
                    onClose={() => {
                        history.goBack();
                    }}
                    onSave={submitForm}
                    steps={[
                        {
                            id: 1,
                            name: 'Select options',
                            component: <InitBundleWizardStep1 formik={formik} />,
                            enableNext: isValid,
                        },
                        {
                            id: 2,
                            name: 'Download bundle',
                            component: (
                                <InitBundleWizardStep2
                                    errorMessage={errorMessage}
                                    formik={formik}
                                />
                            ),
                            nextButtonText: 'Download',
                            canJumpTo: isValid,
                            enableNext: isValid && !isSubmitting,
                        },
                    ]}
                />
            </PageSection>
        </>
    );
}

export default InitBundleWizard;
