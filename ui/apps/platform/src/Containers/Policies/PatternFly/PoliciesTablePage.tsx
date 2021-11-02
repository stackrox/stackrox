import React, { useState, useEffect } from 'react';
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
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import { selectors } from 'reducers';
import { actions as searchActions } from 'reducers/policies/search';
import { SearchEntry, SearchState } from 'reducers/pageSearch';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';
import { getPolicies, reassessPolicies } from 'services/PoliciesService';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import { ListPolicy } from 'types/policy.proto';
// TODO: the policy import dialogue component will be migrated to PF in ROX-8354
import PolicyImportDialogue from '../Table/PolicyImportDialogue';

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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [policies, setPolicies] = useState<ListPolicy[]>([]);
    const [errorMessage, setErrorMessage] = useState('');

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
        // TODO: add toasts using PF in ROX-8454
        return reassessPolicies();
    }

    // on first load
    useEffect(() => {
        setIsLoading(true);
        getPolicies()
            .then((data) => setPolicies(data))
            .catch((error) => {
                setPolicies([]);
                const parsedMessage = checkForPermissionErrorMessage(error);
                setErrorMessage(parsedMessage);
            })
            .finally(() => setIsLoading(false));
    }, [setIsLoading, setPolicies, setErrorMessage]);

    // to watch on search options
    useEffect(() => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            const query = searchOptionsToQuery(searchOptions);
            getPolicies(query)
                .then((data) => setPolicies(data))
                .catch((error) => {
                    setPolicies([]);
                    const parsedMessage = checkForPermissionErrorMessage(error);
                    setErrorMessage(parsedMessage);
                });
        }
    }, [setErrorMessage, setPolicies, searchOptions]);

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
        </PageSection>
    );
}

export default PoliciesTablePage;
