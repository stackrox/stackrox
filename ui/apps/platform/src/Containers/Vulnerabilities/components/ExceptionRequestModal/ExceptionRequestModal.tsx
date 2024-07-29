import React from 'react';
import { Alert, Modal, ModalBoxBody, pluralize } from '@patternfly/react-core';
import { FormikHelpers } from 'formik';
import dateFns from 'date-fns';

import {
    UpdateVulnerabilityExceptionRequest,
    VulnerabilityException,
    createDeferralVulnerabilityException,
    createFalsePositiveVulnerabilityException,
    updateVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useAnalytics, {
    WORKLOAD_CVE_DEFERRAL_EXCEPTION_REQUESTED,
    WORKLOAD_CVE_FALSE_POSITIVE_EXCEPTION_REQUESTED,
} from 'hooks/useAnalytics';
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

function normalizeExpiryForAnalytics(
    expiry: DeferralValues['expiry']
):
    | { expiryType: 'CUSTOM_DATE' | 'TIME'; expiryDays: number }
    | { expiryType: 'ALL_CVE_FIXABLE' | 'ANY_CVE_FIXABLE' | 'INDEFINITE' } {
    if (!expiry) {
        return { expiryType: 'INDEFINITE' };
    }
    const expiryType = expiry.type;

    if (expiry.type === 'CUSTOM_DATE') {
        return { expiryType, expiryDays: dateFns.differenceInDays(expiry.date, new Date()) };
    }
    if (expiry.type === 'TIME') {
        return { expiryType, expiryDays: expiry.days };
    }

    return { expiryType: expiry.type };
}

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
    const { analyticsTrack } = useAnalytics();
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
                onSuccess: (exception: VulnerabilityException) => {
                    const properties = normalizeExpiryForAnalytics(formValues.expiry);
                    analyticsTrack({
                        event: WORKLOAD_CVE_DEFERRAL_EXCEPTION_REQUESTED,
                        properties,
                    });
                    return onExceptionRequestSuccess(exception);
                },
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
            onSuccess: (exception: VulnerabilityException) => {
                analyticsTrack({
                    event: WORKLOAD_CVE_FALSE_POSITIVE_EXCEPTION_REQUESTED,
                    properties: {},
                });
                return onExceptionRequestSuccess(exception);
            },
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
            <ModalBoxBody className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                {!!submissionError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error submitting the exception request"
                        component="p"
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
