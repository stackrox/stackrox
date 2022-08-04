import React, { ReactElement, useState } from 'react';
import { Modal, ModalVariant, Title, TitleSizes } from '@patternfly/react-core';
import * as yup from 'yup';

import AccessScopeForm from 'Containers/AccessControl/AccessScopes/AccessScopeForm';
import {
    LabelSelectorsEditingState,
    getIsEditingLabelSelectors,
    getIsValidRules,
} from 'Containers/AccessControl/AccessScopes/accessScopes.utils';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import useFormModal from 'hooks/patternfly/useFormModal';
import { createAccessScope, AccessScope, accessScopeNew } from 'services/AccessScopesService';

export type ResourceScopeFormModalProps = {
    isOpen: boolean;
    updateResourceScopeList: (string) => void;
    onToggleResourceScopeModal: () => void;
    resourceScopes: AccessScope[];
};

function ResourceScopeFormModal({
    isOpen,
    updateResourceScopeList,
    onToggleResourceScopeModal,
    resourceScopes,
}: ResourceScopeFormModalProps): ReactElement {
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    // Disable Save button while editing label selectors.
    const [labelSelectorsEditingState, setLabelSelectorsEditingState] =
        useState<LabelSelectorsEditingState>({
            clusterLabelSelectors: -1,
            namespaceLabelSelectors: -1,
        });

    const formInitialValues = { ...accessScopeNew };

    const { formik, message, onHandleSubmit, onHandleCancel } = useFormModal<AccessScope>({
        initialValues: formInitialValues,
        validationSchema: yup.object({
            name: yup
                .string()
                .required()
                .test(
                    'non-unique-name',
                    'Another access scope already has this name',
                    // Return true if current input name is initial name
                    // or no other access scope already has this name.
                    (nameInput) => resourceScopes.every(({ name }) => nameInput !== name)
                ),
            description: yup.string(),
        }),
        onSendRequest: onSave,
        onCompleteRequest,
        onCancel: onToggleResourceScopeModal,
    });

    function onSave(values: AccessScope): Promise<FormResponseMessage> {
        setAlertSubmit(null);

        return createAccessScope(values).then((entityCreated) => {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            return { isError: false, message: '', data: entityCreated ?? null };
        });
    }

    function onCompleteRequest(response) {
        updateResourceScopeList(response?.data?.id ?? '');
        onToggleResourceScopeModal();
    }

    const title = 'Create resource scope';
    const descriptionText =
        'Add predefined sets of authorized Kubernetes resources that will be used in a report.';

    const header = (
        <>
            <Title id="custom-header-label" headingLevel="h1" size={TitleSizes.xl}>
                {title}
            </Title>
            <p className="pf-u-pt-sm">{descriptionText}</p>
        </>
    );

    const { isSubmitting, dirty, isValid, values } = formik;

    /*
     * A label selector or set requirement is temporarily invalid when it is added,
     * before its first requirement or value has been added.
     */
    const isValidRules = getIsValidRules(values.rules);

    return (
        <Modal
            aria-label="Create new resource scope"
            variant={ModalVariant.large}
            header={header}
            isOpen={isOpen}
            onClose={onHandleCancel}
            actions={[
                <FormSaveButton
                    onSave={onHandleSubmit}
                    isSubmitting={isSubmitting}
                    isTesting={false}
                    isDisabled={
                        !dirty ||
                        !isValid ||
                        !isValidRules ||
                        getIsEditingLabelSelectors(labelSelectorsEditingState) ||
                        isSubmitting
                    }
                >
                    Create resource scope
                </FormSaveButton>,
                <FormCancelButton onCancel={onHandleCancel}>Cancel</FormCancelButton>,
            ]}
        >
            <FormMessage message={message} />
            <AccessScopeForm
                hasAction
                alertSubmit={alertSubmit}
                formik={formik}
                labelSelectorsEditingState={labelSelectorsEditingState}
                setLabelSelectorsEditingState={setLabelSelectorsEditingState}
            />
        </Modal>
    );
}

export default ResourceScopeFormModal;
