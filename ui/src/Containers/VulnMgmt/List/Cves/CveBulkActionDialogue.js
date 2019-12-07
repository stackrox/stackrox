import React, { useState } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';
import get from 'lodash/get';
import set from 'lodash/set';

import CustomDialogue from 'Components/CustomDialogue';
import InfoList from 'Components/InfoList';
import Loader from 'Components/Loader';
import Message from 'Components/Message';
import queryService from 'modules/queryService';
import { createPolicy } from 'services/PoliciesService';
import { truncate } from 'utils/textUtils';

import CveToPolicyShortForm from './CveToPolicyShortForm';

const emptyPolicy = {
    name: '',
    severity: '',
    lifecycleStages: [],
    description: '',
    disabled: false,
    categories: ['Vulnerability Management'],
    fields: {
        cve: ''
    },
    whitelists: []
};

const CveBulkActionDialogue = ({ closeAction, bulkActionCveIds }) => {
    const [showMessage, setShowMessage] = useState(false);

    // the combined CVEs are used for both the policy object and the GraphQL query var
    const cvesStr = bulkActionCveIds.join(',');

    // prepare policy object
    const populatedPolicy = { ...emptyPolicy, fields: { cve: cvesStr } };
    const [policy, setPolicy] = useState(populatedPolicy);

    // use GraphQL to get the (hopefully cached) cve summaries to display in the dialog
    const CVES_QUERY = gql`
        query getCves($query: String) {
            results: vulnerabilities(query: $query) {
                id: cve
                cve
                summary
            }
        }
    `;
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

    function savePolicy() {
        // TODO: make the form submission more robust, and handle adding to an existing policy
        //   this current save function is only for smoke-testing the form
        createPolicy(policy)
            .then(() => {
                setShowMessage(true);
            })
            .catch(error => {
                // eslint-disable-next-line no-console
                console.log(error);
            });
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
            onConfirm={savePolicy}
            confirmText="Save Policy"
            confirmDisabled={showMessage}
            onCancel={handleClose}
        >
            {/* TODO: replace with working form, this is a temporary placeholder only */}
            <div className="p-4">
                {showMessage && <Message type="info" message="Policy successfully saved" />}
                <CveToPolicyShortForm policy={policy} handleChange={handleChange} />
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
