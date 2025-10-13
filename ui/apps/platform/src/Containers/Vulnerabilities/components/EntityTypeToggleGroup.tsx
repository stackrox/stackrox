import React from 'react';
import { ToggleGroup, ToggleGroupItem, pluralize } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { NonEmptyArray } from 'utils/type.utils';
import { EntityTab } from '../types';

type EntityTypeToggleGroupProps<EntityTabType extends EntityTab> = {
    className?: string;
    entityTabs: Readonly<NonEmptyArray<EntityTabType>>;
    entityCounts: Record<EntityTabType, number>;
    onChange: (entityTab: EntityTabType) => void;
};

export function EntityTypeToggleGroup<EntityTabType extends EntityTab>({
    className = '',
    entityTabs,
    entityCounts,
    onChange,
}: EntityTypeToggleGroupProps<EntityTabType>) {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion('entityTab', entityTabs);

    function handleEntityTabChange(entityTab: EntityTabType) {
        setActiveEntityTabKey(entityTab);
        onChange(entityTab);
    }

    return (
        <ToggleGroup className={className} aria-label="Entity type toggle items">
            {entityTabs.map((tab) => (
                <ToggleGroupItem
                    key={tab}
                    text={`${pluralize(entityCounts[tab], tab)}`}
                    buttonId={tab}
                    isSelected={activeEntityTabKey === tab}
                    onChange={() => handleEntityTabChange(tab)}
                />
            ))}
        </ToggleGroup>
    );
}

export default EntityTypeToggleGroup;
