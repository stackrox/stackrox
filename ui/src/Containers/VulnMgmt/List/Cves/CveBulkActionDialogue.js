import React, { useState } from 'react';
import PropTypes from 'prop-types';
import get from 'lodash/get';
import set from 'lodash/set';

import CustomDialogue from 'Components/CustomDialogue';

const CveBulkActionDialogue = ({ closeAction, bulkActionCveIds }) => {
    const [policy, setPolicy] = useState({ name: '' });

    // TODO: add useQuery to get the (hopefully cached) cve summaries to display in the dialog
    //       (this seems easier than refactoring the checkbox tables everywhere to maintain an array of selected entities)

    function handleChange(event) {
        if (get(policy, event.target.name) !== undefined) {
            const newPolicyFields = { ...policy };
            const newValue =
                event.target.type === 'checkbox' ? event.target.checked : event.target.value;
            set(newPolicyFields, event.target.name, newValue);
            setPolicy(newPolicyFields);
        }
    }

    function handleClose() {
        closeAction([]);
    }

    if (bulkActionCveIds.length === 0) return null;

    return (
        <CustomDialogue
            title="Add To Policy"
            text=""
            onConfirm={handleClose}
            confirmText="Shall I do it?"
            confirmDisabled={false}
            onCancel={handleClose}
        >
            {/* TODO: replace with working form, this is a temporary placeholder only */}
            <div className="p-2">
                <form>
                    <div className="mb-4">
                        <label htmlFor="name" className="block py-2 text-base-600 font-700">
                            Policy Name{' '}
                            <span
                                aria-label="Required"
                                data-test-id="required"
                                className="text-alert-500 ml-1"
                            >
                                *
                            </span>
                        </label>
                        <div className="flex">
                            <input
                                id="name"
                                name="name"
                                value={policy.name}
                                onChange={handleChange}
                                disabled={false}
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    Form goes here for {bulkActionCveIds.length}
                </form>
                <div className="p-2">
                    <h3>{`${
                        bulkActionCveIds.length
                    } CVEs listed below will be added to this policy:`}</h3>
                </div>
            </div>
        </CustomDialogue>
    );
};

CveBulkActionDialogue.propTypes = {
    closeAction: PropTypes.func.isRequired,
    bulkActionCveIds: PropTypes.arrayOf(PropTypes.string).isRequired
};

export default CveBulkActionDialogue;
