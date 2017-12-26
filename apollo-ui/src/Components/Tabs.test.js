import React from 'react';
import { shallow } from 'enzyme';
import Tabs from 'Components/Tabs';

describe('Component:Tabs', () => {
    it('shows the first tab as the active tab', () => {
        const tabs = shallow(<Tabs headers={[{ text: 'Tab 1', disabled: false }, { text: 'Tab 2', disabled: false }]} />);
        expect(tabs.state('activeIndex')).toEqual(0);
    });

    it('should be able to switch active tabs', () => {
        const tabs = shallow(<Tabs headers={[{ text: 'Tab 1', disabled: false }, { text: 'Tab 2', disabled: false }, { text: 'Tab 3', disabled: false }]} />);
        expect(tabs.state('activeIndex')).toEqual(0);
        let button = tabs.findWhere(n => n.key() === 'Tab 2');
        button.simulate('click');
        expect(tabs.state('activeIndex')).toEqual(1);
        button = tabs.findWhere(n => n.key() === 'Tab 3');
        button.simulate('click');
        expect(tabs.state('activeIndex')).toEqual(2);
    });

    it('should not be able to switch to a disabled tab', () => {
        const tabs = shallow(<Tabs headers={[{ text: 'Tab 1', disabled: false }, { text: 'Tab 2', disabled: true }, { text: 'Tab 3', disabled: false }]} />);
        expect(tabs.state('activeIndex')).toEqual(0);
        let button = tabs.findWhere(n => n.key() === 'Tab 2');
        button.simulate('click');
        expect(tabs.state('activeIndex')).toEqual(0);
        button = tabs.findWhere(n => n.key() === 'Tab 3');
        button.simulate('click');
        expect(tabs.state('activeIndex')).toEqual(2);
    });
});
