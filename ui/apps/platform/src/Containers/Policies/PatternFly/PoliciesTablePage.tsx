import React, { useState, useEffect, useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import {
    PageSection,
    Bullseye,
    Alert,
    Spinner,
    Dropdown,
    DropdownToggle,
    DropdownItem,
    Button,
    Tooltip,
    Flex,
    FlexItem,
    AlertGroup,
    AlertActionCloseButton,
    AlertVariant,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { selectors } from 'reducers';
import { actions as searchActions } from 'reducers/policies/search';
import { SearchEntry, SearchState } from 'reducers/pageSearch';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';
import {
    getPolicies,
    reassessPolicies,
    deletePolicies,
    exportPolicies,
    updatePoliciesDisabledState,
} from 'services/PoliciesService';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import useToasts from 'hooks/useToasts';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import { ListPolicy } from 'types/policy.proto';
// TODO: the policy import dialogue component will be migrated to PF in ROX-8354
import PolicyImportDialogue from '../Table/PolicyImportDialogue';
import PoliciesTable from './PoliciesTable';

const policiesPageState = createStructuredSelector<
    SearchState,
    { searchOptions: SearchEntry[]; searchModifiers: SearchEntry[] }
>({
    searchOptions: selectors.getPoliciesSearchOptions,
    searchModifiers: selectors.getPoliciesSearchModifiers,
});

function PoliciesTablePage(): React.ReactElement {
    const dispatch = useDispatch();

    const { searchOptions, searchModifiers } = useSelector(policiesPageState);
    const [isLoading, setIsLoading] = useState(false);
    const [policies, setPolicies] = useState<ListPolicy[]>([]);
    const [errorMessage, setErrorMessage] = useState('');
    const { toasts, addToast, removeToast } = useToasts();

    const [isImportModalOpen, setIsImportModalOpen] = useState(false);
    const [isDropdownOpen, setIsDropdownOpen] = useState(false);

    function onToggleDropdown(toggleDropdown) {
        setIsDropdownOpen(toggleDropdown);
    }

    function setSearchOptions(options) {
        dispatch(searchActions.setPoliciesSearchOptions(options));
    }

    function setSearchSuggestions(suggestions) {
        dispatch(searchActions.setPoliciesSearchSuggestions(suggestions));
    }

    function onClickImportPolicy() {
        setIsDropdownOpen(false);
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

    function handlePoliciesError(error) {
        setPolicies([]);
        const parsedMessage = checkForPermissionErrorMessage(error);
        setErrorMessage(parsedMessage);
    }

    const fetchPolicies = useCallback(() => {
        const query = searchOptionsToQuery(searchOptions);
        setIsLoading(true);
        getPolicies(query)
            .then((data) => setPolicies(data))
            .catch(handlePoliciesError)
            .finally(() => setIsLoading(false));
    }, [setPolicies, searchOptions]);

    function deletePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        deletePolicies(ids)
            .then(() => {
                fetchPolicies();
                addToast(`Successfully deleted ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not delete ${policyText}`, 'danger', response.data.message);
            });
    }

    function exportPoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        exportPolicies(ids)
            .then(() => {
                addToast(`Successfully exported ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not export the ${policyText}`, 'danger', response.data.message);
            });
    }

    function enablePoliciesHandler(ids: string[]) {
        const policyText = pluralize('policy', ids.length);
        updatePoliciesDisabledState(ids, false)
            .then(() => {
                fetchPolicies();
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
                fetchPolicies();
                addToast(`Successfully disabled ${policyText}`, 'success');
            })
            .catch(({ response }) => {
                addToast(`Could not disable the ${policyText}`, 'danger', response.data.message);
            });
    }

    useEffect(() => {
        if (
            searchOptions.length === 0 ||
            (searchOptions.length && !searchOptions[searchOptions.length - 1].type)
        ) {
            fetchPolicies();
        }
    }, [fetchPolicies, searchOptions]);

    const dropdownItems = [
        // TODO: add link to create form
        <DropdownItem key="link">Create policy</DropdownItem>,
        <DropdownItem key="action" component="button" onClick={onClickImportPolicy}>
            Import policy
        </DropdownItem>,
    ];

    const defaultOption = searchModifiers.find((x) => x.value === 'Policy:');
    return (
        <PageSection variant="light" isFilled id="policies-table">
            <ReduxSearchInput
                searchOptions={searchOptions}
                searchModifiers={searchModifiers}
                setSearchOptions={setSearchOptions}
                setSearchSuggestions={setSearchSuggestions}
                defaultOption={defaultOption}
                autoCompleteCategories={['POLICIES']}
            />
            {isLoading ?? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            )}
            {errorMessage ? (
                <Bullseye>
                    <Alert variant="danger" title={errorMessage} />
                </Bullseye>
            ) : (
                <>
                    <Flex className="pf-u-mt-sm">
                        <FlexItem>
                            <Dropdown
                                toggle={
                                    <DropdownToggle
                                        onToggle={onToggleDropdown}
                                        toggleIndicator={CaretDownIcon}
                                        isPrimary
                                        id="add-policy-dropdown-toggle"
                                    >
                                        Add Policy
                                    </DropdownToggle>
                                }
                                isOpen={isDropdownOpen}
                                dropdownItems={dropdownItems}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Tooltip content="Manually enrich external data">
                                <Button variant="secondary" onClick={onClickReassessPolicies}>
                                    Reassess all
                                </Button>
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                    <PoliciesTable
                        policies={policies}
                        deletePoliciesHandler={deletePoliciesHandler}
                        exportPoliciesHandler={exportPoliciesHandler}
                        enablePoliciesHandler={enablePoliciesHandler}
                        disablePoliciesHandler={disablePoliciesHandler}
                    />
                </>
            )}
            {isImportModalOpen && (
                <PolicyImportDialogue
                    closeAction={() => {
                        setIsImportModalOpen(false);
                    }}
                    importPolicySuccess={() => {
                        setIsImportModalOpen(false);
                    }}
                />
            )}
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
