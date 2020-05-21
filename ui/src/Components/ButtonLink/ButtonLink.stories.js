import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { RefreshCcw } from 'react-feather';

import ButtonLink from './ButtonLink';

export default {
    title: 'ButtonLink',
    component: ButtonLink,
};

export const withText = () => (
    <MemoryRouter>
        <ButtonLink linkTo="/">View All</ButtonLink>
    </MemoryRouter>
);

export const withTextAndIcon = () => (
    <MemoryRouter>
        <ButtonLink linkTo="/" icon={<RefreshCcw size="14" className="mx-1 lg:ml-1 lg:mr-3" />}>
            Scan
        </ButtonLink>
    </MemoryRouter>
);
