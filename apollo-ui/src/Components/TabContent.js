import React, { Component } from 'react';

class TabContent extends Component {
    constructor(props) {
        super(props);

        this.state = {
            name: props.name,
            active: props.active,
            children: props.children
        }
    }

    componentWillReceiveProps(nextProps) {
        this.setState({ name: nextProps.name, children: nextProps.children, active: nextProps.active });
    }

    render() {
        return (
            <div className={(this.state.active === this.state.name) ? 'flex flex-col p-2' : 'hidden p-2'}>
                {this.state.children}
            </div>
        );
    }
    
}

export default TabContent;
