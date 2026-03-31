import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom-v5-compat';

import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';

import DeploymentStatusFilter from './DeploymentStatusFilter';

async function renderWithRouter(ui: React.ReactElement, initialEntry = '') {
    let renderResult: ReturnType<typeof render>;
    await actAndFlushTaskQueue(() => {
        renderResult = render(
            <MemoryRouter initialEntries={[initialEntry]}>{ui}</MemoryRouter>
        );
    });
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    return renderResult!;
}

describe('DeploymentStatusFilter', () => {
    it('renders both Deployed and Deleted options', async () => {
        await renderWithRouter(<DeploymentStatusFilter />);
        expect(screen.getByText('Deployed')).toBeInTheDocument();
        expect(screen.getByText('Deleted')).toBeInTheDocument();
    });

    it('selects Deployed by default', async () => {
        await renderWithRouter(<DeploymentStatusFilter />);
        // PatternFly ToggleGroupItem marks the selected item with aria-pressed="true".
        expect(screen.getByText('Deployed').closest('button')).toHaveAttribute(
            'aria-pressed',
            'true'
        );
        expect(screen.getByText('Deleted').closest('button')).toHaveAttribute(
            'aria-pressed',
            'false'
        );
    });

    it('selects Deleted when deploymentStatus=DELETED is in the URL', async () => {
        await renderWithRouter(<DeploymentStatusFilter />, '?deploymentStatus=DELETED');
        expect(screen.getByText('Deleted').closest('button')).toHaveAttribute(
            'aria-pressed',
            'true'
        );
    });

    it('calls onChange when a different option is selected', async () => {
        const onChange = vi.fn();
        await renderWithRouter(<DeploymentStatusFilter onChange={onChange} />);
        await userEvent.click(screen.getByText('Deleted'));
        expect(onChange).toHaveBeenCalledTimes(1);
    });
});
