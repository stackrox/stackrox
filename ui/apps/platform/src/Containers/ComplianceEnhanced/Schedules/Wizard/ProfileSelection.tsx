import { useCallback, useState } from 'react';
import type { FormEvent, ReactElement, RefObject } from 'react';
import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import {
    Alert,
    Content,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import TBodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import useTableSelection from 'hooks/useTableSelection';
import type { ComplianceProfileSummary } from 'services/ComplianceCommon';
import { listProfileSummaries } from 'services/ComplianceProfileService';
import type { SearchFilter } from 'types/search';
import { getTableUIState } from 'utils/getTableUIState';

import { complianceProfileOperatorKindLabels } from '../../compliance.constants';
import { profileSearchFilterConfig } from '../../searchFilterConfig';
import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

const searchFilterConfig = [profileSearchFilterConfig];

export type ProfileSelectionProps = {
    alertRef: RefObject<HTMLDivElement>;
    clusterIds: string[];
};

function ProfileSelection({ alertRef, clusterIds }: ProfileSelectionProps): ReactElement {
    const {
        setFieldValue,
        values: formikValues,
        touched: formikTouched,
    }: FormikContextType<ScanConfigFormValues> = useFormikContext();

    const [searchFilter, setSearchFilter] = useState<SearchFilter>({});
    const [expandedProfileNames, setExpandedProfileNames] = useState<string[]>([]);
    const setProfileExpanded = (name: string, isExpanding = true) =>
        setExpandedProfileNames((prevExpanded) => {
            const otherExpandedProfileNames = prevExpanded.filter(
                (profileName) => profileName !== name
            );
            return isExpanding ? [...otherExpandedProfileNames, name] : otherExpandedProfileNames;
        });

    const onSearch: OnSearchCallback = (payload) => {
        setSearchFilter((prevSearchFilter) => updateSearchFilter(prevSearchFilter, payload));
    };

    const listProfilesQuery = useCallback(() => {
        if (clusterIds.length > 0) {
            return listProfileSummaries(clusterIds, searchFilter);
        }
        return Promise.resolve([]);
    }, [clusterIds, searchFilter]);
    const {
        data: profiles,
        isLoading: isFetchingProfiles,
        error: profilesFetchError,
    } = useRestQuery(listProfilesQuery);

    const profileList = profiles ?? [];

    const profileIsPreSelected = useCallback(
        (row) => formikValues.profiles.includes(row.name),
        [formikValues.profiles]
    );

    const { selected, onSelect } = useTableSelection(profileList, profileIsPreSelected, 'name');

    const handleSelect = (
        event: FormEvent<HTMLInputElement>,
        isSelected: boolean,
        rowId: number
    ) => {
        onSelect(event, isSelected, rowId);

        const visibleSelectedNames = profileList
            .filter((_, index) => {
                return index === rowId ? isSelected : selected[index];
            })
            .map((profile) => profile.name);

        const hiddenSelectedNames = formikValues.profiles.filter(
            (name) => !profileList.some((p) => p.name === name)
        );

        setFieldValue('profiles', [...hiddenSelectedNames, ...visibleSelectedNames]);
    };

    const isProfileExpanded = (name: string) => expandedProfileNames.includes(name);
    const totalColumns = 7;

    const tableState = getTableUIState({
        isLoading: isFetchingProfiles,
        data: profiles,
        error: profilesFetchError,
        searchFilter,
    });

    return (
        <>
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v6-u-py-lg pf-v6-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Profiles</Title>
                    </FlexItem>
                    <FlexItem>Select profiles to be included in the scan</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v6-u-py-lg pf-v6-u-px-lg" ref={alertRef}>
                {formikTouched.profiles && formikValues.profiles.length === 0 && (
                    <Alert
                        title="At least one profile is required to proceed"
                        component="p"
                        variant="danger"
                        isInline
                    />
                )}
                <Toolbar>
                    <ToolbarContent>
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={onSearch}
                        />
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <CompoundSearchFilterLabels
                                attributesSeparateFromConfig={[]}
                                config={searchFilterConfig}
                                onFilterChange={setSearchFilter}
                                searchFilter={searchFilter}
                            />
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <Table>
                    <Thead noWrap>
                        <Tr>
                            <Th screenReaderText="Row selection" />
                            <Th screenReaderText="Row expansion" />
                            <Th>Profile</Th>
                            <Th>Type</Th>
                            <Th>Rule set</Th>
                            <Th>Applicability</Th>
                            <Th>Version</Th>
                        </Tr>
                    </Thead>
                    <TBodyUnified<ComplianceProfileSummary>
                        tableState={tableState}
                        colSpan={totalColumns}
                        filteredEmptyProps={{ onClearFilters: () => setSearchFilter({}) }}
                        renderer={({ data }) =>
                            data.map(
                                (
                                    {
                                        description,
                                        name,
                                        productType,
                                        ruleCount,
                                        title,
                                        profileVersion,
                                        operatorKind,
                                    },
                                    rowIndex
                                ) => (
                                    <Tbody isExpanded={isProfileExpanded(name)} key={name}>
                                        <Tr>
                                            <Td
                                                select={{
                                                    rowIndex,
                                                    onSelect: (event, isSelected) =>
                                                        handleSelect(event, isSelected, rowIndex),
                                                    isSelected: selected[rowIndex],
                                                }}
                                            />
                                            <Td
                                                expand={{
                                                    rowIndex,
                                                    isExpanded: isProfileExpanded(name),
                                                    onToggle: () =>
                                                        setProfileExpanded(
                                                            name,
                                                            !isProfileExpanded(name)
                                                        ),
                                                }}
                                            />
                                            <Td dataLabel="Profile">{name}</Td>
                                            <Td dataLabel="Type">
                                                {(operatorKind &&
                                                    complianceProfileOperatorKindLabels[
                                                        operatorKind
                                                    ]) ??
                                                    '—'}
                                            </Td>
                                            <Td dataLabel="Rule set">{ruleCount}</Td>
                                            <Td dataLabel="Applicability">{productType}</Td>
                                            <Td dataLabel="Version">{profileVersion || '-'}</Td>
                                        </Tr>
                                        <Tr isExpanded={isProfileExpanded(name)}>
                                            <Td colSpan={totalColumns}>
                                                <ExpandableRowContent>
                                                    <Content component="p">{title}</Content>
                                                    <Content component="p">{description}</Content>
                                                </ExpandableRowContent>
                                            </Td>
                                        </Tr>
                                    </Tbody>
                                )
                            )
                        }
                    />
                </Table>
            </Form>
        </>
    );
}

export default ProfileSelection;
