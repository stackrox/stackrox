import React, { ReactElement } from 'react';
import { Button, ModalBoxBody, ModalBoxFooter, Alert } from '@patternfly/react-core';
import { Formik } from 'formik';
import * as yup from 'yup';

import { ListPolicy } from 'types/policy.proto';
import {
    MIN_POLICY_NAME_LENGTH,
    POLICY_DUPE_ACTIONS,
    hasDuplicateIdOnly,
    checkForBlockedSubmit,
} from './PolicyImport.utils';
import DuplicatePolicyForm from './DuplicatePolicyForm';

const RESOLUTION = { resolution: '', newName: '' };

type DuplicateErrors = {
    type: string;
    incomingName: string;
    incomingId: string;
    duplicateName: string;
};

type ImportPolicyJSONErrorProps = {
    handleCancelModal: () => void;
    startImportPolicies: () => void;
    policies: ListPolicy[];
    duplicateErrors: DuplicateErrors[];
    errorMessages: string[];
    duplicateResolution: { resolution: string; newName: string };
    setDuplicateResolution: (duplicateResolution) => void;
};

function ImportPolicyJSONError({
    handleCancelModal,
    startImportPolicies,
    policies,
    duplicateErrors,
    duplicateResolution,
    setDuplicateResolution,
    errorMessages,
}: ImportPolicyJSONErrorProps): ReactElement {
    function updateResolution(key, value) {
        setDuplicateResolution({ ...duplicateResolution, [key]: value });
    }

    const showKeepBothPolicies = hasDuplicateIdOnly(duplicateErrors);
    const isBlocked = checkForBlockedSubmit({
        numPolicies: policies?.length ?? 0,
        messageType: 'error',
        hasDuplicateErrors: !!duplicateErrors,
        duplicateResolution,
    });

    return (
        <Formik
            initialValues={RESOLUTION}
            onSubmit={() => {}}
            validationSchema={yup.object({
                newName: yup.string().when('resolution', {
                    is: POLICY_DUPE_ACTIONS.RENAME,
                    then: (newNameSchema) =>
                        newNameSchema
                            .trim()
                            .min(
                                MIN_POLICY_NAME_LENGTH,
                                `A policy name must be at least ${MIN_POLICY_NAME_LENGTH} characters.`
                            ),
                }),
            })}
        >
            <>
                <ModalBoxBody>
                    Address the errors below to continue importing policies
                    <Alert
                        title="Policies already exist"
                        variant="danger"
                        className="pf-u-mt-md"
                        isInline
                    >
                        <ul>
                            {errorMessages.map((msg) => (
                                <li key={msg} className="py-2">
                                    {msg}
                                </li>
                            ))}
                        </ul>
                        {duplicateErrors && (
                            <DuplicatePolicyForm
                                updateResolution={updateResolution}
                                showKeepBothPolicies={showKeepBothPolicies}
                            />
                        )}
                    </Alert>
                </ModalBoxBody>
                <ModalBoxFooter>
                    <Button
                        key="import"
                        variant="primary"
                        onClick={startImportPolicies}
                        isDisabled={isBlocked}
                    >
                        Resume import
                    </Button>
                    <Button key="cancel" variant="link" onClick={handleCancelModal}>
                        Cancel
                    </Button>
                </ModalBoxFooter>
            </>
        </Formik>
    );
}

export default ImportPolicyJSONError;
