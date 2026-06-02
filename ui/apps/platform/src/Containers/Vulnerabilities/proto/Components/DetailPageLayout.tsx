import React, { ReactNode } from 'react';
import { BreadcrumbNav, BreadcrumbNavItem } from './BreadcrumbNav';
import { Title } from '@patternfly/react-core';

interface DetailPageLayoutProps {
    breadcrumbs: BreadcrumbNavItem[];
    title: string;
    subtitle?: string;
    actions?: ReactNode;
    children: ReactNode;
}

export function DetailPageLayout({
    breadcrumbs,
    title,
    subtitle,
    actions,
    children,
}: DetailPageLayoutProps): JSX.Element {
    return (
        <div>
            <BreadcrumbNav items={breadcrumbs} />

            <div
                style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '16px',
                }}
            >
                <div>
                    <Title headingLevel="h1">{title}</Title>
                    {subtitle && (
                        <p style={{ color: 'var(--pf-global--Color--200)', marginTop: '4px' }}>
                            {subtitle}
                        </p>
                    )}
                </div>
                {actions && <div>{actions}</div>}
            </div>

            {children}
        </div>
    );
}
