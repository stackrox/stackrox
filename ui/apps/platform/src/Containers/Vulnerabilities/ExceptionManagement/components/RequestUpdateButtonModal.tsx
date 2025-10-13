import React from 'react';
import { Button } from '@patternfly/react-core';

import { VulnerabilityException } from 'services/VulnerabilityExceptionService';

import useExceptionRequestModal from '../../hooks/useExceptionRequestModal';
import CompletedExceptionRequestModal from '../../components/ExceptionRequestModal/CompletedExceptionRequestModal';
import ExceptionRequestModal from '../../components/ExceptionRequestModal/ExceptionRequestModal';
import useRequestCVEsDetails from '../hooks/useRequestCVEsDetails';

type RequestUpdateButtonModalProps = {
    exception: VulnerabilityException;
    onSuccess: (vulnerabilityException: VulnerabilityException) => void;
};

function RequestUpdateButtonModal({ exception, onSuccess }: RequestUpdateButtonModalProps) {
    const { id, targetState, scope } = exception;
    const { registry, remote, tag } = scope.imageScope;

    const { isLoading: isRequestCVEsDetailsLoading, requestCVEsDetails } =
        useRequestCVEsDetails(exception);

    const { exceptionRequestModalOptions, completedException, showModal, closeModals } =
        useExceptionRequestModal();

    function openModal() {
        showModal({
            type: targetState === 'DEFERRED' ? 'DEFERRAL' : 'FALSE_POSITIVE',
            cves: requestCVEsDetails,
        });
    }

    const isGlobalScope = registry === '.*' && remote === '.*' && tag === '.*';

    return (
        <>
            <Button variant="primary" onClick={openModal} disabled={isRequestCVEsDetailsLoading}>
                Update request
            </Button>
            {exceptionRequestModalOptions && (
                <ExceptionRequestModal
                    cves={exceptionRequestModalOptions.cves}
                    isUpdate
                    id={id}
                    type={exceptionRequestModalOptions.type}
                    scopeContext={isGlobalScope ? 'GLOBAL' : { imageName: scope.imageScope }}
                    onExceptionRequestSuccess={(exception) => {
                        showModal({ type: 'COMPLETION', exception });
                        return onSuccess(exception);
                    }}
                    onClose={closeModals}
                />
            )}
            {completedException && (
                <CompletedExceptionRequestModal
                    isUpdate
                    exceptionRequest={completedException}
                    onClose={closeModals}
                />
            )}
        </>
    );
}

export default RequestUpdateButtonModal;
