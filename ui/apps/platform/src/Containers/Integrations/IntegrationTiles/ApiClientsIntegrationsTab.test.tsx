import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom-v5-compat';
import { vi } from 'vitest';

import ApiClientsIntegrationsTab from './ApiClientsIntegrationsTab';

vi.mock('./IntegrationsTabPage', () => ({
    default: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

describe('ApiClientsIntegrationsTab', () => {
    const sourcesEnabled = [
        'imageIntegrations',
        'notifiers',
        'cloudSources',
        'apiClients',
        'authProviders',
    ] as const;

    test('renders the ServiceNow external integration tile', () => {
        render(
            <MemoryRouter initialEntries={['/main/integrations/apiClients']}>
                <ApiClientsIntegrationsTab sourcesEnabled={[...sourcesEnabled]} />
            </MemoryRouter>
        );

        expect(screen.getByText('ServiceNow')).toBeInTheDocument();
    });

    test('renders a link to the ServiceNow store', () => {
        render(
            <MemoryRouter initialEntries={['/main/integrations/apiClients']}>
                <ApiClientsIntegrationsTab sourcesEnabled={[...sourcesEnabled]} />
            </MemoryRouter>
        );

        const link = screen.getByRole('link', {
            name: 'Open ServiceNow in a new tab',
        });
        expect(link).toHaveAttribute('href');
        expect(link).toHaveAttribute('href', expect.stringContaining('store.servicenow.com'));
    });
});
