import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import pluralize from 'pluralize';

import TileList from './TileList';

export default {
    title: 'TileList',
    component: TileList
};

export const basicTileList = () => {
    const tiles = [
        {
            count: 15,
            label: pluralize('Deployment', 15)
        },
        {
            count: 3,
            label: pluralize('Image', 3)
        },
        {
            count: 23,
            label: pluralize('Component', 23)
        }
    ];

    return (
        <MemoryRouter>
            <div className="flex">
                <TileList items={tiles} />
            </div>
        </MemoryRouter>
    );
};

export const withTitle = () => {
    const matches = [
        {
            count: 26,
            label: pluralize('Deployment', 26)
        }
    ];

    const contains = [
        {
            count: 54,
            label: pluralize('CVE', 54)
        },
        {
            count: 8,
            label: pluralize('Component', 8)
        }
    ];

    return (
        <MemoryRouter>
            <div className="flex">
                <div className="flex flex-col">
                    <TileList items={matches} title="Matches" />
                    <TileList items={contains} title="Contains" />
                </div>
            </div>
        </MemoryRouter>
    );
};
