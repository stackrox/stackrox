/* eslint-disable @typescript-eslint/no-unsafe-return */
import React, { useMemo } from 'react';
import type { ComponentClass, FunctionComponent, PropsWithChildren, ReactNode } from 'react';
import { observer, ScaleDetailsLevel } from '@patternfly/react-topology';
import type {
    Node,
    ShapeProps,
    WithDragNodeProps,
    WithSelectionProps,
} from '@patternfly/react-topology';
import AlternateIcon from '@patternfly/react-icons/dist/esm/icons/regions-icon';
import DefaultIcon from '@patternfly/react-icons/dist/esm/icons/builder-image-icon';
import useDetailsLevel from '@patternfly/react-topology/dist/esm/hooks/useDetailsLevel';
import type { SVGIconProps } from '@patternfly/react-icons/dist/js/createIcon';
import DefaultFakeGroup from './DefaultFakeGroup';

const ICON_PADDING = 20;

export enum DataTypes {
    Default,
    Alternate,
}

type StyleGroupProps = {
    element: Node;
    collapsedWidth?: number;
    collapsedHeight?: number;
    onCollapseChange?: (group: Node, collapsed: boolean) => void;
    getCollapsedShape?: (node: Node) => FunctionComponent<PropsWithChildren<ShapeProps>>;
    collapsedShadowOffset?: number; // defaults to 10
} & WithDragNodeProps &
    WithSelectionProps;

const StyleFakeGroup: FunctionComponent<PropsWithChildren<StyleGroupProps>> = ({
    element,
    collapsedWidth = 75,
    collapsedHeight = 75,
    ...rest
}) => {
    const data = element.getData();
    const detailsLevel = useDetailsLevel();

    const getTypeIcon = (dataType?: DataTypes): ComponentClass<SVGIconProps> => {
        switch (dataType) {
            case DataTypes.Alternate:
                return AlternateIcon;
            default:
                return DefaultIcon;
        }
    };

    const renderIcon = (): ReactNode => {
        const iconSize = Math.min(collapsedWidth, collapsedHeight) - ICON_PADDING * 2;
        const Component = getTypeIcon(data.dataType);

        return (
            <g
                transform={`translate(${(collapsedWidth - iconSize) / 2}, ${
                    (collapsedHeight - iconSize) / 2
                })`}
            >
                <Component style={{ color: '#393F44' }} width={iconSize} height={iconSize} />
            </g>
        );
    };

    const passedData = useMemo(() => {
        const newData = { ...data };
        Object.keys(newData).forEach((key) => {
            if (newData[key] === undefined) {
                delete newData[key];
            }
        });
        return newData;
    }, [data]);

    return (
        <DefaultFakeGroup
            element={element}
            collapsedWidth={collapsedWidth}
            collapsedHeight={collapsedHeight}
            showLabel={detailsLevel === ScaleDetailsLevel.high}
            {...rest}
            {...passedData}
        >
            {renderIcon()}
        </DefaultFakeGroup>
    );
};

export default observer(StyleFakeGroup);
