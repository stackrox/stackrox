/* eslint-disable @typescript-eslint/no-unsafe-return */
import * as React from 'react';
import { observer } from 'mobx-react';
import {
    DefaultEdge,
    Edge,
    WithContextMenuProps,
    WithSelectionProps,
} from '@patternfly/react-topology';

type StyleEdgeProps = {
    element: Edge;
} & WithContextMenuProps &
    WithSelectionProps;

const StyleEdge: React.FunctionComponent<StyleEdgeProps> = ({
    element,
    onContextMenu,
    contextMenuOpen,
    ...rest
}) => {
    const data = element.getData();

    const passedData = React.useMemo(() => {
        const newData = { ...data };
        Object.keys(newData).forEach((key) => {
            if (newData[key] === undefined) {
                delete newData[key];
            }
        });
        return newData;
    }, [data]);

    return (
        <DefaultEdge
            element={element}
            {...rest}
            {...passedData}
            onContextMenu={data?.showContextMenu ? onContextMenu : undefined}
            contextMenuOpen={contextMenuOpen}
        />
    );
};

export default observer(StyleEdge);
