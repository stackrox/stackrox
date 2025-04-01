import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { render } from '@testing-library/react';

function renderWithRouter(ui, { route = '/' } = {}) {
    return {
        ...render(<MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>),
    };
}

export default renderWithRouter;
