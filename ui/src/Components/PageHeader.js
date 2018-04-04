import React, { Component } from 'react';
import PropTypes from 'prop-types';

class PageHeader extends Component {
    static propTypes = {
        header: PropTypes.string.isRequired,
        isViewFiltered: PropTypes.bool,
        children: PropTypes.element
    };

    static defaultProps = {
        isViewFiltered: false,
        children: null
    };

    renderFilteredViewText = () => (
        <div className="text-primary-400 mt-2 font-400 italic">
            {this.props.isViewFiltered ? 'Filtered view' : 'Default view'}
        </div>
    );

    render() {
        return (
            <div className="flex flex-row bg-white py-3 px-4 border-b border-primary-300">
                <div className="w-48 self-center">
                    <div className="text-base-600 uppercase text-lg tracking-wide">
                        {this.props.header}
                    </div>
                    {this.renderFilteredViewText()}
                </div>
                <div className="w-full">{this.props.children}</div>
            </div>
        );
    }
}

export default PageHeader;
