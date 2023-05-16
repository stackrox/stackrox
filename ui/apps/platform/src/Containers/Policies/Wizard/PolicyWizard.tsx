import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { FormikProvider, useFormik } from 'formik';
import {
    Wizard,
    Breadcrumb,
    Title,
    BreadcrumbItem,
    Divider,
    PageSection,
} from '@patternfly/react-core';

import { createPolicy, savePolicy } from 'services/PoliciesService';
import { fetchAlertCount } from 'services/AlertsService';
import { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { policiesBasePath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import { getServerPolicy } from '../policies.utils';
import { getValidationSchema } from './policyValidationSchemas';
import PolicyDetailsForm from './Step1/PolicyDetailsForm';
import PolicyBehaviorForm from './Step2/PolicyBehaviorForm';
import PolicyCriteriaForm from './Step3/PolicyCriteriaForm';
import PolicyScopeForm from './Step4/PolicyScopeForm';
import ReviewPolicyForm from './Step5/ReviewPolicyForm';

import './PolicyWizard.css';

type PolicyWizardProps = {
    pageAction: ExtendedPageAction;
    policy: ClientPolicy;
};

function PolicyWizard({ pageAction, policy }: PolicyWizardProps): ReactElement {
    const history = useHistory();
    const [stepId, setStepId] = useState(1);
    const [stepIdReached, setStepIdReached] = useState(1);
    const [isValidOnServer, setIsValidOnServer] = useState(false);
    const [policyErrorMessage, setPolicyErrorMessage] = useState('');
    const [isBadRequest, setIsBadRequest] = useState(false);
    const [hasActiveViolations, setHasActiveViolations] = useState(false);

    const formik = useFormik({
        initialValues: policy,
        onSubmit: (values: ClientPolicy, { setSubmitting }) => {
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
    const {
        dirty,
        isSubmitting,
        isValid: isValidOnClient,
        submitForm,
        validateForm,
        values,
    } = formik;

    function closeWizard(): void {
        history.goBack();
    }

    function scrollToTop() {
        // wizard does not by default scroll to top of body when navigating to a step
        document.getElementsByClassName('pf-c-wizard__main')[0].scrollTop = 0;
    }

    function onBack(newStep): void {
        const { id } = newStep;
        setStepId(id);
        scrollToTop();
    }

    function onGoToStep(newStep): void {
        const { id } = newStep;
        // TODO Maybe prevent going forward to previously visited step if current step is not valid,
        // after having moved backward to step which was valid, but made a change which is not valid?
        // Maybe allow going backward in that situation? For example, from criteria to behavior?
        setStepId(id);
        scrollToTop();
    }

    function onNext(newStep): void {
        const { id } = newStep;
        setStepId(id);
        setStepIdReached(stepIdReached < id ? id : stepIdReached);
        scrollToTop();
    }

    useEffect(() => {
        // Imperative call is needed to validate when the step changes,
        // before any value has changed to cause validation.
        validateForm().catch(() => {});
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [stepId]); // but not validateForm

    useEffect(() => {
        if (policy?.name) {
            const queryObj = {
                Policy: policy.name || '',
            };
            const { request: countRequest } = fetchAlertCount(queryObj);
            // eslint-disable-next-line no-void
            void countRequest.then((counts) => {
                if (counts > 0) {
                    setHasActiveViolations(true);
                } else {
                    setHasActiveViolations(false);
                }
            });
        }
    }, [policy]);

    const canJumpToAny = pageAction === 'clone' || pageAction === 'edit';

    return (
        <>
            <PageSection variant="light" isFilled id="policy-page" className="pf-u-pb-0">
                <Breadcrumb className="pf-u-mb-md">
                    <BreadcrumbItemLink to={policiesBasePath}>Policies</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{policy?.name || 'Create policy'}</BreadcrumbItem>
                </Breadcrumb>
                <Title headingLevel="h1">{policy?.name || 'Create policy'}</Title>
                <div className="pf-u-mb-md pf-u-mt-sm">
                    Design custom security policies for your environment
                </div>
                <Divider component="div" />
            </PageSection>
            <PageSection
                variant="light"
                isFilled
                hasOverflowScroll
                padding={{ default: 'noPadding' }}
                className="pf-u-h-100"
            >
                <FormikProvider value={formik}>
                    <Wizard
                        navAriaLabel={`${pageAction} policy steps`}
                        mainAriaLabel={`${pageAction} policy content`}
                        onClose={closeWizard}
                        onSave={submitForm}
                        hasNoBodyPadding
                        steps={[
                            {
                                id: 1,
                                name: 'Policy details',
                                component: (
                                    <PolicyDetailsForm
                                        id={values.id}
                                        mitreVectorsLocked={values.mitreVectorsLocked}
                                    />
                                ),
                                canJumpTo: canJumpToAny || stepIdReached >= 1,
                                enableNext: isValidOnClient,
                            },
                            {
                                id: 2,
                                name: 'Policy behavior',
                                component: (
                                    <PolicyBehaviorForm hasActiveViolations={hasActiveViolations} />
                                ),
                                canJumpTo: canJumpToAny || stepIdReached >= 2,
                                enableNext: isValidOnClient,
                            },
                            {
                                id: 3,
                                name: 'Policy criteria',
                                component: (
                                    <PolicyCriteriaForm hasActiveViolations={hasActiveViolations} />
                                ),
                                canJumpTo: canJumpToAny || stepIdReached >= 3,
                                enableNext: isValidOnClient,
                            },
                            {
                                id: 4,
                                name: 'Policy scope',
                                component: <PolicyScopeForm />,
                                canJumpTo: canJumpToAny || stepIdReached >= 4,
                                enableNext: isValidOnClient,
                            },
                            {
                                id: 5,
                                name: 'Review policy',
                                component: (
                                    <ReviewPolicyForm
                                        isBadRequest={isBadRequest}
                                        policyErrorMessage={policyErrorMessage}
                                        setIsBadRequest={setIsBadRequest}
                                        setIsValidOnServer={setIsValidOnServer}
                                        setPolicyErrorMessage={setPolicyErrorMessage}
                                    />
                                ),
                                nextButtonText: 'Save',
                                canJumpTo: canJumpToAny || stepIdReached >= 5,
                                enableNext:
                                    (dirty || pageAction === 'clone') &&
                                    isValidOnServer &&
                                    !isSubmitting,
                            },
                        ]}
                        onBack={onBack}
                        onGoToStep={onGoToStep}
                        onNext={onNext}
                    />
                </FormikProvider>
            </PageSection>
        </>
    );
}

export default PolicyWizard;
