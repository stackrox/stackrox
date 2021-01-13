import React, { ReactElement, ReactNode, useState } from 'react';

const tableRowClassName = 'relative border-b';
const baseTableRowClassName = `${tableRowClassName} border-base-300 bg-base-100`;
const alertTableRowClassName = `${tableRowClassName} border-alert-300 bg-alert-200 text-alert-800`;

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
            <div className="-translate-x-full transform translate-y-1 whitespace-no-wrap z-0">
                {children}
            </div>
        </td>
    );
}

export type TableRowProps = {
    colorType: 'alert' | null;
    row: {
        isGrouped: boolean;
    };
    children: ReactNode;
    HoveredRowComponent?: ReactNode;
    HoveredGroupedRowComponent?: ReactNode;
    GroupedRowComponent?: ReactNode;
};

function TableRow({
    colorType,
    row,
    children,
    HoveredRowComponent = null,
    HoveredGroupedRowComponent = null,
    GroupedRowComponent = null,
}: TableRowProps): ReactElement {
    const [isHovered, setIsHovered] = useState(false);

    const className = colorType === 'alert' ? alertTableRowClassName : baseTableRowClassName;
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
            className={className}
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

export default TableRow;
