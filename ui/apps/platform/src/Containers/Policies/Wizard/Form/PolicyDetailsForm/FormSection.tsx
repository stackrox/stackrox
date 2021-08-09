import React, { ReactElement, ReactNode } from 'react';

export type FormCardProps = {
    dataTestId?: string | null;
    headerText: string;
    headerComponents: ReactNode;
    children: ReactNode;
    showInnerPadding?: boolean;
};

type FormSectionChildProps = {
    children: ReactNode;
};

export function FormSectionFooter({ children }: FormSectionChildProps): ReactElement {
    return <div>{children}</div>;
}

export function FormSectionBody({ children }: FormSectionChildProps): ReactElement {
    return <div className="p-3 h-full">{children}</div>;
}

export function FormSection({
    dataTestId = null,
    headerText,
    headerComponents,
    children,
}: FormCardProps): ReactElement {
    return (
        <div className="px-3 pt-5" data-testid={dataTestId}>
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
