import { useCallback, useState } from 'react';
import type { FormEvent, ReactElement, RefObject } from 'react';
import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import {
    Alert,
    Bullseye,
    Content,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useTableSelection from 'hooks/useTableSelection';
import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

import { complianceProfileOperatorKindLabels } from '../../compliance.constants';
import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

export type ProfileSelectionProps = {
    alertRef: RefObject<HTMLDivElement>;
    profiles: ComplianceProfileSummary[];
    isFetchingProfiles: boolean;
};

function ProfileSelection({
    alertRef,
    profiles,
    isFetchingProfiles,
}: ProfileSelectionProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isTailoredProfilesEnabled = isFeatureFlagEnabled('ROX_TAILORED_PROFILES');

    const {
        setFieldValue,
        setTouched,
        values: formikValues,
        touched: formikTouched,
    }: FormikContextType<ScanConfigFormValues> = useFormikContext();

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

    const { selected, onSelect } = useTableSelection(profiles, profileIsPreSelected, 'name');

    const handleSelect = (
        event: FormEvent<HTMLInputElement>,
        isSelected: boolean,
        rowId: number
    ) => {
        onSelect(event, isSelected, rowId);

        const newSelectedProfileNames = profiles
            .filter((_, index) => {
                return index === rowId ? isSelected : selected[index];
            })
            .map((profile) => profile.name);

        setTouched({ ...formikTouched, profiles: true });
        setFieldValue('profiles', newSelectedProfileNames);
    };

    const isProfileExpanded = (name: string) => expandedProfileNames.includes(name);
    const totalColumns = isTailoredProfilesEnabled ? 7 : 6;

    function renderTableContent() {
        return profiles?.map(
            (
                { description, name, productType, ruleCount, title, profileVersion, operatorKind },
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
                                onToggle: () => setProfileExpanded(name, !isProfileExpanded(name)),
                            }}
                        />
                        <Td dataLabel="Profile">{name}</Td>
                        {isTailoredProfilesEnabled && (
                            <Td dataLabel="Kind">
                                {(operatorKind &&
                                    complianceProfileOperatorKindLabels[operatorKind]) ??
                                    '—'}
                            </Td>
                        )}
                        <Td dataLabel="Rule set">{ruleCount}</Td>
                        <Td dataLabel="Applicability">{productType}</Td>
                        <Td dataLabel="Version">{profileVersion || '-'}</Td>
                    </Tr>
                    <Tr isExpanded={isProfileExpanded(name)}>
                        <Td dataLabel="Profile details" colSpan={totalColumns}>
                            <ExpandableRowContent>
                                <Content component="p" className="pf-v6-u-font-weight-bold">
                                    {title}
                                </Content>
                                <Divider component="div" className="pf-v6-u-my-md" />
                                <Content component="p">{description}</Content>
                            </ExpandableRowContent>
                        </Td>
                    </Tr>
                </Tbody>
            )
        );
    }

    function renderLoadingContent() {
        return (
            <Tbody>
                <Tr>
                    <Td colSpan={totalColumns}>
                        <Bullseye>
                            <Spinner />
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
                    <Td colSpan={totalColumns}>
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
                <Table>
                    <Thead noWrap>
                        <Tr>
                            <Th>
                                <span className="pf-v6-screen-reader">Row selection</span>
                            </Th>
                            <Th>
                                <span className="pf-v6-screen-reader">Row expansion</span>
                            </Th>
                            <Th>Profile</Th>
                            {isTailoredProfilesEnabled && <Th>Kind</Th>}
                            <Th>Rule set</Th>
                            <Th>Applicability</Th>
                            <Th>Version</Th>
                        </Tr>
                    </Thead>
                    {renderTableBodyContent()}
                </Table>
            </Form>
        </>
    );
}

export default ProfileSelection;
