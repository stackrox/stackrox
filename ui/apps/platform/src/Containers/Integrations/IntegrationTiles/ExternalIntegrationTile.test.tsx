import type { SVGProps } from 'react';
import { render, screen } from '@testing-library/react';

import ExternalIntegrationTile from './ExternalIntegrationTile';

function MockLogo(props: SVGProps<SVGSVGElement>) {
    return <svg data-testid="mock-logo" {...props} />;
}

describe('ExternalIntegrationTile', () => {
    test('renders the integration label and logo', () => {
        render(
            <ExternalIntegrationTile Logo={MockLogo} label="ServiceNow" url="https://example.com" />
        );

        expect(screen.getByText('ServiceNow')).toBeInTheDocument();
        expect(screen.getByRole('img', { name: 'ServiceNow logo' })).toBeInTheDocument();
    });

    test('renders an external link pointing to the provided URL', () => {
        render(
            <ExternalIntegrationTile
                Logo={MockLogo}
                label="ServiceNow"
                url="https://store.servicenow.com/example"
            />
        );

        const link = screen.getByRole('link', {
            name: 'Open ServiceNow in a new tab',
        });
        expect(link).toHaveAttribute('href', 'https://store.servicenow.com/example');
        expect(link).toHaveAttribute('target', '_blank');
    });

    test('does not render a count badge', () => {
        render(
            <ExternalIntegrationTile Logo={MockLogo} label="ServiceNow" url="https://example.com" />
        );

        expect(screen.queryByText(/^\d+$/)).not.toBeInTheDocument();
    });
});
