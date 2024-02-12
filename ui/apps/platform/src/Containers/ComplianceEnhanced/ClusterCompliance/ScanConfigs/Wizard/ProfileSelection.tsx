import React, { ReactElement, useCallback, useState } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import {
    Bullseye,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Spinner,
    Text,
    Title,
} from '@patternfly/react-core';
import {
    Caption,
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useTableSelection from 'hooks/useTableSelection';
import { ComplianceProfileSummary } from 'services/ComplianceEnhancedService';

import { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

export type ProfileSelectionProps = {
    profiles: ComplianceProfileSummary[];
    isFetchingProfiles: boolean;
};

function ProfileSelection({ profiles, isFetchingProfiles }: ProfileSelectionProps): ReactElement {
    const { setFieldValue, values: formikValues }: FormikContextType<ScanConfigFormValues> =
        useFormikContext();

    const [expandedProfileNames, setExpandedProfileNames] = useState<string[]>([]);
    const setProfileExpanded = (name: string, isExpanding = true) =>
        setExpandedProfileNames((prevExpanded) => {
            const otherExpandedProfileNames = prevExpanded.filter(
                (profileName) => profileName !== name
            );
            return isExpanding ? [...otherExpandedProfileNames, name] : otherExpandedProfileNames;
        });

    const profileIsPreSelected = useCallback(
        (row) => formikValues.profiles.includes(row.name),
        [formikValues.profiles]
    );

    const { allRowsSelected, selected, onSelect, onSelectAll } = useTableSelection(
        profiles,
        profileIsPreSelected,
        'name'
    );

    const handleSelect = (
        event: React.FormEvent<HTMLInputElement>,
        isSelected: boolean,
        rowId: number
    ) => {
        onSelect(event, isSelected, rowId);

        const newSelectedIds = profiles
            .filter((_, index) => {
                return index === rowId ? isSelected : selected[index];
            })
            .map((profile) => profile.name);

        setFieldValue('profiles', newSelectedIds);
    };

    const handleSelectAll = (event: React.FormEvent<HTMLInputElement>, isSelected: boolean) => {
        onSelectAll(event, isSelected);

        const newSelectedIds = isSelected ? profiles.map((profile) => profile.name) : [];

        setFieldValue('profiles', newSelectedIds);
    };

    const isProfileExpanded = (name: string) => expandedProfileNames.includes(name);

    function renderTableContent() {
        return profiles?.map(({ description, name, productType, ruleCount, title }, rowIndex) => (
            <Tbody>
                <Tr key={name}>
                    <Td
                        key={name}
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
                            onToggle: () => setProfileExpanded(name, !isProfileExpanded(name)),
                        }}
                    />
                    <Td>{name}</Td>
                    <Td>{ruleCount}</Td>
                    <Td>{productType}</Td>
                </Tr>
                <Tr isExpanded={isProfileExpanded(name)}>
                    <Td colSpan={1}></Td>
                    <Td colSpan={1}></Td>
                    <Td dataLabel="" colSpan={3}>
                        <ExpandableRowContent>
                            <Text className="pf-u-font-weight-bold">{title}</Text>
                            <Divider component="div" className="pf-u-my-md" />
                            <Text>{description}</Text>
                        </ExpandableRowContent>
                    </Td>
                </Tr>
            </Tbody>
        ));
    }

    function renderLoadingContent() {
        return (
            <Tbody>
                <Tr>
                    <Td colSpan={5}>
                        <Bullseye>
                            <Spinner isSVG />
                        </Bullseye>
                    </Td>
                </Tr>
            </Tbody>
        );
    }

    function renderEmptyContent() {
        return (
            <Tbody>
                <Tr>
                    <Td colSpan={5}>
                        <Bullseye>
                            <EmptyStateTemplate
                                title="No profiles"
                                headingLevel="h3"
                                icon={SearchIcon}
                            />
                        </Bullseye>
                    </Td>
                </Tr>
            </Tbody>
        );
    }

    function renderTableBodyContent() {
        if (isFetchingProfiles) {
            return renderLoadingContent();
        }
        if (profiles && profiles.length > 0) {
            return renderTableContent();
        }
        if (profiles && profiles.length === 0) {
            return renderEmptyContent();
        }
        return null;
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Profiles</Title>
                    </FlexItem>
                    <FlexItem>Select profiles to be included in the scan</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
                <TableComposable>
                    <Caption>At least one profile is required.</Caption>
                    <Thead noWrap>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: handleSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th />
                            <Th>Profile</Th>
                            <Th>Rule set</Th>
                            <Th>Applicability</Th>
                        </Tr>
                    </Thead>
                    {renderTableBodyContent()}
                </TableComposable>
            </Form>
        </>
    );
}

export default ProfileSelection;
