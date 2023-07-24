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
        mockDeploymentsWithListeningEndpoints([]);

        setup();

        expect(
            await screen.findByText('No deployments with listening endpoints found')
        ).toBeInTheDocument();
    });
});
