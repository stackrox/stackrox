import { useEffect } from 'react';
import { PageSection, Tab, Tabs, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import type { ComplianceProfileSummary } from 'services/ComplianceCommon';

import {
    getDefaultProfile,
    getProfileTab,
    getProfilesByTab,
    getTabsFromProfiles,
} from './ProfilesToggleGroup.utils';
import type { ProfileTab } from './ProfilesToggleGroup.utils';

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
    const profileTabs = getTabsFromProfiles(profiles);
    const selectedProfile = profiles.find((profile) => profile.name === profileName);
    const activeTab = selectedProfile ? getProfileTab(selectedProfile) : profileTabs[0];

    useEffect(() => {
        // URL profileName is not in the list (e.g. scan config filter changed): first tab + first profile by name
        if (!selectedProfile) {
            const fallback = getDefaultProfile(profiles);
            if (fallback) {
                handleToggleChange(fallback.name);
            }
        }
    }, [selectedProfile, profiles, handleToggleChange]);

    function handleTabSelection(tab: ProfileTab) {
        const first = getProfilesByTab(profiles, tab)[0];
        if (first) {
            handleToggleChange(first.name);
        }
    }

    const profilesByTab = getProfilesByTab(profiles, activeTab);

    return (
        <>
            <PageSection type="tabs">
                <Tabs
                    activeKey={activeTab ?? ''}
                    onSelect={(_e, key) => {
                        handleTabSelection(String(key));
                    }}
                    isBox
                >
                    {profileTabs.map((standardShortName) => (
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
                    {profilesByTab.map(({ name }) => (
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
