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
        <Breadcrumb style={{ marginBottom: '20px' }}>
            {items.map((item, index) => {
                const isCurrentPage = !item.path || index === items.length - 1;
                const truncatedLabel =
                    item.label.length > 60 ? `${item.label.substring(0, 57)}...` : item.label;

                return (
                    <BreadcrumbItem
                        key={`${item.label}-${index}`}
                        render={({ className }) =>
                            isCurrentPage ? (
                                <span className={className} title={item.label}>
                                    {truncatedLabel}
                                </span>
                            ) : (
                                <Link to={item.path!} className={className} title={item.label}>
                                    {truncatedLabel}
                                </Link>
                            )
                        }
                    />
                );
            })}
        </Breadcrumb>
    );
}
