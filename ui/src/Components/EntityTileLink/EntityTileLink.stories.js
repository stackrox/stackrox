import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import entityTypes from 'constants/entityTypes';

import EntityTileLink from './EntityTileLink';

export default {
    title: 'EntityTileLink',
    component: EntityTileLink
};

export const withData = () => {
    return (
        <MemoryRouter>
            <EntityTileLink count={10} entityType={entityTypes.POLICY} url="/url/to/somewhere" />
        </MemoryRouter>
    );
};

export const withPositionFirst = () => {
    return (
        <MemoryRouter>
            <EntityTileLink
                count={10}
                entityType={entityTypes.POLICY}
                url="/url/to/somewhere"
                position="first"
            />
        </MemoryRouter>
    );
};

export const withPositionMiddle = () => {
    return (
        <MemoryRouter>
            <EntityTileLink
                count={10}
                entityType={entityTypes.POLICY}
                url="/url/to/somewhere"
                position="middle"
            />
        </MemoryRouter>
    );
};

export const withPositionLast = () => {
    return (
        <MemoryRouter>
            <EntityTileLink
                count={10}
                entityType={entityTypes.POLICY}
                url="/url/to/somewhere"
                position="last"
            />
        </MemoryRouter>
    );
};
