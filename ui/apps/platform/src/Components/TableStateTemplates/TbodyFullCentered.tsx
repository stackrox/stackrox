import React from 'react';
import { Bullseye } from '@patternfly/react-core';
import { Tbody, Tr, Td } from '@patternfly/react-table';

export type TbodyFullCenteredProps = {
    colSpan: number;
    children: React.ReactNode;
};

export function TbodyFullCentered({ colSpan, children }: TbodyFullCenteredProps) {
    return (
        <Tbody>
            <Tr>
                <Td colSpan={colSpan}>
                    <Bullseye className="pf-v5-u-my-2xl">{children}</Bullseye>
                </Td>
            </Tr>
        </Tbody>
    );
}
