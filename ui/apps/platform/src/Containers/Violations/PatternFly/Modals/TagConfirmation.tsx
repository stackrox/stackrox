import React, { ReactElement, useState, memo } from 'react';
import {
    Modal,
    ModalVariant,
    Button,
    Spinner,
    Alert,
    SelectOption,
    Select,
    SelectVariant,
} from '@patternfly/react-core';
import { gql, useMutation } from '@apollo/client';
import pluralize from 'pluralize';
import { toast } from 'react-toastify';
import Raven from 'raven-js';

import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import useMultiSelect from 'hooks/useMultiSelect';
import ViolationTagsSearchAutoComplete from 'Containers/AnalystNotes/ViolationTags/ViolationTagsSearchAutoComplete';

const BULK_ADD_ALERT_TAGS = gql`
    mutation bulkAddAlertTags($resourceIds: [ID!]!, $tags: [String!]!) {
        bulkAddAlertTags(resourceIds: $resourceIds, tags: $tags)
    }
`;

type TagConfirmationProps = {
    isOpen: boolean;
    closeModal: () => void;
    cancelModal: () => void;
    selectedAlertIds: string[];
};

function TagConfirmation({
    isOpen,
    closeModal,
    cancelModal,
    selectedAlertIds,
}: TagConfirmationProps): ReactElement {
    const [tags, setTags] = useState([]);
    const { isOpen: isSelectOpen, onToggle, onSelect, onClear } = useMultiSelect(onChange, tags);
    const [addBulkTags, { loading: isLoading, error }] = useMutation(BULK_ADD_ALERT_TAGS);
    const { hasErrors } = captureGraphQLErrors([error]);

    function tagViolations() {
        addBulkTags({
            variables: { resourceIds: selectedAlertIds, tags },
        }).then(
            () => {
                toast('Tags were successfully added');
                setTags([]);
                closeModal();
            },
            (err) => Raven.captureException(err)
        );
    }

    function onChange(newTags) {
        setTags(newTags);
    }

    const modalTitle = `Add Tags for ${selectedAlertIds.length} ${pluralize(
        'Violation',
        selectedAlertIds.length
    )}`;

    return (
        <Modal
            isOpen={isOpen}
            variant={ModalVariant.small}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={tagViolations}
                    isDisabled={!tags.length}
                >
                    Confirm
                </Button>,
                <Button key="cancel" variant="link" onClick={cancelModal}>
                    Cancel
                </Button>,
            ]}
            onClose={cancelModal}
            title={modalTitle}
            data-testid="tag-confirmation-modal"
        >
            {isLoading && <Spinner isSVG />}
            {hasErrors && (
                <Alert
                    variant="warning"
                    isInline
                    className="pf-u-mb-sm"
                    title="There was an error adding tags."
                />
            )}
            <ViolationTagsSearchAutoComplete>
                {({ options = [], onInputChange }) => (
                    <Select
                        variant={SelectVariant.typeaheadMulti}
                        selections={tags}
                        onChange={onChange}
                        placeholderText="Select or create new tags."
                        onTypeaheadInputChanged={onInputChange}
                        menuAppendTo="parent"
                        isCreatable
                        onToggle={onToggle}
                        onSelect={onSelect}
                        onClear={onClear}
                        isOpen={isSelectOpen}
                    >
                        {options.map((option) => (
                            <SelectOption key={option} value={option} />
                        ))}
                    </Select>
                )}
            </ViolationTagsSearchAutoComplete>
        </Modal>
    );
}

export default memo(TagConfirmation);
