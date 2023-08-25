import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ErrorBoundary from 'Containers/ErrorBoundary'; // @TODO Move ErrorBoundary to Components directory
import { PagerDots, PagerButtonGroup } from 'Components/PagerControls';

function Widget({
    header,
    bodyClassName,
    className,
    children,
    headerComponents,
    pages,
    onPageChange,
    id,
    titleComponents,
}) {
    const [currentPage, setCurrentPage] = useState(0);

    function changePage(pageNum) {
        setCurrentPage(pageNum);
        if (onPageChange) {
            onPageChange(pageNum);
        }
    }

    function handlePageNext() {
        const targetPage = currentPage + 1;
        if (targetPage >= pages) {
            return;
        }

        changePage(targetPage);
    }

    function handlePagePrev() {
        const targetPage = currentPage - 1;
        if (targetPage < 0) {
            return;
        }
        changePage(targetPage);
    }

    function handleSetPage(page) {
        if (page < 0 || page >= pages) {
            return;
        }
        setCurrentPage(page);
    }

    let pagerControls;
    if (pages > 1) {
        pagerControls = {
            arrows: (
                <PagerButtonGroup
                    onPageNext={handlePageNext}
                    onPagePrev={handlePagePrev}
                    enablePrev={currentPage - 1 >= 0}
                    enableNext={currentPage + 1 < pages}
                />
            ),
            dots: (
                <PagerDots
                    onPageChange={handleSetPage}
                    pageCount={pages}
                    currentPage={currentPage}
                    className="hidden"
                />
            ),
        };
    }

    const childrenWithPageProp =
        pages && pages > 1 ? (
            <>
                {React.Children.map(children, (child) =>
                    React.cloneElement(child, { currentPage })
                )}
            </>
        ) : (
            children
        );
    const headerContent = titleComponents || <div className="line-clamp">{header}</div>;
    return (
        <div
            className={`widget flex flex-col shadow rounded relative rounded bg-base-100 ${className}`}
            data-testid={id}
        >
            <div className="border-b border-base-300">
                <div className="flex flex-auto min-h-10 word-break">
                    <div
                        className="flex flex-auto text-base-600 items-center px-2 leading-normal font-700"
                        data-testid="widget-header"
                    >
                        <div className="w-full">{headerContent}</div>
                        {pagerControls ? pagerControls.arrows : null}
                    </div>
                    {headerComponents && (
                        <div className="flex justify-end items-center pr-2 relative border-l border-base-300 pl-2">
                            {headerComponents}
                        </div>
                    )}
                </div>
            </div>
            <div className={`flex h-full ${bodyClassName}`} data-testid="widget-body">
                <ErrorBoundary>{childrenWithPageProp}</ErrorBoundary>
            </div>
            {pagerControls ? pagerControls.dots : null}
        </div>
    );
}

Widget.propTypes = {
    header: PropTypes.string,
    titleComponents: PropTypes.node,
    bodyClassName: PropTypes.string,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    headerComponents: PropTypes.element,
    pages: PropTypes.number,
    onPageChange: PropTypes.func,
    id: PropTypes.string,
};

Widget.defaultProps = {
    header: '',
    titleComponents: null,
    bodyClassName: '',
    className: 'w-full bg-base-100',
    headerComponents: null,
    pages: 0,
    onPageChange: null,
    id: 'widget',
};

export default Widget;
