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
            <div className="flex">
                <TileLink text="10 POLICIES" url="/url/to/somewhere" />
            </div>
        </MemoryRouter>
    );
};

export const withSuperText = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink superText="10" text="POLICIES" url="/url/to/somewhere" />
            </div>
        </MemoryRouter>
    );
};

export const withSubText = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink text="10 POLICIES" url="/url/to/somewhere" subText="(0 failing)" />
            </div>
        </MemoryRouter>
    );
};

export const withError = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink
                    text="10 POLICIES"
                    url="/url/to/somewhere"
                    subText="(15 failing)"
                    isError
                />
            </div>
        </MemoryRouter>
    );
};

export const withLoading = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink text="10 CVES" url="/url/to/somewhere" subText="(15 fixable)" loading />
            </div>
        </MemoryRouter>
    );
};

export const withIcon = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink
                    text="65%"
                    url="/url/to/somewhere"
                    subText="Image Health"
                    icon={<TrendingUp className="text-primary-500 h-4 w-4" />}
                />
            </div>
        </MemoryRouter>
    );
};

export const withPositionFirst = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink text="10 POLICIES" url="/url/to/somewhere" position="first" />
            </div>
        </MemoryRouter>
    );
};

export const withPositionMiddle = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink text="10 POLICIES" url="/url/to/somewhere" position="middle" />
            </div>
        </MemoryRouter>
    );
};

export const withPositionLast = () => {
    return (
        <MemoryRouter>
            <div className="flex">
                <TileLink text="10 POLICIES" url="/url/to/somewhere" position="last" />
            </div>
        </MemoryRouter>
    );
};
