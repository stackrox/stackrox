import React, { useState } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useMutation } from 'react-apollo';
import pluralize from 'pluralize';
import { toast } from 'react-toastify';

import captureGraphQLErrors from 'modules/captureGraphQLErrors';
import CustomDialogue from 'Components/CustomDialogue';
import MessageBanner from 'Components/MessageBanner';
import Tags from 'Components/Tags';

const BULK_ADD_ALERT_TAGS = gql`
    mutation bulkAddAlertTags($resourceIds: [ID!]!, $tags: [String!]!) {
        bulkAddAlertTags(resourceIds: $resourceIds, tags: $tags)
    }
`;

function TagConfirmation({ setDialogue, checkedAlertIds, setCheckedAlertIds }) {
    const [tags, setTags] = useState([]);
    const [addBulkTags, { loading: isLoading, error, data }] = useMutation(BULK_ADD_ALERT_TAGS);
    const { hasErrors } = captureGraphQLErrors([error]);

    // if 'bulkAddAlertTags' is true, the modification was successful
    const isSuccessfullyAdded = data && data.bulkAddAlertTags;

    if (isSuccessfullyAdded) {
        toast('Tags were successfully added');
        closeAndClear();
    }

    function closeAndClear() {
        setDialogue(null);
        setCheckedAlertIds([]);
    }

    function tagViolations() {
        addBulkTags({
            variables: { resourceIds: checkedAlertIds, tags }
        });
    }

    const dialogueTitle = `Add Tags for ${checkedAlertIds.length} ${pluralize(
        'Violation',
        checkedAlertIds.length
    )}`;

    return (
        <CustomDialogue
            title={dialogueTitle}
            onConfirm={tagViolations}
            onCancel={closeAndClear}
            className="w-full md:w-1/2 lg:w-1/3"
            isLoading={isLoading}
            loadingText="Adding Tags"
        >
            {hasErrors && (
                <MessageBanner
                    type="error"
                    showCancel
                    message="There was an error adding tags. Please try again in a bit."
                />
            )}
            <div className="p-4">
                <Tags tags={tags} onChange={setTags} defaultOpen />
            </div>
        </CustomDialogue>
    );
}

TagConfirmation.propTypes = {
    setDialogue: PropTypes.func.isRequired,
    checkedAlertIds: PropTypes.arrayOf(PropTypes.string).isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired
};

export default React.memo(TagConfirmation);
