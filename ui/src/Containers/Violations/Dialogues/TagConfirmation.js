import React, { useState } from 'react';
import PropTypes from 'prop-types';

import CustomDialogue from 'Components/CustomDialogue';
import Tags from 'Components/Tags';

function TagConfirmation({ setDialogue, setCheckedAlertIds }) {
    const [tags, setTags] = useState();

    function closeAndClear() {
        setDialogue(null);
        setCheckedAlertIds([]);
        setTags([]);
    }

    function tagViolations() {
        // do something
        closeAndClear();
    }

    function close() {
        setDialogue(null);
    }

    return (
        <CustomDialogue
            onConfirm={tagViolations}
            onCancel={close}
            className="w-full md:w-1/2 lg:w-1/3"
        >
            <div className="p-4">
                <Tags
                    type="New Violation"
                    tags={tags}
                    onChange={setTags}
                    defaultOpen
                    isCollapsible={false}
                />
            </div>
        </CustomDialogue>
    );
}

TagConfirmation.propTypes = {
    setDialogue: PropTypes.func.isRequired,
    setCheckedAlertIds: PropTypes.func.isRequired
};

export default TagConfirmation;
