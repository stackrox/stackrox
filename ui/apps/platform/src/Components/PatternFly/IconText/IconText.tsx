import React, { ReactElement } from 'react';

export type IconTextProps = {
    icon: ReactElement;
    text: string;
    isTextOnly?: boolean;
};

function IconText({ icon, text, isTextOnly }: IconTextProps): ReactElement {
    if (isTextOnly) {
        // Wrap text in fragment so function returns ReactElement,
        // because return ReactNode can cause error in caller:
        // TS2786 IconText cannot be used as a JSX component.
        return <>{text}</>; // Export as PDF
    }

    // Display flex because classic styles have display block for svg element.
    // Align center because PatternFly guideline.
    return (
        <span className="pf-u-display-inline-flex pf-u-align-items-center">
            {icon}
            <span className="pf-u-pl-sm pf-u-text-nowrap">{text}</span>
        </span>
    );
}

export default IconText;
