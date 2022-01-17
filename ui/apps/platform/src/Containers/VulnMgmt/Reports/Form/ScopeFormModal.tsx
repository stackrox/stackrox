import React, { ReactElement, useState } from 'react';
import {
    Button,
    ButtonVariant,
    Modal,
    ModalVariant,
    Title,
    TitleSizes,
} from '@patternfly/react-core';

import AccessScopeForm from 'Containers/AccessControl/AccessScopes/AccessScopeForm';
import { accessScopeNew } from 'Containers/AccessControl/AccessScopes/AccessScopes';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { AccessScope } from 'services/AccessScopesService';

export type ScopeFormValues = {
    imageAppliesTo: string;
    comment: string;
};

export type ScopeFormModalProps = {
    accessScopes: AccessScope[];
    isOpen: boolean;
    onSendRequest: (values: ScopeFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancelScopeModal: () => void;
};

function ScopeFormModal({
    accessScopes = [],
    isOpen,
    onSendRequest,
    onCompleteRequest,
    onCancelScopeModal,
}: ScopeFormModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);

    function onHandleSubmit() {
        console.log('onHandleSubmit');
    }

    function onCancelHandler() {
        setMessage(null);
        onCancelScopeModal();
    }

    function placeholderPromiseHandler() {
        return Promise.resolve(null);
    }

    const title = 'Create resource scope';

    const header = (
        <>
            <Title id="custom-header-label" headingLevel="h1" size={TitleSizes.xl}>
                {title}
            </Title>
            <p className="pf-u-pt-sm">
                Add predefined sets of Kubernetes resources that users should be able to access.
            </p>
        </>
    );

    return (
        <Modal
            variant={ModalVariant.default}
            header={header}
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button
                    key="save-scope"
                    variant={ButtonVariant.primary}
                    onClick={onHandleSubmit}
                    isDisabled={false}
                    isLoading={false}
                >
                    Save resource scope
                </Button>,
                <Button key="cancel-modal" variant={ButtonVariant.link} onClick={onCancelHandler}>
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <AccessScopeForm
                isActionable
                action="create"
                accessScope={accessScopeNew}
                accessScopes={accessScopes}
                handleCancel={onCancelHandler}
                handleEdit={onCancelHandler}
                handleSubmit={placeholderPromiseHandler}
            />
        </Modal>
    );
}

export default ScopeFormModal;
