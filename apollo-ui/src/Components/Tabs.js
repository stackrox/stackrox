import React, { Component } from 'react';

class Tabs extends Component {
    constructor(props) {
        super(props);

        this.state = {
            active: props.headers[0]
        }

        this.tabClick = this.tabClick.bind(this);
    }

    getHeaders() {
        var active = this.state.active;
        var tabClick = this.tabClick;
        return this.props.headers.map(function (header, i) {
            var tabClass = (active === header) ? 'tab tab-active mt-2' : 'tab mt-2';
            if (header.disabled) tabClass = "tab disabled mt-2";
            return <button className={tabClass} key={header + '-' + i} onClick={() => tabClick(header)}>{header.text}</button>;
        });
    }

    tabClick(header) {
        if(header.disabled) return;
        this.setState({ active: header });
    }

    renderChildren() {
        return React.Children.map(this.props.children, child => {
            return React.cloneElement(child, { active: this.state.active});
        });
    }

    render() {
        return (
            <div className="w-full bg-white flex flex-col">
                <div className={`flex shadow-underline font-bold mb-3 bg-primary-100 pl-3 ${ this.props.className }`}>
                    {this.getHeaders()}
                </div>
                <div className="overflow-hidden h-full flex-1">{this.renderChildren()}</div>
            </div>
        );
    }
    
}

export default Tabs;
