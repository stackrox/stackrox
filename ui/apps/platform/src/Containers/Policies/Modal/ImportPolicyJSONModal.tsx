import React, { useState } from 'react';
import type { ReactElement } from 'react';
import { Modal } from '@patternfly/react-core';

import { importPolicies } from 'services/PoliciesService';
import type { Policy } from 'types/policy.proto';
import {
    parsePolicyImportErrors,
    getResolvedPolicies,
    getErrorMessages,
    checkDupeOnlyErrors,
} from './PolicyImport.utils';
import type { PolicyImportError, PolicyResolution } from './PolicyImport.utils';
import ImportPolicyJSONSuccess from './ImportPolicyJSONSuccess';
import ImportPolicyJSONModalError from './ImportPolicyJSONModalError';
import ImportPolicyJSONUpload from './ImportPolicyJSONUpload';

const RESOLUTION = { resolution: null, newName: '' };

type ImportPolicyJSONModalType = 'upload' | 'success' | 'error';

type ImportPolicyJSONModalProps = {
    cancelModal: () => void;
    isOpen: boolean;
    fetchPoliciesWithQuery: () => void;
};

function ImportPolicyJSONModal({
    cancelModal,
    isOpen,
    fetchPoliciesWithQuery,
}: ImportPolicyJSONModalProps): ReactElement {
    const [policies, setPolicies] = useState<Policy[]>([]);
    const [duplicateErrors, setDuplicateErrors] = useState<PolicyImportError[]>([]);
    const [duplicateResolution, setDuplicateResolution] = useState<PolicyResolution>(RESOLUTION);
    const [modalType, setModalType] = useState<ImportPolicyJSONModalType>('upload');
    const [errorMessages, setErrorMessages] = useState<string[]>([]);

    function startImportPolicies() {
        // Note: this only resolves errors on one policy for MVP,
        // see decision in comment on Jira story, https://issues.redhat.com/browse/ROX-4409
        const [policiesToImport, metadata] = getResolvedPolicies(
            policies,
            duplicateErrors,
            duplicateResolution
        );
        importPolicies(policiesToImport, metadata)
            .then((response) => {
                if (response.allSucceeded) {
                    setModalType('success');

                    // TODO: multiple policies import will be handled in
                    // https://issues.redhat.com/browse/ROX-8613
                    setPolicies([response.responses[0].policy]);
                    setTimeout(() => {
                        handleCancelModal();
                        fetchPoliciesWithQuery();
                    }, 4000);
                } else {
                    const errors = parsePolicyImportErrors(response?.responses);
                    const onlyHasDupeErrors = checkDupeOnlyErrors(errors);
                    if (policiesToImport.length === 1 && onlyHasDupeErrors) {
                        setDuplicateErrors(errors[0]);
                    }

                    // the errors array in the response is a single-element array,
                    //     that contains an array with as many elements as there are policies in the import file
                    //     and each of those elements is an array of error objeccts
                    // hence, we use .flat() to un-ravel that structure to get all the errors
                    const errorMessageArray = getErrorMessages(errors.flat()).map(({ msg }) => msg);

                    setErrorMessages(errorMessageArray);
                    setModalType('error');
                }
            })
            .catch((err) => {
                setErrorMessages([`A network error occurred: ${err.message as string}`]);
                setModalType('error');
            });
    }

    function handleCancelModal() {
        setPolicies([]);
        setModalType('upload');
        setErrorMessages([]);
        cancelModal();
    }

    return (
        <Modal
            title="Import policy JSON"
            isOpen={isOpen}
            variant="small"
            onClose={handleCancelModal}
            data-testid="import-policy-modal"
            aria-label="Import policy"
            hasNoBodyWrapper
        >
            {modalType === 'upload' && (
                <ImportPolicyJSONUpload
                    cancelModal={handleCancelModal}
                    startImportPolicies={startImportPolicies}
                    setPolicies={setPolicies}
                    policies={policies}
                />
            )}
            {modalType === 'error' && (
                <ImportPolicyJSONModalError
                    handleCancelModal={handleCancelModal}
                    policies={policies}
                    startImportPolicies={startImportPolicies}
                    duplicateErrors={duplicateErrors}
                    errorMessages={errorMessages}
                    duplicateResolution={duplicateResolution}
                    setDuplicateResolution={setDuplicateResolution}
                />
            )}
            {modalType === 'success' && (
                <ImportPolicyJSONSuccess policies={policies} handleCloseModal={handleCancelModal} />
            )}
        </Modal>
    );
}

export default ImportPolicyJSONModal;
