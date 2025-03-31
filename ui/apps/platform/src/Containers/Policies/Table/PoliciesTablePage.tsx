import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    PageSection,
    Bullseye,
    Alert,
    Spinner,
    AlertGroup,
    AlertActionCloseButton,
    Divider,
    Button,
} from '@patternfly/react-core';
import pluralize from 'pluralize';
import orderBy from 'lodash/orderBy';

import { policiesBasePath } from 'routePaths';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import {
    getPolicies,
    reassessPolicies,
    deletePolicies,
    exportPolicies,
    updatePoliciesDisabledState,
} from 'services/PoliciesService';
import { savePoliciesAsCustomResource } from 'services/PolicyCustomResourceService';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import useURLSort from 'hooks/useURLSort';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { ListPolicy } from 'types/policy.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { ApiSortOption, SearchFilter } from 'types/search';
import { SortOption } from 'types/table';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import PolicyManagementHeader from 'Containers/PolicyManagement/PolicyManagementHeader';
import ImportPolicyJSONModal from '../Modal/ImportPolicyJSONModal';
import PoliciesTable from './PoliciesTable';
import { columns } from './PoliciesTable.utils';

type PoliciesTablePageProps = {
    hasWriteAccessForPolicy: boolean;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
};

export const sortFields = ['Policy', 'Status', 'Origin', 'Notifiers', 'Severity', 'Lifecycle'];
export const defaultSortOption: SortOption = {
    field: 'Policy',
    direction: 'asc',
};

function PoliciesTablePage({
    hasWriteAccessForPolicy,
    handleChangeSearchFilter,
    searchFilter,
}: PoliciesTablePageProps): React.ReactElement {
    const navigate = useNavigate();
    const { getSortParams, sortOption } = useURLSort({ defaultSortOption, sortFields });

    const [notifiers, setNotifiers] = useState<NotifierIntegration[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [policies, setPolicies] = useState<ListPolicy[]>([]);
    const [errorMessage, setErrorMessage] = useState('');
    const { toasts, addToast, removeToast } = useToasts();

    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    const [isImportModalOpen, setIsImportModalOpen] = useState(false);

    const query = searchFilter ? getRequestQueryStringForSearchFilter(searchFilter) : '';

    function onClickCreatePolicy() {
        navigate(`${policiesBasePath}/?action=create`);
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

    function fetchPolicies(query: string, fetchSortOption: ApiSortOption) {
        // The policy table does not currently support multi sort, but it must handle the case where the sortOption is an array
        // due to the hook's API. Although this should not occur, we will handle it here by using the first option.
        const sortOption = Array.isArray(fetchSortOption) ? fetchSortOption[0] : fetchSortOption;
        setIsLoading(true);
        getPolicies(query)
            .then((data) => {
                const { field, reversed } = sortOption;
                const activeSortIndex = columns.findIndex((col) => col.Header === field) || 0;
                const activeSortDirection = reversed ? 'desc' : 'asc';
                const { sortMethod, accessor } = columns[activeSortIndex];

                let sortedPolicies = [...data];
                if (sortMethod) {
                    sortedPolicies.sort(sortMethod);
                    if (activeSortDirection === 'desc') {
                        sortedPolicies.reverse();
                    }
                } else {
                    sortedPolicies = orderBy(sortedPolicies, [accessor], [activeSortDirection]);
                }

                setPolicies(sortedPolicies);
                setErrorMessage('');
            })
            .catch((error) => {
                setPolicies([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoading(false));
    }

    function deletePoliciesHandler(ids: string[]): Promise<void> {
        const policyText = pluralize('policy', ids.length);
        return deletePolicies(ids)
            .then(() => {
                fetchPolicies(query, sortOption);
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
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                addToast(`Could not export the ${policyText}`, 'danger', message);
            });
    }

    function saveAsCustomResourceHandler(ids: string[], onClearAll?: () => void): Promise<void> {
        return savePoliciesAsCustomResource(ids)
            .then(() => {
                addToast(`Successfully saved selected policies as Custom Resources`, 'success');
                if (onClearAll) {
                    onClearAll();
                }
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                addToast(
                    `Could not save the selected policies as Custom Resources`,
                    'danger',
                    message
                );
            });
    }

    function enablePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        updatePoliciesDisabledState(ids, false)
            .then(() => {
                fetchPolicies(query, sortOption);
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
                fetchPolicies(query, sortOption);
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
        const { request, cancel } = getSearchOptionsForCategory('POLICIES');
        request
            .then((options) => {
                setSearchOptions(options);
            })
            .catch(() => {
                // TODO
            });

        return cancel;
    }, []);

    useEffect(() => {
        fetchPolicies(query, sortOption);
    }, [query, sortOption]);

    let pageContent = (
        <PageSection variant="light" isFilled id="policies-table-loading">
            <Bullseye>
                <Spinner />
            </Bullseye>
        </PageSection>
    );

    if (errorMessage) {
        pageContent = (
            <PageSection variant="light" isFilled id="policies-table-error">
                <Bullseye>
                    <Alert variant="danger" title={errorMessage} component="p" />
                </Bullseye>
            </PageSection>
        );
    }

    if (!isLoading && !errorMessage) {
        pageContent = (
            <PoliciesTable
                notifiers={notifiers}
                policies={policies}
                fetchPoliciesHandler={() => fetchPolicies(query, sortOption)}
                addToast={addToast}
                hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                deletePoliciesHandler={deletePoliciesHandler}
                exportPoliciesHandler={exportPoliciesHandler}
                saveAsCustomResourceHandler={saveAsCustomResourceHandler}
                enablePoliciesHandler={enablePoliciesHandler}
                disablePoliciesHandler={disablePoliciesHandler}
                handleChangeSearchFilter={handleChangeSearchFilter}
                onClickReassessPolicies={onClickReassessPolicies}
                getSortParams={getSortParams}
                searchFilter={searchFilter}
                searchOptions={searchOptions}
            />
        );
    }

    return (
        <>
            <PolicyManagementHeader currentTabTitle="Policies" />
            <Divider component="div" />
            <TabNavSubHeader
                description="Configure security policies for your resources."
                actions={
                    hasWriteAccessForPolicy ? (
                        <>
                            <Button variant="primary" onClick={onClickCreatePolicy}>
                                Create policy
                            </Button>
                            <Button variant="secondary" onClick={onClickImportPolicy}>
                                Import policy
                            </Button>
                        </>
                    ) : (
                        <></>
                    )
                }
            />
            <Divider component="div" />
            {pageContent}
            <ImportPolicyJSONModal
                isOpen={isImportModalOpen}
                cancelModal={() => {
                    setIsImportModalOpen(false);
                }}
                fetchPoliciesWithQuery={() => fetchPolicies(query, sortOption)}
            />
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        variant={variant}
                        title={title}
                        component="p"
                        timeout={4000}
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={`${variant} alert`}
                                onClose={() => removeToast(key)}
                            />
                        }
                        key={key}
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default PoliciesTablePage;
