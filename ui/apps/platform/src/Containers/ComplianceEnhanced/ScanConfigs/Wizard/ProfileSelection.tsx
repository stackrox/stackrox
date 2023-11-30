import React, { ReactElement, useCallback, useEffect } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import isEqual from 'lodash/isEqual';
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

    useEffect(() => {
        const profileIds = profiles.map((profile) => profile.id);
        const selectedProfileIds = profileIds.filter((_, index) => selected[index]);
        if (!isEqual(selectedProfileIds, formikValues.profiles)) {
            setFieldValue('profiles', selectedProfileIds);
        }
    }, [selected, formikValues.profiles, setFieldValue, profiles]);

    function renderTableContent() {
        return profiles?.map(({ id, name, description }, rowIndex) => (
            <Tr key={id}>
                <Td
                    key={id}
                    select={{
                        rowIndex,
                        onSelect,
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
                <Td>
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
                <Td>
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
                                    onSelect: onSelectAll,
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
