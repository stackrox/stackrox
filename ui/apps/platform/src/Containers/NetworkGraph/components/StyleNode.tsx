/* eslint-disable @typescript-eslint/no-unsafe-return */
import * as React from 'react';
import {
    Decorator,
    DEFAULT_DECORATOR_RADIUS,
    DEFAULT_DECORATOR_PADDING,
    DEFAULT_LAYER,
    DefaultNode,
    getDefaultShapeDecoratorCenter,
    Layer,
    Node,
    NodeShape,
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
import { PficonNetworkRangeIcon } from '@patternfly/react-icons';
import useDetailsLevel from '@patternfly/react-topology/dist/esm/hooks/useDetailsLevel';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';
import { ensureExhaustive } from 'utils/type.utils';
import { NetworkPolicyState, DeploymentData, NodeDataType } from '../types/topology.type';

const ICON_PADDING = 20;
const CUSTOM_DECORATOR_PADDING = 2.5;

type StyleNodeProps = {
    element: Node;
    getCustomShape?: (node: Node) => React.FunctionComponent<React.PropsWithChildren<ShapeProps>>;
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

const getTypeIcon = (type?: NodeDataType): React.ComponentClass<SVGIconProps> => {
    switch (type) {
        case 'EXTERNAL_ENTITIES':
        case 'CIDR_BLOCK':
            return PficonNetworkRangeIcon;

        default:
            return DefaultIcon;
    }
};

const renderIcon = (data: { type?: NodeDataType }, element: Node): React.ReactNode => {
    const { width, height } = element.getDimensions();
    const shape = element.getNodeShape();
    const iconSize =
        (shape === NodeShape.trapezoid ? width : Math.min(width, height)) -
        (shape === NodeShape.stadium ? 5 : ICON_PADDING) * 2;
    const Component = getTypeIcon(data.type);

    return (
        <g transform={`translate(${(width - iconSize) / 2}, ${(height - iconSize) / 2})`}>
            <Component style={{ color: '#393F44' }} width={iconSize} height={iconSize} />
        </g>
    );
};

function getPolicyStateIcon(policyState: NetworkPolicyState) {
    switch (policyState) {
        case 'both':
            return <BothPolicyRules width="22px" height="22px" />;
        case 'egress':
            return <EgressOnly width="22px" height="22px" />;
        case 'ingress':
            return <IngressOnly width="22px" height="22px" />;
        case 'none':
            return <NoPolicyRules width="22px" height="22px" />;
        default:
            return ensureExhaustive(policyState);
    }
}

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
    const padding =
        quadrant === TopologyQuadrant.lowerLeft
            ? DEFAULT_DECORATOR_PADDING - CUSTOM_DECORATOR_PADDING
            : DEFAULT_DECORATOR_PADDING;

    return (
        <Decorator
            x={x}
            y={y}
            radius={DEFAULT_DECORATOR_RADIUS}
            showBackground
            icon={icon}
            padding={padding}
        />
    );
};

const renderDecorators = (
    element: Node,
    data: DeploymentData,
    getShapeDecoratorCenter?: (
        quadrant: TopologyQuadrant,
        node: Node
    ) => {
        x: number;
        y: number;
    }
): React.ReactNode => {
    const { showPolicyState, networkPolicyState, showExternalState, isExternallyConnected } = data;
    return (
        <>
            {showExternalState &&
                isExternallyConnected &&
                renderDecorator(
                    element,
                    TopologyQuadrant.upperRight,
                    <PficonNetworkRangeIcon />,
                    getShapeDecoratorCenter
                )}
            {showPolicyState &&
                renderDecorator(
                    element,
                    TopologyQuadrant.lowerLeft,
                    getPolicyStateIcon(networkPolicyState),
                    getShapeDecoratorCenter
                )}
        </>
    );
};

const StyleNode: React.FunctionComponent<React.PropsWithChildren<StyleNodeProps>> = ({
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
        if (detailsLevel === ScaleDetailsLevel.low && onHideCreateConnector) {
            onHideCreateConnector();
        }
    }, [detailsLevel, onHideCreateConnector]);

    const LabelIcon = passedData.labelIcon;

    // @TODO: If multiple classes need to be stringed together, then we need a more systematic way to generate those here
    const className = `${passedData?.isFadedOut ? 'pf-topology-node-faded' : ''}`;

    return (
        <Layer id={hover ? TOP_LAYER : DEFAULT_LAYER}>
            <g ref={hoverRef as React.LegacyRef<SVGGElement>}>
                <DefaultNode
                    className={className}
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
