import React from 'react';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import * as DeploymentsService from 'services/DeploymentsService';
import * as ProcessListeningOnPortsService from 'services/ProcessListeningOnPortsService';
import ListeningEndpointsPage from './ListeningEndpointsPage';

const setup = () => {
    const user = userEvent.setup();
    const utils = renderWithRouter(<ListeningEndpointsPage />);

    return { user, utils };
};

/**
 * Mocks the DeploymentsService.listDeployments and ListeningEndpointsService.getListeningEndpointsForDeployment
 * methods to return the provided deployments instead of making a real API call.
 *
 * @param deployments The deployments to return. A `Partial<Deployment>[]` is sufficient.
 */
function mockDeploymentsWithListeningEndpoints(deployments) {
    jest.spyOn(DeploymentsService, 'listDeployments').mockImplementation((_1, _2, page, count) => {
        const offset = page * count;
        return Promise.resolve(deployments.slice(offset, offset + count));
    });

    jest.spyOn(
        ProcessListeningOnPortsService,
        'getListeningEndpointsForDeployment'
    ).mockImplementation((deploymentId) => ({
        request: Promise.resolve(
            deployments.find((d) => d.id === deploymentId)?.listeningEndpoints ?? []
        ),
        cancel: jest.fn(),
    }));
}

describe('ListeningEndpointsPage', () => {
    it('should render a default message when no deployments are found', async () => {
        setup();

        mockDeploymentsWithListeningEndpoints([]);

        expect(
            await screen.findByText('No deployments with listening endpoints found')
        ).toBeInTheDocument();
    });

    it('should not render deployments without listening endpoints', async () => {
        setup();

        mockDeploymentsWithListeningEndpoints([
            {
                id: 'd-1',
                name: 'deployment-1',
                listeningEndpoints: [],
            },
            {
                id: 'd-2',
                name: 'deployment-2',
                listeningEndpoints: [
                    { id: '1', endpoint: { port: 80, protocol: 'TCP' }, signal: {} },
                ],
            },
        ]);

        expect(await screen.findByText('deployment-2')).toBeInTheDocument();
        expect(screen.queryByText('deployment-1')).not.toBeInTheDocument();
    });

    it('should render a view more button to allow loading more deployments', async () => {
        const { user } = setup();

        // Generate 15 mock deployments with names `deployment-1` through `deployment-15`
        mockDeploymentsWithListeningEndpoints(
            Array.from({ length: 15 }).map((_, i) => ({
                id: `d-${i + 1}`,
                name: `deployment-${i + 1}`,
                listeningEndpoints: [
                    { id: `${i + 1}`, endpoint: { port: 80, protocol: 'TCP' }, signal: {} },
                ],
            }))
        );

        // Check that only 1-10 are loaded
        expect(await screen.findByText('deployment-1')).toBeInTheDocument();
        expect(screen.getByText('deployment-10')).toBeInTheDocument();
        expect(screen.queryByText('deployment-11')).not.toBeInTheDocument();

        // Check that view more button is present
        const viewMoreButton = screen.getByRole('button', { name: 'View more' });
        expect(viewMoreButton).toBeInTheDocument();
        // Click it
        await user.click(viewMoreButton);

        // Expect the button to be disabled during loading
        expect(viewMoreButton).toBeDisabled();

        // Check that 11-15 are loaded
        expect(await screen.findByText('deployment-11')).toBeInTheDocument();
        expect(screen.getByText('deployment-15')).toBeInTheDocument();
        // Check that the originals are still there too
        expect(screen.getByText('deployment-1')).toBeInTheDocument();
        expect(screen.getByText('deployment-10')).toBeInTheDocument();

        // Since all results are loaded, the button should be gone
        expect(screen.queryByRole('button', { name: 'View more' })).not.toBeInTheDocument();
    });
});
