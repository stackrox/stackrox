import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { PagerDots, PagerButtonGroup } from './PagerControls';

class Widget extends Component {
    static propTypes = {
        header: PropTypes.string,
        bodyClassName: PropTypes.string,
        className: PropTypes.string,
        children: PropTypes.node.isRequired,
        headerComponents: PropTypes.element,
        pages: PropTypes.number,
        onPageChange: PropTypes.func,
        id: PropTypes.string
    };

    static defaultProps = {
        header: '',
        bodyClassName: '',
        className: 'w-full bg-base-100',
        headerComponents: null,
        pages: 0,
        onPageChange: null,
        id: 'widget'
    };

    constructor(props) {
        super(props);
        this.state = {
            currentPage: 0
        };
    }

    changePage = pageNum => {
        this.setState({ currentPage: pageNum });
        if (this.props.onPageChange) this.props.onPageChange(pageNum);
    };

    handlePageNext = () => {
        const targetPage = this.state.currentPage + 1;
        if (targetPage >= this.props.pages) return;

        this.changePage(targetPage);
    };

    handlePagePrev = () => {
        const targetPage = this.state.currentPage - 1;
        if (targetPage < 0) return;

        this.changePage(targetPage);
    };

    handleSetPage = page => {
        if (page < 0 || page >= this.props.pages) return;

        this.setState({
            currentPage: page
        });
    };

    render() {
        const { children, pages, header, headerComponents, bodyClassName, className } = this.props;
        const { currentPage } = this.state;

        let pagerControls;
        if (pages > 1) {
            pagerControls = {
                arrows: (
                    <PagerButtonGroup
                        onPageNext={this.handlePageNext}
                        onPagePrev={this.handlePagePrev}
                        isPrev={currentPage - 1 >= 0}
                        isNext={currentPage + 1 < pages}
                    />
                ),
                dots: (
                    <PagerDots
                        onPageChange={this.handleSetPage}
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
                    {React.Children.map(children, child =>
                        React.cloneElement(child, { currentPage })
                    )}
                </React.Fragment>
            ) : (
                children
            );

        return (
            <div
                className={`flex flex-col shadow rounded relative rounded bg-base-100 h-full ${className}`}
                data-test-id={this.props.id}
            >
                <div className="border-b border-base-300">
                    <div className="flex w-full h-10 word-break">
                        <div
                            className="flex flex-1 text-base-600 pt-1 uppercase items-center tracking-wide px-3 leading-normal font-700"
                            data-test-id="widget-header"
                        >
                            <div className="flex-grow">{header}</div>
                            {pagerControls ? pagerControls.arrows : null}
                        </div>
                        {headerComponents && (
                            <div className="flex items-center pr-3 relative">
                                {headerComponents}
                            </div>
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
}

export default Widget;
