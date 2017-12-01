import React, { Component } from 'react';

class Tabs extends Component {
    constructor(props) {
        super(props);

        var headers = props.header.split(',');

        this.state = {
            headers: headers,
            active: headers[0],
            children: props.children
        }

        this.tabClick = this.tabClick.bind(this);
    }

    getHeaders() {
        var active = this.state.active;
        var tabClick = this.tabClick;
        return this.state.headers.map(function (header, i) {
            var tabClass = (active === header) ? 'tab-active' : 'tab';
            return <div className={tabClass} key={header + '-' + i} onClick={() => tabClick(header)}>{header}</div>;
        });
    }

    tabClick(header) {
        this.setState({ active: header });
    }

    renderChildren() {
        return React.Children.map(this.state.children, child => {
            return React.cloneElement(child, { active: this.state.active});
        });
    }

    render() {
        return (
            <div className="flex flex-col flex-1">
                <div className="flex flex-row font-mono font-bold">
                    {this.getHeaders()}
                    <div className="flex flex-1 border-b border-gray-light"></div>
                </div>
                {this.renderChildren()}
            </div>
        );
    }
    
}

export default Tabs;
