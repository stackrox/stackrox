import React from 'react';
import { Alert, Modal, ModalBoxBody, pluralize } from '@patternfly/react-core';
import { FormikHelpers } from 'formik';

import {
    BaseVulnerabilityException,
    createDeferralVulnerabilityException,
    createFalsePositiveVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { CveExceptionRequestType } from '../../types';
import {
    DeferralValues,
    FalsePositiveValues,
    ScopeContext,
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
    cves: ExceptionRequestFormProps['cves'];
    scopeContext: ScopeContext;
    onExceptionRequestSuccess: (vulnerabilityException: BaseVulnerabilityException) => void;
    onClose: () => void;
};

function ExceptionRequestModal({
    type,
    cves,
    scopeContext,
    onExceptionRequestSuccess,
    onClose,
}: ExceptionRequestModalProps) {
    const cveCountText = pluralize(cves.length, 'workload CVE');
    const title =
        type === 'DEFERRAL'
            ? `Request deferral for ${cveCountText}`
            : `Mark ${cveCountText} as false positive`;

    const createDeferralMutation = useRestMutation(createDeferralVulnerabilityException);
    const createFalsePositiveMutation = useRestMutation(createFalsePositiveVulnerabilityException);

    function onDeferralSubmit(formValues: DeferralValues, helpers: FormikHelpers<DeferralValues>) {
        if (formValues.expiry) {
            const payload = formValuesToDeferralRequest(formValues, formValues.expiry);
            createDeferralMutation.mutate(payload, {
                onSuccess: onExceptionRequestSuccess,
                onError: () => helpers.setSubmitting(false),
            });
        } else {
            helpers.setFieldError('expiry', 'An expiry is required');
        }
    }

    function onFalsePositiveSubmit(
        formValues: FalsePositiveValues,
        helpers: FormikHelpers<FalsePositiveValues>
    ) {
        const payload = formValuesToFalsePositiveRequest(formValues);
        createFalsePositiveMutation.mutate(payload, {
            onSuccess: onExceptionRequestSuccess,
            onError: () => helpers.setSubmitting(false),
        });
    }

    const formProps =
        type === 'DEFERRAL'
            ? {
                  formHeaderText: `CVEs will be marked as deferred after approval`,
                  commentFieldLabel: `Deferral rationale`,
                  onSubmit: onDeferralSubmit,
                  showExpiryField: true,
              }
            : {
                  formHeaderText: `CVEs will be marked as false positive after approval`,
                  commentFieldLabel: `False positive rationale`,
                  onSubmit: onFalsePositiveSubmit,
                  showExpiryField: false,
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
