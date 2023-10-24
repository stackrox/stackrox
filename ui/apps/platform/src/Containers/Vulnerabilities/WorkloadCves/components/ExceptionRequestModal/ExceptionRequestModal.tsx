import React from 'react';
import { Alert, Modal, ModalBoxBody, pluralize } from '@patternfly/react-core';
import { FormikHelpers } from 'formik';

import {
    BaseVulnerabilityException,
    createDeferralVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { CveExceptionRequestType } from '../../types';
import { DeferralValues, ScopeContext, formValuesToDeferralRequest } from './utils';
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

    const { mutate, error: deferralError } = useRestMutation(createDeferralVulnerabilityException);

    function onDeferralSubmit(formValues: DeferralValues, helpers: FormikHelpers<DeferralValues>) {
        if (formValues.expiry) {
            const payload = formValuesToDeferralRequest(formValues, formValues.expiry);
            mutate(payload, {
                onSuccess: onExceptionRequestSuccess,
                onError: () => helpers.setSubmitting(false),
            });
        } else {
            helpers.setFieldError('expiry', 'An expiry is required');
        }
    }

    const submissionError = deferralError;

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
                {type === 'DEFERRAL' && (
                    <ExceptionRequestForm
                        cves={cves}
                        scopeContext={scopeContext}
                        onSubmit={onDeferralSubmit}
                        onCancel={onClose}
                        showExpiryField
                        formHeaderText="CVEs will be marked as deferred after approval"
                        commentFieldLabel="Deferral rationale"
                    />
                )}
            </ModalBoxBody>
        </Modal>
    );
}

export default ExceptionRequestModal;
