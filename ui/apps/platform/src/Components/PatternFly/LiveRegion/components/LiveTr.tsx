import React from 'react';
import { Tr, TrProps } from '@patternfly/react-table';

import LiveTd from './LiveTd';

export type LiveTrProps = Omit<TrProps, 'ref'>;

function LiveTr({ children, ...props }: LiveTrProps) {
    return (
        <Tr {...props}>
            {React.Children.map(children, (child) => {
                if (React.isValidElement(child)) {
                    return React.createElement(LiveTd, { ...child.props });
                }
                return child;
            })}
        </Tr>
    );
}

export default LiveTr;
