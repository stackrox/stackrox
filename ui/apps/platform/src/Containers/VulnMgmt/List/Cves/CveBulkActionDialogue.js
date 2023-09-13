import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import get from 'lodash/get';
import set from 'lodash/set';
import uniqBy from 'lodash/uniqBy';
import { Alert } from '@patternfly/react-core';

import InfoList from 'Components/InfoList';
import Loader from 'Components/Loader';
import { POLICY_ENTITY_ALL_FIELDS_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import entityTypes from 'constants/entityTypes';
import queryService from 'utils/queryService';
import { createPolicy, savePolicy } from 'services/PoliciesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { truncate } from 'utils/textUtils';
import { splitCvesByType } from 'utils/vulnerabilityUtils';

import CustomDialogue from '../../Components/CustomDialogue';

import CveToPolicyShortForm, { emptyPolicy } from './CveToPolicyShortForm';
import { parseCveNamesFromIds } from './ListCVEs.utils';

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

const CveBulkActionDialogue = ({ closeAction, bulkActionCveIds, cveType }) => {
    const [messageObj, setMessageObj] = useState(null);
    const dialogueRef = useRef(null);

    // the combined CVEs are used for the GraphQL query var
    const cvesStr = parseCveNamesFromIds(bulkActionCveIds).join(','); // only use the cve name, not the OS after the hash

    // prepare policy object
    const [policyIdentifer, setPolicyIdentifier] = useState('');

    // prepare policy object
    const populatedPolicy = { ...emptyPolicy, fields: { cve: cvesStr } };
    const [policy, setPolicy] = useState(populatedPolicy);
    const [policies, setPolicies] = useState([]);

    // use GraphQL to get the (hopefully cached) cve summaries to display in the dialog
    let CVE_QUERY = '';

    switch (cveType) {
        case entityTypes.NODE_CVE: {
            CVE_QUERY = gql`
                query getNodeCves($query: String) {
                    results: nodeVulnerabilities(query: $query) {
                        id
                        cve
                        summary
                    }
                }
            `;
            break;
        }
        case entityTypes.CLUSTER_CVE: {
            CVE_QUERY = gql`
                query getClusterCves($query: String) {
                    results: clusterVulnerabilities(query: $query) {
                        id
                        cve
                        summary
                    }
                }
            `;
            break;
        }
        case entityTypes.IMAGE_CVE:
        default: {
            CVE_QUERY = gql`
                query getImageCves($query: String) {
                    results: imageVulnerabilities(query: $query) {
                        id
                        cve
                        summary
                    }
                }
            `;
            break;
        }
    }

    const cvesObj = {
        cve: cvesStr,
    };
    const CVE_QUERY_OPTIONS = {
        variables: {
            query: queryService.objectToWhereClause(cvesObj),
        },
    };
    const { loading: cveLoading, data: cveData } = useQuery(CVE_QUERY, CVE_QUERY_OPTIONS);
    const cveItems =
        !cveLoading && cveData && cveData.results && cveData.results.length ? cveData.results : [];

    // split on vulnerabilityType is only for legacy RockDB support
    const {
        IMAGE_CVE: allowedCves,
        K8S_CVE: k8sCves,
        OPENSHIFT_CVE: openShiftCves,
    } = splitCvesByType(cveItems);
    const disallowedCves = cveType === entityTypes.CVE ? k8sCves.concat(openShiftCves) : [];

    // only the allowed CVEs are combined for use in the policy
    const allowedCvesValues =
        cveType === entityTypes.CVE
            ? allowedCves.map((cve) => ({ value: cve.cve }))
            : cveItems.map((cve) => ({ value: cve.cve }));
    const cvesToDisplay = cveType === entityTypes.CVE ? allowedCves : uniqBy(cveItems, 'cve');

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

    useEffect(() => {
        if (!policyLoading && policyData?.results?.length) {
            const existingPolicies = policyData.results
                .filter((policyToFilter) => !policyToFilter?.isDefault)
                .map((policyToMap, idx) => ({
                    ...policyToMap,
                    value: idx,
                    label: policyToMap.name,
                }));
            setPolicies(existingPolicies);
        }
    }, [policyLoading, policyData]);

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
        const newPolicy = cloneDeep(selectedPolicy);
        const newCveSection = {
            sectionName: 'CVEs',
            policyGroups: [{ fieldName: 'CVE', values: allowedCvesValues }],
        };

        if (policyExists) {
            // find policySection and policyGroup with CVEs
            const { policySectionIdx, policyGroupIdx } = findCVEField(newPolicy.policySections);

            // it matches an existing policy's ID, so must have been selected from existing list
            const newPolicySections = [...newPolicy.policySections];

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
                setMessageObj({ variant: 'success', title: 'Policy successfully saved' });

                // close the dialog after giving the user a little time to process the success message
                dialogueRef.current.scrollTo({ top: 0, behavior: 'smooth' });
                setTimeout(handleClose, 3000);
            })
            .catch((error) => {
                setMessageObj({
                    variant: 'danger',
                    title: 'Policy could not be saved. Please try again.',
                    text: getAxiosErrorMessage(error),
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
                <span className="min-w-32 font-700">{item.cve}</span>
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
            title="Add to policy"
            onConfirm={cvesToDisplay.length > 0 ? addToPolicy : null}
            confirmText="Save policy"
            confirmDisabled={Boolean(
                messageObj ||
                    policy.name.length < 6 ||
                    !policy.severity ||
                    !policy.lifecycleStages.length
            )}
            onCancel={closeWithoutSaving}
        >
            <div className="overflow-auto p-4" ref={dialogueRef}>
                {!cveLoading && cveType === entityTypes.CVE && cvesToDisplay.length === 0 ? (
                    <p>The selected CVEs cannot be added to a policy.</p>
                ) : (
                    <>
                        {messageObj && (
                            <Alert variant={messageObj.variant} isInline title={messageObj.title}>
                                {messageObj.text}
                            </Alert>
                        )}
                        <CveToPolicyShortForm
                            policy={policy}
                            handleChange={handleChange}
                            policies={policies}
                            selectedPolicy={policyIdentifer}
                            setSelectedPolicy={setSelectedPolicy}
                        />
                        <div className="pt-2">
                            <h3 className="mb-2">{`${cvesToDisplay.length} CVEs listed below will be added to this policy:`}</h3>
                            {cveLoading && <Loader />}
                            {!cveLoading && (
                                <InfoList
                                    items={cvesToDisplay}
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
    cveType: PropTypes.string.isRequired,
};

export default CveBulkActionDialogue;
