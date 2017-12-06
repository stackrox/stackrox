import React, { Component } from 'react';

class TabContent extends Component {
    render() {
        return (
            <div className={(this.props.active === this.props.name) ? 'flex flex-col p-2' : 'hidden p-2'}>
                {this.props.children}
            </div>
        );
    }
}

export default TabContent;
