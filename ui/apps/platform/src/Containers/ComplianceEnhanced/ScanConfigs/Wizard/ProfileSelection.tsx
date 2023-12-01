import React, { ReactElement, useCallback } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import {
    Bullseye,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useTableSelection from 'hooks/useTableSelection';
import { ComplianceProfile } from 'services/ComplianceEnhancedService';

import { ScanConfigFormValues } from './useFormikScanConfig';

export type ProfileSelectionProps = {
    profiles: ComplianceProfile[];
    isFetchingProfiles: boolean;
};

function ProfileSelection({ profiles, isFetchingProfiles }: ProfileSelectionProps): ReactElement {
    const { setFieldValue, values: formikValues }: FormikContextType<ScanConfigFormValues> =
        useFormikContext();

    const profileIsPreSelected = useCallback(
        (row) => formikValues.profiles.includes(row.id),
        [formikValues.profiles]
    );

    const { allRowsSelected, selected, onSelect, onSelectAll } = useTableSelection(
        profiles,
        profileIsPreSelected
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
            .map((profile) => profile.id);

        setFieldValue('profiles', newSelectedIds);
    };

    const handleSelectAll = (event: React.FormEvent<HTMLInputElement>, isSelected: boolean) => {
        onSelectAll(event, isSelected);

        const newSelectedIds = isSelected ? profiles.map((profile) => profile.id) : [];

        setFieldValue('profiles', newSelectedIds);
    };

    function renderTableContent() {
        return profiles?.map(({ id, name, description }, rowIndex) => (
            <Tr key={id}>
                <Td
                    key={id}
                    select={{
                        rowIndex,
                        onSelect: (event, isSelected) => handleSelect(event, isSelected, rowIndex),
                        isSelected: selected[rowIndex],
                    }}
                />
                <Td>{name}</Td>
                <Td>{description}</Td>
            </Tr>
        ));
    }

    function renderLoadingContent() {
        return (
            <Tr>
                <Td colSpan={3}>
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                </Td>
            </Tr>
        );
    }

    function renderEmptyContent() {
        return (
            <Tr>
                <Td colSpan={3}>
                    <Bullseye>
                        <EmptyStateTemplate
                            title="No profiles found"
                            headingLevel="h2"
                            icon={SearchIcon}
                        />
                    </Bullseye>
                </Td>
            </Tr>
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
                        <Title headingLevel="h1">Profiles</Title>
                    </FlexItem>
                    <FlexItem>Select profiles to be included in the scan</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
                <TableComposable variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: handleSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th>Name</Th>
                            <Th>Description</Th>
                        </Tr>
                    </Thead>
                    <Tbody>{renderTableBodyContent()}</Tbody>
                </TableComposable>
            </Form>
        </>
    );
}

export default ProfileSelection;
