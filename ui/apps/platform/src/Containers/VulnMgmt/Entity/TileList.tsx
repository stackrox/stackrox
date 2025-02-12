import React, { ReactElement } from 'react';

import TileLink from 'Components/TileLink';
import { VulnerabilityManagementEntityType } from 'utils/entityRelationships';

/*
 * OrdinaryCase for consistency with entityCountNounOrdinaryCase
 * for table links
 * for table heading
 */
import { entityNounOrdinaryCase } from '../entitiesForVulnerabilityManagement';

export type TileListItem = {
    count: number;
    entityType: VulnerabilityManagementEntityType;
    url: string;
};

export type TileListProps = {
    items: TileListItem[];
    title: string;
};

function TileList({ items, title }: TileListProps): ReactElement {
    return (
        <div className="text-base-600 rounded border mx-2 my-3 bg-primary-200 border-primary-400">
            <h3 className="border-b text-base-600 text-center p-1 leading-normal font-700 border-base-400">
                {title}
            </h3>
            <ul className="pb-2">
                {items.map(({ count, entityType, url }) => (
                    <li className="p-2 pb-0" key={entityType}>
                        <TileLink
                            colorClasses="border-primary-400 hover:bg-primary-200 rounded"
                            superText={count}
                            text={entityNounOrdinaryCase(count, entityType)}
                            url={url}
                        />
                    </li>
                ))}
            </ul>
        </div>
    );
}

export default TileList;
