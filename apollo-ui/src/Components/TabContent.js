import React, { Component } from 'react';

class TabContent extends Component {
    render() {
        return (
            <div className={(this.props.active === this.props.name) ? 'flex flex-col h-full transition' : 'hidden'}>
                {this.props.children}
            </div>
        );
    }
}

export default TabContent;
