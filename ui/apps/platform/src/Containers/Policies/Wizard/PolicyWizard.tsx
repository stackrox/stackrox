import React, { ReactElement, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { FormikProvider, useFormik } from 'formik';
import {
    Alert,
    Breadcrumb,
    Title,
    BreadcrumbItem,
    Divider,
    PageSection,
    Wizard,
    WizardStep,
    WizardStepType,
} from '@patternfly/react-core';

import { createPolicy, savePolicy } from 'services/PoliciesService';
import { fetchAlertCount } from 'services/AlertsService';
import { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { policiesBasePath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import {
    POLICY_BEHAVIOR_ACTIONS_ID,
    POLICY_BEHAVIOR_ID,
    POLICY_BEHAVIOR_SCOPE_ID,
    POLICY_DEFINITION_DETAILS_ID,
    POLICY_DEFINITION_ID,
    POLICY_DEFINITION_LIFECYCLE_ID,
    POLICY_DEFINITION_RULES_ID,
    POLICY_REVIEW_ID,
} from '../policies.constants';
import { getServerPolicy, isExternalPolicy } from '../policies.utils';
import { getValidationSchema } from './policyValidationSchemas';
import PolicyDetailsForm from './Step1/PolicyDetailsForm';
import PolicyBehaviorForm from './Step2/PolicyBehaviorForm';
import PolicyCriteriaForm from './Step3/PolicyCriteriaForm';
import PolicyScopeForm from './Step4/PolicyScopeForm';
import PolicyActionsForm from './Step5/PolicyActionsForm';
import ReviewPolicyForm from './Step6/ReviewPolicyForm';

import './PolicyWizard.css';

type PolicyWizardProps = {
    pageAction: ExtendedPageAction;
    policy: ClientPolicy;
};

function PolicyWizard({ pageAction, policy }: PolicyWizardProps): ReactElement {
    const navigate = useNavigate();
    const [stepId, setStepId] = useState<number | string>(POLICY_DEFINITION_DETAILS_ID);
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
                    navigate(-1);
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
        navigate(-1);
    }

    function scrollToTop() {
        // wizard does not by default scroll to top of body when navigating to a step
        document.getElementsByClassName('pf-v5-c-wizard__main')[0].scrollTop = 0;
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

    function onStepChange(_event, currentStep: WizardStepType): void {
        setStepId(currentStep.id);
        scrollToTop();
    }

    const canJumpToAny = pageAction === 'clone' || pageAction === 'edit';

    return (
        <>
            {isExternalPolicy(policy) && (
                <Alert isInline title="Externally managed policy" component="p" variant="warning">
                    You are editing a policy that is managed externally. Any local changes to this
                    policy will be automatically overwritten during the next resync.
                </Alert>
            )}
            <PageSection variant="light" isFilled id="policy-page" className="pf-v5-u-pb-0">
                <Breadcrumb className="pf-v5-u-mb-md">
                    <BreadcrumbItemLink to={policiesBasePath}>Policies</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{policy?.name || 'Create policy'}</BreadcrumbItem>
                </Breadcrumb>
                <Title headingLevel="h1">{policy?.name || 'Create policy'}</Title>
                <div className="pf-v5-u-mb-md pf-v5-u-mt-sm">
                    Design custom security policies for your environment
                </div>
                <Divider component="div" />
            </PageSection>
            <PageSection
                variant="light"
                isFilled
                hasOverflowScroll
                padding={{ default: 'noPadding' }}
                className="pf-v5-u-h-100"
            >
                <FormikProvider value={formik}>
                    <Wizard
                        navAriaLabel={`${pageAction} policy steps`}
                        onClose={closeWizard}
                        onSave={submitForm}
                        isVisitRequired={!canJumpToAny}
                        onStepChange={onStepChange}
                    >
                        <WizardStep
                            name="Policy definition"
                            id={POLICY_DEFINITION_ID}
                            isExpandable
                            steps={[
                                <WizardStep
                                    name="Details"
                                    id={POLICY_DEFINITION_DETAILS_ID}
                                    key={POLICY_DEFINITION_DETAILS_ID}
                                    body={{ hasNoPadding: true }}
                                    footer={{ isNextDisabled: !isValidOnClient }}
                                >
                                    <PolicyDetailsForm
                                        id={values.id}
                                        mitreVectorsLocked={values.mitreVectorsLocked}
                                    />
                                </WizardStep>,
                                <WizardStep
                                    name="Lifecycle"
                                    id={POLICY_DEFINITION_LIFECYCLE_ID}
                                    key={POLICY_DEFINITION_LIFECYCLE_ID}
                                    body={{ hasNoPadding: true }}
                                    footer={{ isNextDisabled: !isValidOnClient }}
                                >
                                    <PolicyBehaviorForm hasActiveViolations={hasActiveViolations} />
                                </WizardStep>,
                                <WizardStep
                                    name="Rules"
                                    id={POLICY_DEFINITION_RULES_ID}
                                    key={POLICY_DEFINITION_RULES_ID}
                                    body={{ hasNoPadding: true }}
                                    footer={{ isNextDisabled: !isValidOnClient }}
                                >
                                    <PolicyCriteriaForm hasActiveViolations={hasActiveViolations} />
                                </WizardStep>,
                            ]}
                        />
                        <WizardStep
                            name="Policy behavior"
                            id={POLICY_BEHAVIOR_ID}
                            isExpandable
                            steps={[
                                <WizardStep
                                    name="Scope"
                                    id={POLICY_BEHAVIOR_SCOPE_ID}
                                    key={POLICY_BEHAVIOR_SCOPE_ID}
                                    body={{ hasNoPadding: true }}
                                    footer={{ isNextDisabled: !isValidOnClient }}
                                >
                                    <PolicyScopeForm />
                                </WizardStep>,
                                <WizardStep
                                    name="Actions"
                                    id={POLICY_BEHAVIOR_ACTIONS_ID}
                                    key={POLICY_BEHAVIOR_ACTIONS_ID}
                                    body={{ hasNoPadding: true }}
                                    footer={{ isNextDisabled: !isValidOnClient }}
                                >
                                    <PolicyActionsForm />
                                </WizardStep>,
                            ]}
                        />
                        <WizardStep
                            name="Review"
                            id={POLICY_REVIEW_ID}
                            body={{ hasNoPadding: true }}
                            footer={{
                                nextButtonText: 'Save',
                                isNextDisabled: !(
                                    (dirty || pageAction === 'clone') &&
                                    isValidOnServer &&
                                    !isSubmitting
                                ),
                            }}
                        >
                            <ReviewPolicyForm
                                isBadRequest={isBadRequest}
                                policyErrorMessage={policyErrorMessage}
                                setIsBadRequest={setIsBadRequest}
                                setIsValidOnServer={setIsValidOnServer}
                                setPolicyErrorMessage={setPolicyErrorMessage}
                            />
                        </WizardStep>
                    </Wizard>
                </FormikProvider>
            </PageSection>
        </>
    );
}

export default PolicyWizard;
