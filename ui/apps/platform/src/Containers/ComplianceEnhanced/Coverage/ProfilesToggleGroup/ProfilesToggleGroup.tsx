import { useEffect, useMemo } from 'react';
import { PageSection, Tab, Tabs, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

import { getProfileTab, getStandardTabs } from './ProfilesToggleGroup.utils';

const tabContentId = 'profiles-toggle-group';

type ProfilesToggleGroupProps = {
    profileName: string;
    profiles: ComplianceProfileSummary[];
    handleToggleChange: (selectedProfile: string) => void;
};

function ProfilesToggleGroup({
    profileName,
    profiles,
    handleToggleChange,
}: ProfilesToggleGroupProps) {
    const standardTabs = useMemo(() => getStandardTabs(profiles), [profiles]);
    const selectedProfile = useMemo(
        () => profiles.find((profile) => profile.name === profileName),
        [profiles, profileName]
    );

    const selectedStandard = useMemo(() => {
        if (selectedProfile) {
            return getProfileTab(selectedProfile);
        }
        return standardTabs[0];
    }, [selectedProfile, standardTabs]);

    useEffect(() => {
        // URL profileName is not in the list of profiles from the response (e.g. scan config filter changed)
        // then jump to first list entry
        if (!selectedProfile && profiles[0]?.name) {
            handleToggleChange(profiles[0].name);
        }
    }, [selectedProfile, profiles, handleToggleChange]);

    function handleStandardSelection(standardShortName: string) {
        const firstProfileInStandard = profiles.find(
            (profile) => getProfileTab(profile) === standardShortName
        );

        if (firstProfileInStandard) {
            handleToggleChange(firstProfileInStandard.name);
        }
    }

    const filteredProfiles: ComplianceProfileSummary[] = useMemo(() => {
        return profiles.filter((profile) => getProfileTab(profile) === selectedStandard);
    }, [profiles, selectedStandard]);

    return (
        <>
            <PageSection type="tabs">
                <Tabs
                    activeKey={selectedStandard ?? ''}
                    onSelect={(_e, key) => {
                        handleStandardSelection(String(key));
                    }}
                    isBox
                >
                    {standardTabs.map((standardShortName) => (
                        <Tab
                            key={standardShortName}
                            eventKey={standardShortName}
                            title={standardShortName}
                            tabContentId={tabContentId}
                        />
                    ))}
                </Tabs>
            </PageSection>
            <PageSection>
                <ToggleGroup id={tabContentId} aria-label="Toggle for selected profile view">
                    {filteredProfiles.map(({ name }) => (
                        <ToggleGroupItem
                            key={name}
                            text={name}
                            buttonId="compliance-profiles-toggle-group"
                            isSelected={profileName === name}
                            onChange={() => handleToggleChange(name)}
                        />
                    ))}
                </ToggleGroup>
            </PageSection>
        </>
    );
}

export default ProfilesToggleGroup;
