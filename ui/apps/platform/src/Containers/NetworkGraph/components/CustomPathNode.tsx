import * as React from 'react';
import { observer } from 'mobx-react';
import {
    WithCreateConnectorProps,
    Node,
    WithContextMenuProps,
    WithDragNodeProps,
    WithSelectionProps,
    WithDndDragProps,
    WithDndDropProps,
} from '@patternfly/react-topology';
import Path from './shapes/Path';
import DemoDefaultNode from './DemoDefaultNode';

type CustomPathNodeProps = {
    element: Node;
    droppable?: boolean;
    canDrop?: boolean;
} & WithSelectionProps &
    WithDragNodeProps &
    WithDndDragProps &
    WithDndDropProps &
    WithCreateConnectorProps &
    WithContextMenuProps;

const CustomPathNode: React.FunctionComponent<CustomPathNodeProps> = (props) => (
    <DemoDefaultNode getCustomShape={() => Path} {...props} />
);

export default observer(CustomPathNode);
