import React, { useState, useEffect } from 'react';
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

import SearchFilterInput from 'Components/SearchFilterInput';
import {
    getPolicies,
    reassessPolicies,
    deletePolicies,
    exportPolicies,
    updatePoliciesDisabledState,
} from 'services/PoliciesService';
import useToasts from 'hooks/useToasts';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { ListPolicy } from 'types/policy.proto';
import { SearchFilter } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getRequestQueryStringForSearchFilter } from './policies.utils';
// TODO: the policy import dialogue component will be migrated to PF in ROX-8354
import PolicyImportDialogue from '../Table/PolicyImportDialogue';
import PoliciesTable from './PoliciesTable';

const searchCategory = 'POLICIES';

type PoliciesTablePageProps = {
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
};

function PoliciesTablePage({
    handleChangeSearchFilter,
    searchFilter,
}: PoliciesTablePageProps): React.ReactElement {
    const [isLoading, setIsLoading] = useState(false);
    const [policies, setPolicies] = useState<ListPolicy[]>([]);
    const [errorMessage, setErrorMessage] = useState('');
    const { toasts, addToast, removeToast } = useToasts();

    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    const [isImportModalOpen, setIsImportModalOpen] = useState(false);
    const [isDropdownOpen, setIsDropdownOpen] = useState(false);

    function onToggleDropdown(toggleDropdown) {
        setIsDropdownOpen(toggleDropdown);
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
        getSearchOptionsForCategory(searchCategory)
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

    const dropdownItems = [
        // TODO: add link to create form
        <DropdownItem key="link">Create policy</DropdownItem>,
        <DropdownItem key="action" component="button" onClick={onClickImportPolicy}>
            Import policy
        </DropdownItem>,
    ];

    return (
        <PageSection variant="light" isFilled id="policies-table">
            <SearchFilterInput
                className="w-full"
                handleChangeSearchFilter={handleChangeSearchFilter}
                placeholder="Filter policies"
                searchCategory={searchCategory}
                searchFilter={searchFilter ?? {}}
                searchOptions={searchOptions}
            />
            {isLoading && (
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
