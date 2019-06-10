import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { PagerDots, PagerButtonGroup } from './PagerControls';

function Widget({
    header,
    bodyClassName,
    className,
    children,
    headerComponents,
    pages,
    onPageChange,
    id
}) {
    const [currentPage, setCurrentPage] = useState(0);

    function changePage(pageNum) {
        setCurrentPage(pageNum);
        if (onPageChange) onPageChange(pageNum);
    }

    function handlePageNext() {
        const targetPage = currentPage + 1;
        if (targetPage >= pages) return;

        changePage(targetPage);
    }

    function handlePagePrev() {
        const targetPage = currentPage - 1;
        if (targetPage < 0) return;
        changePage(targetPage);
    }

    function handleSetPage(page) {
        if (page < 0 || page >= pages) return;
        setCurrentPage(page);
    }

    let pagerControls;
    if (pages > 1) {
        pagerControls = {
            arrows: (
                <PagerButtonGroup
                    onPageNext={handlePageNext}
                    onPagePrev={handlePagePrev}
                    isPrev={currentPage - 1 >= 0}
                    isNext={currentPage + 1 < pages}
                />
            ),
            dots: (
                <PagerDots
                    onPageChange={handleSetPage}
                    pageCount={pages}
                    currentPage={currentPage}
                    className="hidden"
                />
            )
        };
    }

    const childrenWithPageProp =
        pages && pages > 1 ? (
            <React.Fragment>
                {React.Children.map(children, child => React.cloneElement(child, { currentPage }))}
            </React.Fragment>
        ) : (
            children
        );

    return (
        <div
            className={`flex flex-col shadow rounded relative rounded bg-base-100 ${className}`}
            data-test-id={id}
        >
            <div className="border-b border-base-300">
                <div className="flex w-full h-10 word-break">
                    <div
                        className="flex flex-1 text-sm text-base-600 pt-1 uppercase items-center tracking-wide px-3 leading-normal font-700"
                        data-test-id="widget-header"
                    >
                        <div className="flex-grow">{header}</div>
                        {pagerControls ? pagerControls.arrows : null}
                    </div>
                    {headerComponents && (
                        <div className="flex items-center pr-3 relative">{headerComponents}</div>
                    )}
                </div>
            </div>
            <div className={`flex h-full ${bodyClassName}`}>
                <ErrorBoundary>{childrenWithPageProp}</ErrorBoundary>
            </div>
            {pagerControls ? pagerControls.dots : null}
        </div>
    );
}

Widget.propTypes = {
    header: PropTypes.string,
    bodyClassName: PropTypes.string,
    className: PropTypes.string,
    children: PropTypes.node.isRequired,
    headerComponents: PropTypes.element,
    pages: PropTypes.number,
    onPageChange: PropTypes.func,
    id: PropTypes.string
};

Widget.defaultProps = {
    header: '',
    bodyClassName: '',
    className: 'w-full bg-base-100',
    headerComponents: null,
    pages: 0,
    onPageChange: null,
    id: 'widget'
};

export default Widget;
