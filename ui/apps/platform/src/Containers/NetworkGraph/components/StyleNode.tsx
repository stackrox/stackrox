/* eslint-disable @typescript-eslint/no-unused-expressions */
/* eslint-disable @typescript-eslint/no-unsafe-return */
import * as React from 'react';
import {
    Decorator,
    DEFAULT_DECORATOR_RADIUS,
    DEFAULT_LAYER,
    DefaultNode,
    getDefaultShapeDecoratorCenter,
    Layer,
    Node,
    NodeShape,
    NodeStatus,
    observer,
    ScaleDetailsLevel,
    ShapeProps,
    TOP_LAYER,
    TopologyQuadrant,
    useHover,
    WithContextMenuProps,
    WithCreateConnectorProps,
    WithDragNodeProps,
    WithSelectionProps,
} from '@patternfly/react-topology';
import DefaultIcon from '@patternfly/react-icons/dist/esm/icons/builder-image-icon';
import AlternateIcon from '@patternfly/react-icons/dist/esm/icons/regions-icon';
import FolderOpenIcon from '@patternfly/react-icons/dist/esm/icons/folder-open-icon';
import BlueprintIcon from '@patternfly/react-icons/dist/esm/icons/blueprint-icon';
import PauseCircle from '@patternfly/react-icons/dist/esm/icons/pause-circle-icon';
import useDetailsLevel from '@patternfly/react-topology/dist/esm/hooks/useDetailsLevel';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export enum DataTypes {
    Default,
    Alternate,
}
const ICON_PADDING = 20;

type StyleNodeProps = {
    element: Node;
    getCustomShape?: (node: Node) => React.FunctionComponent<ShapeProps>;
    getShapeDecoratorCenter?: (quadrant: TopologyQuadrant, node: Node) => { x: number; y: number };
    showLabel?: boolean; // Defaults to true
    labelIcon?: React.ComponentClass<SVGIconProps>;
    showStatusDecorator?: boolean; // Defaults to false
    regrouping?: boolean;
    dragging?: boolean;
} & WithContextMenuProps &
    WithCreateConnectorProps &
    WithDragNodeProps &
    WithSelectionProps;

const getTypeIcon = (dataType?: DataTypes): any => {
    switch (dataType) {
        case DataTypes.Alternate:
            return AlternateIcon;
        default:
            return DefaultIcon;
    }
};

const renderIcon = (data: { dataType?: DataTypes }, element: Node): React.ReactNode => {
    const { width, height } = element.getDimensions();
    const shape = element.getNodeShape();
    const iconSize =
        (shape === NodeShape.trapezoid ? width : Math.min(width, height)) -
        (shape === NodeShape.stadium ? 5 : ICON_PADDING) * 2;
    const Component = getTypeIcon(data.dataType);

    return (
        <g transform={`translate(${(width - iconSize) / 2}, ${(height - iconSize) / 2})`}>
            <Component style={{ color: '#393F44' }} width={iconSize} height={iconSize} />
        </g>
    );
};

const renderDecorator = (
    element: Node,
    quadrant: TopologyQuadrant,
    icon: React.ReactNode,
    getShapeDecoratorCenter?: (
        quadrant: TopologyQuadrant,
        node: Node,
        radius?: number
    ) => {
        x: number;
        y: number;
    }
): React.ReactNode => {
    const { x, y } = getShapeDecoratorCenter
        ? getShapeDecoratorCenter(quadrant, element)
        : getDefaultShapeDecoratorCenter(quadrant, element);

    return <Decorator x={x} y={y} radius={DEFAULT_DECORATOR_RADIUS} showBackground icon={icon} />;
};

const renderDecorators = (
    element: Node,
    data: { showDecorators?: boolean },
    getShapeDecoratorCenter?: (
        quadrant: TopologyQuadrant,
        node: Node
    ) => {
        x: number;
        y: number;
    }
): React.ReactNode => {
    if (!data.showDecorators) {
        return null;
    }
    const nodeStatus = element.getNodeStatus();
    return (
        <>
            {!nodeStatus || nodeStatus === NodeStatus.default
                ? renderDecorator(
                      element,
                      TopologyQuadrant.upperLeft,
                      <FolderOpenIcon />,
                      getShapeDecoratorCenter
                  )
                : null}
            {renderDecorator(
                element,
                TopologyQuadrant.upperRight,
                <BlueprintIcon />,
                getShapeDecoratorCenter
            )}
            {renderDecorator(
                element,
                TopologyQuadrant.lowerLeft,
                <PauseCircle />,
                getShapeDecoratorCenter
            )}
        </>
    );
};

const StyleNode: React.FunctionComponent<StyleNodeProps> = ({
    element,
    onContextMenu,
    contextMenuOpen,
    showLabel,
    dragging,
    regrouping,
    onShowCreateConnector,
    onHideCreateConnector,
    ...rest
}) => {
    const data = element.getData();
    const detailsLevel = useDetailsLevel();
    const [hover, hoverRef] = useHover();

    const passedData = React.useMemo(() => {
        const newData = { ...data };
        Object.keys(newData).forEach((key) => {
            if (newData[key] === undefined) {
                delete newData[key];
            }
        });
        return newData;
    }, [data]);

    React.useEffect(() => {
        if (detailsLevel === ScaleDetailsLevel.low) {
            onHideCreateConnector && onHideCreateConnector();
        }
    }, [detailsLevel, onHideCreateConnector]);

    const LabelIcon = passedData.labelIcon;
    return (
        <Layer id={hover ? TOP_LAYER : DEFAULT_LAYER}>
            <g ref={hoverRef as React.LegacyRef<SVGGElement>}>
                <DefaultNode
                    element={element}
                    scaleLabel={detailsLevel !== ScaleDetailsLevel.high}
                    scaleNode={hover && detailsLevel === ScaleDetailsLevel.low}
                    {...rest}
                    {...passedData}
                    dragging={dragging}
                    regrouping={regrouping}
                    showLabel={hover || (detailsLevel === ScaleDetailsLevel.high && showLabel)}
                    showStatusBackground={!hover && detailsLevel === ScaleDetailsLevel.low}
                    showStatusDecorator={
                        detailsLevel === ScaleDetailsLevel.high && passedData.showStatusDecorator
                    }
                    onContextMenu={data.showContextMenu ? onContextMenu : undefined}
                    contextMenuOpen={contextMenuOpen}
                    onShowCreateConnector={
                        detailsLevel !== ScaleDetailsLevel.low ? onShowCreateConnector : undefined
                    }
                    onHideCreateConnector={onHideCreateConnector}
                    labelIcon={LabelIcon && <LabelIcon noVerticalAlign />}
                    attachments={
                        (hover || detailsLevel === ScaleDetailsLevel.high) &&
                        renderDecorators(element, passedData, rest.getShapeDecoratorCenter)
                    }
                >
                    {(hover || detailsLevel !== ScaleDetailsLevel.low) &&
                        renderIcon(passedData, element)}
                </DefaultNode>
            </g>
        </Layer>
    );
};

export default observer(StyleNode);
