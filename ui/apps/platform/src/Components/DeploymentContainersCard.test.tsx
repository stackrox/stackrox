import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MemoryRouter } from 'react-router-dom';

import type { Container } from 'types/deployment.proto';
import DeploymentContainersCard from './DeploymentContainersCard';

import useFeatureFlags from 'hooks/useFeatureFlags';

vi.mock('hooks/useFeatureFlags', () => ({
    default: vi.fn(),
}));

const mockedUseFeatureFlags = vi.mocked(useFeatureFlags);

function mockContainer(overrides: Partial<Container> & Pick<Container, 'id' | 'name'>): Container {
    return {
        config: {
            env: [],
            command: [],
            args: [],
            directory: '',
            user: '',
            uid: '',
            appArmorProfile: '',
        },
        image: {
            id: '',
            name: { fullName: '', registry: '', remote: '', tag: '' },
            notPullable: false,
        },
        securityContext: {
            privileged: false,
            selinux: null,
            dropCapabilities: [],
            addCapabilities: [],
            readOnlyRootFilesystem: false,
            seccompProfile: null,
            allowPrivilegeEscalation: false,
        },
        volumes: [],
        ports: [],
        secrets: [],
        resources: { cpuCoresRequest: 0, cpuCoresLimit: 0, memoryMbRequest: 0, memoryMbLimit: 0 },
        type: 'REGULAR',
        ...overrides,
    };
}

const containers: Container[] = [
    mockContainer({ id: '1', name: 'app', type: 'REGULAR' }),
    mockContainer({ id: '2', name: 'sidecar', type: 'REGULAR' }),
    mockContainer({ id: '3', name: 'init-db', type: 'INIT' }),
];

const getImageUrl = (id: string) => `/images/${id}`;

describe('DeploymentContainersCard', () => {
    it('groups init containers separately when feature flag is enabled', () => {
        mockedUseFeatureFlags.mockReturnValue({
            isFeatureFlagEnabled: (flag) => flag === 'ROX_INIT_CONTAINER_SUPPORT',
            isLoadingFeatureFlags: false,
            error: undefined,
        });

        render(
            <MemoryRouter>
                <DeploymentContainersCard
                    containers={containers}
                    title="Container configuration"
                    getImageUrl={getImageUrl}
                />
            </MemoryRouter>
        );

        expect(screen.getByText('Container configuration')).toBeInTheDocument();
        expect(screen.getByText('Init container configuration')).toBeInTheDocument();
        expect(screen.getByText('app')).toBeInTheDocument();
        expect(screen.getByText('sidecar')).toBeInTheDocument();
        expect(screen.getByText('init-db')).toBeInTheDocument();
        expect(screen.getByText('Init')).toBeInTheDocument();
    });

    it('shows all containers together when feature flag is disabled', () => {
        mockedUseFeatureFlags.mockReturnValue({
            isFeatureFlagEnabled: () => false,
            isLoadingFeatureFlags: false,
            error: undefined,
        });

        render(
            <MemoryRouter>
                <DeploymentContainersCard
                    containers={containers}
                    title="Container configuration"
                    getImageUrl={getImageUrl}
                />
            </MemoryRouter>
        );

        expect(screen.getByText('Container configuration')).toBeInTheDocument();
        expect(screen.queryByText('Init container configuration')).not.toBeInTheDocument();
        expect(screen.getByText('app')).toBeInTheDocument();
        expect(screen.getByText('sidecar')).toBeInTheDocument();
        expect(screen.getByText('init-db')).toBeInTheDocument();
    });
});
