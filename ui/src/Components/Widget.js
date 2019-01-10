import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { PagerDots, PagerButtonGroup } from './PagerControls';

class Widget extends Component {
    static propTypes = {
        header: PropTypes.string,
        bodyClassName: PropTypes.string,
        className: PropTypes.string,
        children: PropTypes.node.isRequired,
        headerComponents: PropTypes.element,
        pages: PropTypes.number,
        onPageChange: PropTypes.func
    };

    static defaultProps = {
        header: '',
        bodyClassName: null,
        className: 'w-full',
        headerComponents: null,
        pages: 0,
        onPageChange: null
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
                    />
                ),
                dots: (
                    <PagerDots
                        onPageChange={this.handleSetPage}
                        pageCount={pages}
                        currentPage={currentPage}
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
                className={`flex flex-col h-full border border-base-400 relative rounded ${className}`}
                data-test-id="widget"
            >
                <div className="border-b border-base-400">
                    <div className="flex w-full h-10 word-break">
                        <div
                            className="flex flex-1 text-base-600 uppercase items-center tracking-wide pl-2 pr-2 leading-normal font-700"
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
                <div className={`flex h-full overflow-y-auto ${bodyClassName}`}>
                    {childrenWithPageProp}
                </div>
                {pagerControls ? pagerControls.dots : null}
            </div>
        );
    }
}

export default Widget;
