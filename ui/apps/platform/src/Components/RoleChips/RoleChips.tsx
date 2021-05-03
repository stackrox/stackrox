import React, { ReactElement } from 'react';
import { Flex } from '@patternfly/react-core';

export type RoleChipsProps = {
    roleNames: string[];
};

function RoleChips({ roleNames }: RoleChipsProps): ReactElement {
    if (roleNames.length === 0) {
        return <span>No roles</span>;
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {roleNames.map((roleName) => (
                <div className="pf-c-chip" key={roleName}>
                    <span className="pf-c-chip__text">{roleName}</span>
                </div>
            ))}
        </Flex>
    );
}

export default RoleChips;
