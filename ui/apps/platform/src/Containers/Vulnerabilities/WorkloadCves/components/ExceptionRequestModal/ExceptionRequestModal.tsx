import React from 'react';
import { Alert, Modal, ModalBoxBody, pluralize } from '@patternfly/react-core';
import { FormikHelpers } from 'formik';

import {
    UpdateVulnerabilityExceptionRequest,
    VulnerabilityException,
    createDeferralVulnerabilityException,
    createFalsePositiveVulnerabilityException,
    updateVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { CveExceptionRequestType } from '../../types';
import {
    DeferralValues,
    FalsePositiveValues,
    ScopeContext,
    deferralValidationSchema,
    falsePositiveValidationSchema,
    formValuesToDeferralRequest,
    formValuesToFalsePositiveRequest,
} from './utils';
import ExceptionRequestForm, { ExceptionRequestFormProps } from './ExceptionRequestForm';

export type ExceptionRequestModalOptions = {
    type: CveExceptionRequestType;
    cves: ExceptionRequestFormProps['cves'];
} | null;

export type ExceptionRequestModalProps = {
    type: CveExceptionRequestType;
    isUpdate?: boolean;
    id?: string;
    cves: ExceptionRequestFormProps['cves'];
    scopeContext: ScopeContext;
    onExceptionRequestSuccess: (vulnerabilityException: VulnerabilityException) => void;
    onClose: () => void;
};

function ExceptionRequestModal({
    type,
    isUpdate = false,
    id = '',
    cves,
    scopeContext,
    onExceptionRequestSuccess,
    onClose,
}: ExceptionRequestModalProps) {
    const cveCountText = pluralize(cves.length, 'workload CVE');
    const titleAction = isUpdate ? 'Update' : 'Request';
    const title =
        type === 'DEFERRAL'
            ? `${titleAction} deferral for ${cveCountText}`
            : `${titleAction} false positive for ${cveCountText}`;

    const createDeferralMutation = useRestMutation(createDeferralVulnerabilityException);
    const createFalsePositiveMutation = useRestMutation(createFalsePositiveVulnerabilityException);
    const updateRequestMutation = useRestMutation(updateVulnerabilityException);

    function onDeferralSubmit(formValues: DeferralValues, helpers: FormikHelpers<DeferralValues>) {
        if (formValues.expiry) {
            const payload = formValuesToDeferralRequest(formValues, formValues.expiry);
            const callbackOptions = {
                onSuccess: onExceptionRequestSuccess,
                onError: () => helpers.setSubmitting(false),
            };
            if (isUpdate) {
                const updatedPayload: UpdateVulnerabilityExceptionRequest = {
                    id,
                    comment: payload.comment,
                    deferralUpdate: {
                        cves: payload.cves,
                        expiry: payload.exceptionExpiry,
                    },
                };
                updateRequestMutation.mutate(updatedPayload, callbackOptions);
            } else {
                createDeferralMutation.mutate(payload, callbackOptions);
            }
        } else {
            helpers.setFieldError('expiry', 'An expiry is required');
        }
    }

    function onFalsePositiveSubmit(
        formValues: FalsePositiveValues,
        helpers: FormikHelpers<FalsePositiveValues>
    ) {
        const payload = formValuesToFalsePositiveRequest(formValues);
        const callbackOptions = {
            onSuccess: onExceptionRequestSuccess,
            onError: () => helpers.setSubmitting(false),
        };
        if (isUpdate) {
            const updatedPayload: UpdateVulnerabilityExceptionRequest = {
                id,
                comment: payload.comment,
                falsePositiveUpdate: {
                    cves: payload.cves,
                },
            };
            updateRequestMutation.mutate(updatedPayload, callbackOptions);
        } else {
            createFalsePositiveMutation.mutate(payload, callbackOptions);
        }
    }

    const formProps =
        type === 'DEFERRAL'
            ? {
                  formHeaderText: `CVEs will be marked as deferred after approval`,
                  commentFieldLabel: `Deferral rationale`,
                  onSubmit: onDeferralSubmit,
                  validationSchema: deferralValidationSchema,
                  showExpiryField: true,
                  showScopeField: !isUpdate,
              }
            : {
                  formHeaderText: `CVEs will be marked as false positive after approval`,
                  commentFieldLabel: `False positive rationale`,
                  onSubmit: onFalsePositiveSubmit,
                  validationSchema: falsePositiveValidationSchema,
                  showExpiryField: false,
                  showScopeField: !isUpdate,
              };

    const submissionError = createDeferralMutation.error ?? createFalsePositiveMutation.error;

    return (
        <Modal
            aria-label={title}
            title={title}
            hasNoBodyWrapper
            onClose={onClose}
            isOpen
            variant="medium"
        >
            <ModalBoxBody className="pf-u-display-flex pf-u-flex-direction-column">
                {submissionError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error submitting the exception request"
                    >
                        {getAxiosErrorMessage(submissionError)}
                    </Alert>
                )}
                <ExceptionRequestForm
                    cves={cves}
                    scopeContext={scopeContext}
                    onCancel={onClose}
                    {...formProps}
                />
            </ModalBoxBody>
        </Modal>
    );
}

export default ExceptionRequestModal;
