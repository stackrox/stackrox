import React, { ReactElement, ReactNode } from 'react';

type PopoverBodyContentProps =
    | {
          bodyContent: ReactNode;
          footerContent: ReactNode;
      }
    | {
          headerContent: ReactNode;
          headerIcon?: React.ReactNode;
          bodyContent: ReactNode;
      }
    | {
          headerContent: ReactNode;
          headerIcon?: React.ReactNode;
          bodyContent: ReactNode;
          footerContent: ReactNode;
      };

// Compose footer, or header, or both for bodyContent prop of Popover element.
// To prevent accessibility issues, render div and p elements (with PatternFly classes).
// For more information: no-Popover-footerContent-headerContent-props lint rule.
function PopoverBodyContent(props: PopoverBodyContentProps): ReactElement {
    return (
        <>
            {'headerContent' in props && props.headerContent && (
                <div className="pf-v5-c-popover__header">
                    <div className="pf-v5-c-popover__title">
                        {props.headerIcon && (
                            <span className="pf-v5-c-popover__title-icon">{props.headerIcon}</span>
                        )}
                        <p className="pf-v5-c-popover__title-text">{props.headerContent}</p>
                    </div>
                </div>
            )}
            <div className="bodyContent">{props.bodyContent}</div>
            {'footerContent' in props && props.footerContent && (
                <div className="pf-v5-c-popover__footer">{props.footerContent}</div>
            )}
        </>
    );
}

export default PopoverBodyContent;
