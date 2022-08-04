import React, { ReactElement, ReactNode } from 'react';

export type FormCardProps = {
    dataTestId?: string | null;
    headerText: string;
    headerComponents?: ReactNode;
    children: ReactNode;
    showInnerPadding?: boolean;
};

type FormSectionChildProps = {
    children: ReactNode;
};

export function FormSectionFooter({ children }: FormSectionChildProps): ReactElement {
    return <div className="p-3 border-t border-base-300">{children}</div>;
}

export function FormSectionBody({ children }: FormSectionChildProps): ReactElement {
    return <div className="p-3 h-full">{children}</div>;
}

export function FormSection({
    dataTestId = null,
    headerText,
    headerComponents = null,
    children,
}: FormCardProps): ReactElement {
    return (
        <div data-testid={dataTestId}>
            <div className="bg-base-100 border-base-200 shadow">
                {(headerText || headerComponents) && (
                    <div className="p-2 pb-2 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between leading-loose">
                        {headerText}
                        {headerComponents && (
                            <div className="header-control float-right">{headerComponents}</div>
                        )}
                    </div>
                )}
                {children}
            </div>
        </div>
    );
}
