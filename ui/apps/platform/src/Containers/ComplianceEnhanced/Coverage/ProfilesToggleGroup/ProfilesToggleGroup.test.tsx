import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';

import type { ComplianceBenchmark, ComplianceProfileSummary } from 'services/ComplianceCommon';

import ProfilesToggleGroup from './ProfilesToggleGroup';

function createComplianceStandard(shortName: string): ComplianceBenchmark {
    return { name: '', version: '', description: '', provider: '', shortName };
}

function createProfile(
    name: string,
    partial: Partial<ComplianceProfileSummary> = {}
): ComplianceProfileSummary {
    return {
        name,
        productType: 'Platform',
        description: '',
        title: '',
        ruleCount: 0,
        profileVersion: '',
        standards: [],
        ...partial,
    };
}

describe('ProfilesToggleGroup', () => {
    test('toggle group lists only profiles for the active tab', () => {
        const profiles = [
            createProfile('profile-under-cis-one', {
                standards: [createComplianceStandard('CIS')],
            }),
            createProfile('profile-under-cis-two', {
                standards: [createComplianceStandard('CIS')],
            }),
            createProfile('profile-under-nist-one', {
                standards: [createComplianceStandard('NIST')],
            }),
        ];
        render(
            <ProfilesToggleGroup
                profileName="profile-under-cis-one"
                profiles={profiles}
                handleToggleChange={vi.fn()}
            />
        );
        expect(screen.getByRole('button', { name: 'profile-under-cis-one' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'profile-under-cis-two' })).toBeInTheDocument();
        expect(
            screen.queryByRole('button', { name: 'profile-under-nist-one' })
        ).not.toBeInTheDocument();
    });

    test('selecting a tab navigates to the first profile in that tab', async () => {
        const user = userEvent.setup();
        const handleToggleChange = vi.fn();
        const profiles = [
            createProfile('profile-standard-nist-first', {
                standards: [createComplianceStandard('NIST')],
            }),
            createProfile('profile-standard-nist-second', {
                standards: [createComplianceStandard('NIST')],
            }),
            createProfile('profile-standard-cis-selected', {
                standards: [createComplianceStandard('CIS')],
            }),
        ];
        render(
            <ProfilesToggleGroup
                profileName="profile-standard-cis-selected"
                profiles={profiles}
                handleToggleChange={handleToggleChange}
            />
        );

        await user.click(screen.getByRole('tab', { name: 'NIST' }));

        expect(handleToggleChange).toHaveBeenCalledWith('profile-standard-nist-first');
    });

    test('selecting another profile in the toggle group calls handleToggleChange', async () => {
        const user = userEvent.setup();
        const handleToggleChange = vi.fn();
        const profiles = [
            createProfile('profile-shared-pci-one', {
                standards: [createComplianceStandard('PCI')],
            }),
            createProfile('profile-shared-pci-two', {
                standards: [createComplianceStandard('PCI')],
            }),
        ];
        render(
            <ProfilesToggleGroup
                profileName="profile-shared-pci-one"
                profiles={profiles}
                handleToggleChange={handleToggleChange}
            />
        );

        await user.click(screen.getByRole('button', { name: 'profile-shared-pci-two' }));

        expect(handleToggleChange).toHaveBeenCalledWith('profile-shared-pci-two');
    });

    // e.g. user filters by scan schedule: the URL still names a profile that is not in that
    // schedule's filtered list, so profileName is stale relative to `profiles`.
    test('when profileName is missing from profiles, falls back to first profile in the list', async () => {
        const handleToggleChange = vi.fn();
        const profiles = [
            createProfile('profile-list-first', {
                standards: [createComplianceStandard('HIPAA')],
            }),
            createProfile('profile-list-second', {
                standards: [createComplianceStandard('HIPAA')],
            }),
        ];
        render(
            <ProfilesToggleGroup
                profileName="stale-url-profile-name"
                profiles={profiles}
                handleToggleChange={handleToggleChange}
            />
        );

        await waitFor(() => {
            expect(handleToggleChange).toHaveBeenCalledWith('profile-list-first');
        });
    });
});
