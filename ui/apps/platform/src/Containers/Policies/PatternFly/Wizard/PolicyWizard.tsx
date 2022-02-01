import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { FormikProvider, useFormik } from 'formik';
import { Wizard, Breadcrumb, Title, BreadcrumbItem, Divider } from '@patternfly/react-core';

import { createPolicy, savePolicy } from 'services/PoliciesService';
import { Policy } from 'types/policy.proto';
import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import { getServerPolicy } from '../policies.utils';
import { getValidationSchema } from './policyValidationSchemas';
import PolicyDetailsForm from './Step1/PolicyDetailsForm';
import PolicyBehaviorForm from './Step2/PolicyBehaviorForm';
import PolicyCriteriaForm from './Step3/PolicyCriteriaForm';
import PolicyScopeForm from './Step4/PolicyScopeForm';
import ReviewPolicyForm from './Step5/ReviewPolicyForm';

type PolicyWizardProps = {
    pageAction: ExtendedPageAction;
    policy: Policy;
    clusters: Cluster[];
    notifiers: NotifierIntegration[];
};

function PolicyWizard({
    pageAction,
    policy,
    clusters,
    notifiers,
}: PolicyWizardProps): ReactElement {
    const history = useHistory();
    const [stepId, setStepId] = useState(1);
    const [stepIdReached, setStepIdReached] = useState(1);
    const [isValidOnServer, setIsValidOnServer] = useState(false);
    const [policyErrorMessage, setPolicyErrorMessage] = useState('');
    const [isBadRequest, setIsBadRequest] = useState(false);

    const formik = useFormik({
        initialValues: policy,
        onSubmit: (values: Policy, { setSubmitting }) => {
            setPolicyErrorMessage('');
            setIsBadRequest(false);
            const serverPolicy = getServerPolicy(values);
            const request =
                pageAction === 'edit' ? savePolicy(serverPolicy) : createPolicy(serverPolicy);
            request
                .then(() => {
                    history.goBack();
                })
                .catch((error) => {
                    setPolicyErrorMessage(getAxiosErrorMessage(error));
                    if (error.response?.status === 400) {
                        setIsBadRequest(true);
                    }
                })
                .finally(() => {
                    setSubmitting(false);
                });
        },
        validateOnMount: true,
        validationSchema: getValidationSchema(stepId),
    });
    const { dirty, isSubmitting, isValid: isValidOnClient, submitForm, validateForm } = formik;

    function closeWizard(): void {
        history.goBack();
    }

    function onBack(newStep): void {
        const { id } = newStep;
        setStepId(id);
    }

    function onGoToStep(newStep): void {
        const { id } = newStep;
        // TODO Maybe prevent going forward to previously visited step if current step is not valid,
        // after having moved backward to step which was valid, but made a change which is not valid?
        // Maybe allow going backward in that situation? For example, from criteria to behavior?
        setStepId(id);
    }

    function onNext(newStep): void {
        const { id } = newStep;
        setStepId(id);
        setStepIdReached(stepIdReached < id ? id : stepIdReached);
    }

    useEffect(() => {
        // Imperative call is needed to validate when the step changes,
        // before any value has changed to cause validation.
        validateForm().catch(() => {});
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [stepId]); // but not validateForm

    return (
        <>
            <Breadcrumb className="pf-u-mb-md">
                <BreadcrumbItemLink to={policiesBasePath}>Policies</BreadcrumbItemLink>
                <BreadcrumbItem isActive>{policy?.name || 'Create policy'}</BreadcrumbItem>
            </Breadcrumb>
            <Title headingLevel="h1">{policy?.name || 'Create policy'}</Title>
            <div className="pf-u-mb-md pf-u-mt-sm">
                Design custom security policies for your environment
            </div>
            <Divider component="div" />
            <FormikProvider value={formik}>
                <Wizard
                    navAriaLabel={`${pageAction} policy steps`}
                    mainAriaLabel={`${pageAction} policy content`}
                    onClose={closeWizard}
                    onSave={submitForm}
                    steps={[
                        {
                            id: 1,
                            name: 'Policy details',
                            component: <PolicyDetailsForm />,
                            canJumpTo: stepIdReached >= 1,
                            enableNext: isValidOnClient,
                        },
                        {
                            id: 2,
                            name: 'Policy behavior',
                            component: <PolicyBehaviorForm />,
                            canJumpTo: stepIdReached >= 2,
                            enableNext: isValidOnClient,
                        },
                        {
                            id: 3,
                            name: 'Policy criteria',
                            component: <PolicyCriteriaForm />,
                            canJumpTo: stepIdReached >= 3,
                            enableNext: isValidOnClient,
                        },
                        {
                            id: 4,
                            name: 'Policy scope',
                            component: <PolicyScopeForm clusters={clusters} />,
                            canJumpTo: stepIdReached >= 4,
                            enableNext: isValidOnClient,
                        },
                        {
                            id: 5,
                            name: 'Review policy',
                            component: (
                                <ReviewPolicyForm
                                    clusters={clusters}
                                    isBadRequest={isBadRequest}
                                    notifiers={notifiers}
                                    policyErrorMessage={policyErrorMessage}
                                    setIsBadRequest={setIsBadRequest}
                                    setIsValidOnServer={setIsValidOnServer}
                                    setPolicyErrorMessage={setPolicyErrorMessage}
                                />
                            ),
                            nextButtonText: 'Save',
                            canJumpTo: stepIdReached >= 5,
                            enableNext: dirty && isValidOnServer && !isSubmitting,
                        },
                    ]}
                    onBack={onBack}
                    onGoToStep={onGoToStep}
                    onNext={onNext}
                />
            </FormikProvider>
        </>
    );
}

export default PolicyWizard;
