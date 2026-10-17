import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import PreviewViolations from './PreviewViolations';

describe('PreviewViolations', () => {
    it('shows the default empty preview message', () => {
        render(<PreviewViolations alertsFromDryRun={[]} />);

        expect(screen.getByText('No deployments have violations.')).toBeInTheDocument();
    });

    it('shows a custom empty preview message', () => {
        render(
            <PreviewViolations
                alertsFromDryRun={[]}
                noViolationsMessage="No deployments have Build or Deploy type violations."
            />
        );

        expect(
            screen.getByText('No deployments have Build or Deploy type violations.')
        ).toBeInTheDocument();
    });

    it('shows dry run violations grouped by deployment', () => {
        render(
            <PreviewViolations
                alertsFromDryRun={[{ deployment: 'central', violations: ['Violation one'] }]}
            />
        );

        expect(screen.getByRole('heading', { name: 'Deployment results' })).toBeInTheDocument();
        expect(screen.getByRole('heading', { name: 'central' })).toBeInTheDocument();
        expect(screen.getByText('Violation one')).toBeInTheDocument();
    });
});
