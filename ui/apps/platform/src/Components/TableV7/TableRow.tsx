import React, { ReactElement, ReactNode, useState } from 'react';

import { TableColorStyles } from './tableTypes';

function onFocus(): number {
    return 0;
}

export type TableRowOverlayProps = {
    isLayeredFirst?: boolean;
    children: ReactNode;
};

function TableRowOverlay({ isLayeredFirst = false, children }: TableRowOverlayProps): ReactElement {
    return (
        <td className={`flex overflow-visible w-0 sticky top-8 ${isLayeredFirst ? 'z-1' : ''}`}>
            <div className="-translate-x-full transform translate-y-1 whitespace-nowrap z-0">
                {children}
            </div>
        </td>
    );
}

export type TableRowProps = {
    colorStyles: TableColorStyles;
    row: {
        isGrouped: boolean;
    };
    children: ReactNode;
    HoveredRowComponent?: ReactNode;
    HoveredGroupedRowComponent?: ReactNode;
    GroupedRowComponent?: ReactNode;
};

export function TableRow({
    colorStyles,
    row,
    children,
    HoveredRowComponent = null,
    HoveredGroupedRowComponent = null,
    GroupedRowComponent = null,
}: TableRowProps): ReactElement {
    const [isHovered, setIsHovered] = useState(false);

    const { bgColor, borderColor, textColor } = colorStyles;

    const tableRowClassName = `relative border-b ${bgColor} ${borderColor} ${textColor}`;
    const showGroupedRowComponent = row.isGrouped;
    const showHoveredRowComponent = !showGroupedRowComponent && isHovered;
    const showHoveredGroupedRowComponent = showGroupedRowComponent && isHovered;

    const hoveredRowComponent = showHoveredRowComponent && HoveredRowComponent && (
        <TableRowOverlay>{HoveredRowComponent}</TableRowOverlay>
    );

    const groupedRowComponent = showGroupedRowComponent && GroupedRowComponent && (
        <TableRowOverlay isLayeredFirst>{GroupedRowComponent}</TableRowOverlay>
    );

    const hoveredGroupedRowComponent = showHoveredGroupedRowComponent &&
        HoveredGroupedRowComponent && (
            <TableRowOverlay>{HoveredGroupedRowComponent}</TableRowOverlay>
        );

    function onMouseEnter(): void {
        setIsHovered(true);
    }

    function onMouseLeave(): void {
        setIsHovered(false);
    }

    return (
        <tr
            className={tableRowClassName}
            onMouseEnter={onMouseEnter}
            onMouseLeave={onMouseLeave}
            onFocus={onFocus}
            data-testid={Array.isArray(children) ? 'data-row' : 'subhead-row'}
        >
            {children}
            {hoveredRowComponent}
            {groupedRowComponent}
            {hoveredGroupedRowComponent}
        </tr>
    );
}
