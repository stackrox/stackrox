import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import {
    PageSection,
    Bullseye,
    Alert,
    Spinner,
    AlertGroup,
    AlertActionCloseButton,
    AlertVariant,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';
import {
    getPolicies,
    reassessPolicies,
    deletePolicies,
    exportPolicies,
    updatePoliciesDisabledState,
} from 'services/PoliciesService';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import useToasts from 'hooks/useToasts';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { ListPolicy } from 'types/policy.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { SearchFilter } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getRequestQueryStringForSearchFilter } from '../policies.utils';
import ImportPolicyJSONModal from '../Modal/ImportPolicyJSONModal';
import PoliciesTable from './PoliciesTable';

type PoliciesTablePageProps = {
    hasWriteAccessForPolicy: boolean;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
};

function PoliciesTablePage({
    hasWriteAccessForPolicy,
    handleChangeSearchFilter,
    searchFilter,
}: PoliciesTablePageProps): React.ReactElement {
    const history = useHistory();

    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [policies, setPolicies] = useState<ListPolicy[]>([]);
    const [errorMessage, setErrorMessage] = useState('');
    const { toasts, addToast, removeToast } = useToasts();

    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    const [isImportModalOpen, setIsImportModalOpen] = useState(false);

    function onClickCreatePolicy() {
        history.push(`${policiesBasePath}/?action=create`);
    }

    function onClickImportPolicy() {
        setIsImportModalOpen(true);
    }

    function onClickReassessPolicies() {
        return reassessPolicies()
            .then(() => {
                addToast('Successfully reassessed policies', 'success');
            })
            .catch(({ response }) => {
                addToast('Could not reassess policies', 'danger', response.data.message);
            });
    }

    function fetchPolicies(query: string) {
        setIsLoading(true);
        getPolicies(query)
            .then((data) => {
                setPolicies(data);
                setErrorMessage('');
            })
            .catch((error) => {
                setPolicies([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoading(false));
    }

    const query = searchFilter ? getRequestQueryStringForSearchFilter(searchFilter) : '';

    function deletePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        deletePolicies(ids)
            .then(() => {
                fetchPolicies(query);
                addToast(`Successfully deleted ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not delete ${policyText}`, 'danger', response.data.message);
            });
    }

    function exportPoliciesHandler(ids: string[], onClearAll?: () => void) {
        const policyText = pluralize('policy', ids.length);
        exportPolicies(ids)
            .then(() => {
                addToast(`Successfully exported ${policyText}`, 'success');
                if (onClearAll) {
                    onClearAll();
                }
            })
            .catch(({ response }) => {
                addToast(`Could not export the ${policyText}`, 'danger', response.data.message);
            });
    }

    function enablePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        updatePoliciesDisabledState(ids, false)
            .then(() => {
                fetchPolicies(query);
                addToast(`Successfully enabled ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not enable the ${policyText}`, 'danger', response.data.message);
            });
    }

    function disablePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        updatePoliciesDisabledState(ids, true)
            .then(() => {
                fetchPolicies(query);
                addToast(`Successfully disabled ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not disable the ${policyText}`, 'danger', response.data.message);
            });
    }

    useEffect(() => {
        fetchNotifierIntegrations()
            .then((data) => {
                setNotifiers(data as NotifierIntegration[]);
            })
            .catch(() => {
                setNotifiers([]);
            });
    }, []);

    useEffect(() => {
        getSearchOptionsForCategory('POLICIES')
            .then((options) => {
                setSearchOptions(options);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    useEffect(() => {
        fetchPolicies(query);
    }, [query]);

    return (
        <PageSection variant="light" isFilled id="policies-table">
            {isLoading && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {errorMessage ? (
                <Bullseye>
                    <Alert variant="danger" title={errorMessage} />
                </Bullseye>
            ) : (
                <PoliciesTable
                    notifiers={notifiers}
                    policies={policies}
                    hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                    deletePoliciesHandler={deletePoliciesHandler}
                    exportPoliciesHandler={exportPoliciesHandler}
                    enablePoliciesHandler={enablePoliciesHandler}
                    disablePoliciesHandler={disablePoliciesHandler}
                    handleChangeSearchFilter={handleChangeSearchFilter}
                    onClickCreatePolicy={onClickCreatePolicy}
                    onClickImportPolicy={onClickImportPolicy}
                    onClickReassessPolicies={onClickReassessPolicies}
                    searchFilter={searchFilter}
                    searchOptions={searchOptions}
                />
            )}
            <ImportPolicyJSONModal
                isOpen={isImportModalOpen}
                cancelModal={() => {
                    setIsImportModalOpen(false);
                }}
                fetchPoliciesWithQuery={() => fetchPolicies(query)}
            />
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }) => (
                    <Alert
                        variant={AlertVariant[variant]}
                        title={title}
                        timeout={4000}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={`${variant as string} alert`}
                                onClose={() => removeToast(key)}
                            />
                        }
                        key={key}
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </PageSection>
    );
}

export default PoliciesTablePage;
