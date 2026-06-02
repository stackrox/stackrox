import React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

export interface BreadcrumbNavItem {
    label: string;
    path?: string; // If not provided, item is treated as active/current page
}

interface BreadcrumbNavProps {
    items: BreadcrumbNavItem[];
}

export function BreadcrumbNav({ items }: BreadcrumbNavProps): JSX.Element {
    return (
        <Breadcrumb>
            {items.map((item, index) => {
                const isActive = !item.path || index === items.length - 1;
                return (
                    <BreadcrumbItem key={`${item.label}-${index}`} isActive={isActive}>
                        {isActive ? (
                            item.label
                        ) : (
                            <Link to={item.path!}>{item.label}</Link>
                        )}
                    </BreadcrumbItem>
                );
            })}
        </Breadcrumb>
    );
}
