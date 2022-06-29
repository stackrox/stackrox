import React, { useState, useRef } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import get from 'lodash/get';
import set from 'lodash/set';
import { Message } from '@stackrox/ui-components';

import CustomDialogue from 'Components/CustomDialogue';
import InfoList from 'Components/InfoList';
import Loader from 'Components/Loader';
import { POLICY_ENTITY_ALL_FIELDS_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import queryService from 'utils/queryService';
import { createPolicy, savePolicy } from 'services/PoliciesService';
import { truncate } from 'utils/textUtils';
import { splitCvesByType } from 'utils/vulnerabilityUtils';

import CveToPolicyShortForm, { emptyPolicy } from './CveToPolicyShortForm';

const findCVEField = (policySections) => {
    let policySectionIdx = null;
    let policyGroupIdx = null;
    policySections.forEach((policySection, sectionIdx) => {
        policySection.policyGroups.forEach(({ fieldName }, groupIdx) => {
            if (fieldName === 'CVE') {
                policySectionIdx = sectionIdx;
                policyGroupIdx = groupIdx;
            }
        });
    });
    return { policySectionIdx, policyGroupIdx };
};

const CveBulkActionDialogue = ({ closeAction, bulkActionCveIds }) => {
    const [messageObj, setMessageObj] = useState(null);
    const dialogueRef = useRef(null);

    // the combined CVEs are used for the GraphQL query var
    const cvesStr = bulkActionCveIds.join(',');

    // prepare policy object
    const [policyIdentifer, setPolicyIdentifier] = useState('');

    // prepare policy object
    const populatedPolicy = { ...emptyPolicy, fields: { cve: cvesStr } };
    const [policy, setPolicy] = useState(populatedPolicy);
    const [policies, setPolicies] = useState([]);

    // use GraphQL to get the (hopefully cached) cve summaries to display in the dialog
    const CVES_QUERY = gql`
        query getCves($query: String) {
            results: vulnerabilities(query: $query) {
                id
                cve
                summary
                vulnerabilityTypes
            }
        }
    `;
    const cvesObj = {
        cve: cvesStr,
    };
    const cveQueryOptions = {
        variables: {
            query: queryService.objectToWhereClause(cvesObj),
        },
    };
    const { loading: cveLoading, data: cveData } = useQuery(CVES_QUERY, cveQueryOptions);
    const cveItems =
        !cveLoading && cveData && cveData.results && cveData.results.length ? cveData.results : [];

    const {
        IMAGE_CVE: allowedCves,
        K8S_CVE: k8sCves,
        OPENSHIFT_CVE: openShiftCves,
    } = splitCvesByType(cveItems);
    const disallowedCves = k8sCves.concat(openShiftCves);

    // only the allowed CVEs are combined for use in the policy
    const allowedCvesValues = allowedCves.map((cve) => ({ value: cve.cve }));

    // use GraphQL to get existing vulnerability-related policies
    const POLICIES_QUERY = gql`
        query getPolicies($policyQuery: String, $scopeQuery: String) {
            results: policies(query: $policyQuery) {
                ...policyFields
                unusedVarSink(query: $scopeQuery)
            }
        }
        ${POLICY_ENTITY_ALL_FIELDS_FRAGMENT}
    `;
    const policyQueryOptions = {
        variables: {
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management',
            }),
            scopeQuery: '',
        },
    };
    const { loading: policyLoading, data: policyData } = useQuery(
        POLICIES_QUERY,
        policyQueryOptions
    );

    if (
        !policyLoading &&
        policyData &&
        policyData.results &&
        policyData.results.length &&
        policies.length === 0
    ) {
        const existingPolicies = policyData.results.map((pol, idx) => ({
            ...pol,
            value: idx,
            label: pol.name,
        }));
        setPolicies(existingPolicies);
    }

    function handleChange(event) {
        if (get(policy, event.target.name) !== undefined) {
            const newPolicyFields = { ...policy };
            const newValue =
                event.target.type === 'checkbox' ? event.target.checked : event.target.value;
            set(newPolicyFields, event.target.name, newValue);
            setPolicy(newPolicyFields);
        }
    }

    function setSelectedPolicy(selectedPolicy) {
        // checking if the policy already exists or has already been added to the policy list
        const policyExists = policies && policies.find((pol) => pol.value === selectedPolicy.value);
        const newPolicy = { ...selectedPolicy };
        const newCveSection = {
            sectionName: 'CVEs',
            policyGroups: [{ fieldName: 'CVE', values: allowedCvesValues }],
        };

        if (policyExists) {
            // find policySection and policyGroup with CVEs
            const { policySectionIdx, policyGroupIdx } = findCVEField(
                selectedPolicy.policySections
            );
            // it matches an existing policy's ID, so must have been selected from existing list
            const newPolicySections = [...selectedPolicy.policySections];
            if (policySectionIdx !== null) {
                newPolicySections[policySectionIdx].policyGroups[policyGroupIdx].values.push(
                    ...allowedCvesValues
                );
            } else {
                newPolicySections.push(newCveSection);
            }
            newPolicy.policySections = newPolicySections;
        } else {
            // 1. not in existing list, so must be a typed name instead of an ID
            // 2. also use this opportunity to only add allowed CVEs to new policy
            newPolicy.policySections = [newCveSection];
        }

        // update the form
        setPolicy(newPolicy);
        setPolicyIdentifier(selectedPolicy.value);

        if (!policyExists) {
            setPolicies([...policies, newPolicy]);
        }
    }

    function handleClose(idsToStaySelected) {
        closeAction(idsToStaySelected);
    }

    function closeWithoutSaving() {
        handleClose(bulkActionCveIds);
    }

    function addToPolicy() {
        // TODO: make the form submission more robust
        //   this current save function is only for smoke-testing the form
        const addToFunc = policy.id ? savePolicy : createPolicy;

        addToFunc(policy)
            .then(() => {
                setMessageObj({ type: 'success', message: 'Policy successfully saved' });

                // close the dialog after giving the user a little time to process the success message
                dialogueRef.current.scrollTo({ top: 0, behavior: 'smooth' });
                setTimeout(handleClose, 3000);
            })
            .catch((error) => {
                setMessageObj({
                    type: 'error',
                    message: `Policy could not be saved. Please try again. (${error})`,
                });

                // hide the error message after giving the user time to read it
                dialogueRef.current.scrollTo({ top: 0, behavior: 'smooth' });
                setTimeout(() => {
                    setMessageObj(null);
                }, 7000);
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

    // render section
    if (bulkActionCveIds.length === 0) {
        return null;
    }

    return (
        <CustomDialogue
            className="max-w-3/4 md:max-w-2/3 lg:max-w-1/2"
            title="Add To Policy"
            text=""
            onConfirm={allowedCves.length > 0 ? addToPolicy : null}
            confirmText="Save Policy"
            confirmDisabled={
                messageObj ||
                policy.name.length < 6 ||
                !policy.severity ||
                !policy.lifecycleStages.length
            }
            onCancel={closeWithoutSaving}
        >
            <div className="overflow-auto p-4" ref={dialogueRef}>
                {!cveLoading && allowedCves.length === 0 ? (
                    <p>The selected CVEs cannot be added to a policy.</p>
                ) : (
                    <>
                        {messageObj && (
                            <Message type={messageObj.type}>{messageObj.message}</Message>
                        )}
                        <CveToPolicyShortForm
                            policy={policy}
                            handleChange={handleChange}
                            policies={policies}
                            selectedPolicy={policyIdentifer}
                            setSelectedPolicy={setSelectedPolicy}
                        />
                        <div className="pt-2">
                            <h3 className="mb-2">{`${allowedCves.length} CVEs listed below will be added to this policy:`}</h3>
                            {cveLoading && <Loader />}
                            {!cveLoading && (
                                <InfoList
                                    items={allowedCves}
                                    renderItem={renderCve}
                                    extraClassNames="h-48"
                                />
                            )}
                        </div>
                        {!cveLoading && disallowedCves.length > 0 && (
                            <div className="pt-2">
                                <h3 className="mb-2">
                                    {`The following ${disallowedCves.length} CVEs cannot be added to a policy.`}
                                </h3>
                                <InfoList
                                    items={disallowedCves}
                                    renderItem={renderCve}
                                    extraClassNames="h-24"
                                />
                            </div>
                        )}
                    </>
                )}
            </div>
        </CustomDialogue>
    );
};

CveBulkActionDialogue.propTypes = {
    closeAction: PropTypes.func.isRequired,
    bulkActionCveIds: PropTypes.arrayOf(PropTypes.string).isRequired,
};

export default CveBulkActionDialogue;
