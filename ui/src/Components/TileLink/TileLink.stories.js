import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import { TrendingUp } from 'react-feather';
import TileLink from './TileLink';

export default {
    title: 'TileLink',
    component: TileLink
};

export const withData = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" />
        </MemoryRouter>
    );
};

export const withSubText = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" subText="(0 failing)" />
        </MemoryRouter>
    );
};

export const withError = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" subText="(15 failing)" isError />
        </MemoryRouter>
    );
};

export const withLoading = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 CVES" url="/url/to/somewhere" subText="(15 fixable)" loading />
        </MemoryRouter>
    );
};

export const withIcon = () => {
    return (
        <MemoryRouter>
            <TileLink
                text="65%"
                url="/url/to/somewhere"
                subText="Image Health"
                icon={<TrendingUp className="text-primary-500 h-4 w-4" />}
            />
        </MemoryRouter>
    );
};

export const withPositionFirst = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" position="first" />
        </MemoryRouter>
    );
};

export const withPositionMiddle = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" position="middle" />
        </MemoryRouter>
    );
};

export const withPositionLast = () => {
    return (
        <MemoryRouter>
            <TileLink text="10 POLICIES" url="/url/to/somewhere" position="last" />
        </MemoryRouter>
    );
};
