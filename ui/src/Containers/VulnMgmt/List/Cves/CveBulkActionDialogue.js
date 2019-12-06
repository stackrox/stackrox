import React, { useState } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';
import get from 'lodash/get';
import set from 'lodash/set';

import CustomDialogue from 'Components/CustomDialogue';
import InfoList from 'Components/InfoList';
import Loader from 'Components/Loader';
import queryService from 'modules/queryService';
import { truncate } from 'utils/textUtils';

const CveBulkActionDialogue = ({ closeAction, bulkActionCveIds }) => {
    const [policy, setPolicy] = useState({ name: '' });

    // TODO: add useQuery to get the (hopefully cached) cve summaries to display in the dialog
    //       (this seems easier than refactoring the checkbox tables everywhere to maintain an array of selected entities)
    const CVES_QUERY = gql`
        query getCves($query: String) {
            results: vulnerabilities(query: $query) {
                id: cve
                cve
                summary
            }
        }
    `;

    const cvesStr = bulkActionCveIds.join(',');
    const cvesObj = {
        cve: cvesStr
    };
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(cvesObj)
        }
    };
    const { loading, data: cveData } = useQuery(CVES_QUERY, queryOptions);

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
    function renderCve(item) {
        const truncatedSummary = truncate(item.summary, 120);
        return (
            <li key={item.id} className="flex items-center bg-tertiary-200 mb-2 p-2">
                <span className="min-w-32">{item.cve}</span>
                <span>{truncatedSummary}</span>
            </li>
        );
    }

    const cveItems =
        !loading && cveData && cveData.results && cveData.results.length ? cveData.results : [];

    if (bulkActionCveIds.length === 0) return null;

    return (
        <CustomDialogue
            className="max-w-3/4 md:max-w-2/3 lg:max-w-1/2"
            title="Add To Policy"
            text=""
            onConfirm={handleClose}
            confirmText="Shall I do it?"
            confirmDisabled={false}
            onCancel={handleClose}
        >
            {/* TODO: replace with working form, this is a temporary placeholder only */}
            <div className="p-4">
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
                </form>
                <div className="pt-2">
                    <h3 className="mb-2">{`${
                        bulkActionCveIds.length
                    } CVEs listed below will be added to this policy:`}</h3>
                    {loading && <Loader />}
                    {!loading && (
                        <InfoList items={cveItems} renderItem={renderCve} extraClassNames="h-48" />
                    )}
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
