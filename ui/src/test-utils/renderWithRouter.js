import React from 'react';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
// eslint-disable-next-line
import { render } from '@testing-library/react';

function renderWithRouter(
    ui,
    { route = '/', history = createMemoryHistory({ initialEntries: [route] }) } = {}
) {
    return {
        ...render(<Router history={history}>{ui}</Router>),
        // adding `history` to the returned utilities to allow us
        // to reference it in our tests (just try to avoid using
        // this to test implementation details).
        history
    };
}

export default renderWithRouter;
