import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { FormikProvider, useFormik } from 'formik';
import { Wizard, Breadcrumb, Title, BreadcrumbItem, Divider } from '@patternfly/react-core';

import { Policy } from 'types/policy.proto';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { ExtendedPageAction } from 'utils/queryStringUtils';

import { getValidationSchema } from './policyValidationSchemas';
import PolicyDetailsForm from './Step1/PolicyDetailsForm';
import PolicyBehaviorForm from './Step2/PolicyBehaviorForm';
import PolicyCriteriaForm from './Step3/PolicyCriteriaForm';
import ReviewPolicyForm from './Step5/ReviewPolicyForm';

type PolicyWizardProps = {
    pageAction: ExtendedPageAction;
    policy: Policy;
};

function PolicyWizard({ pageAction, policy }: PolicyWizardProps): ReactElement {
    const history = useHistory();
    const [stepId, setStepId] = useState(1);
    const [stepIdReached, setStepIdReached] = useState(1);

    const formik = useFormik({
        initialValues: policy,
        onSubmit: () => {},
        validateOnMount: true,
        validationSchema: getValidationSchema(stepId),
    });
    const { isValid, validateForm } = formik;

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
                    steps={[
                        {
                            id: 1,
                            name: 'Policy details',
                            component: <PolicyDetailsForm />,
                            canJumpTo: stepIdReached >= 1,
                            enableNext: isValid,
                        },
                        {
                            id: 2,
                            name: 'Policy behavior',
                            component: <PolicyBehaviorForm />,
                            canJumpTo: stepIdReached >= 2,
                            enableNext: isValid,
                        },
                        {
                            id: 3,
                            name: 'Policy criteria',
                            component: <PolicyCriteriaForm />,
                            canJumpTo: stepIdReached >= 3,
                            enableNext: isValid,
                        },
                        {
                            id: 4,
                            name: 'Policy scope',
                            component: <div>PolicyScopeForm</div>,
                            canJumpTo: stepIdReached >= 4,
                            enableNext: isValid,
                        },
                        {
                            id: 5,
                            name: 'Review policy',
                            component: <ReviewPolicyForm />,
                            nextButtonText: 'Save',
                            canJumpTo: stepIdReached >= 5,
                            enableNext: isValid,
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
