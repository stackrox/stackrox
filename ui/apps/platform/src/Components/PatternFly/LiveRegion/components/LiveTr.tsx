import React from 'react';
import { Tr, TrProps } from '@patternfly/react-table';

import LiveTd from './LiveTd';

function isTdComponent(child: React.ReactNode): child is React.ReactElement {
    return (
        React.isValidElement(child) &&
        typeof child.type !== 'string' &&
        'displayName' in child.type &&
        child.type.displayName === 'Td'
    );
}

export type LiveTrProps = Omit<TrProps, 'ref'>;

function LiveTr({ children, ...props }: LiveTrProps) {
    return (
        <Tr {...props}>
            {React.Children.map(children, (child) => {
                // If the child is a Td component, replace it in with a LiveTd component that
                // provides its own Td
                // otherwise return the child as is
                if (isTdComponent(child)) {
                    return React.createElement(LiveTd, { ...child.props });
                }
                return child;
            })}
        </Tr>
    );
}

export default LiveTr;
